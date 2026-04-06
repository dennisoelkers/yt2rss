package feed

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dnhn/yt2rss/internal/db"
	"github.com/dnhn/yt2rss/internal/models"
	"github.com/dnhn/yt2rss/internal/youtube"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	db        *db.DB
	ytClient  *youtube.Client
	generator *Generator
	templates *template.Template
	baseURL   string
	cacheDir  string
	ytDlpBin  string
}

func New(db *db.DB, ytClient *youtube.Client, baseURL string, cacheDir string, ytDlpBin string) *Handler {
	templates := template.Must(template.ParseFiles(
		"internal/templates/index.html",
		"internal/templates/feed.html",
	))

	if cacheDir != "" {
		os.MkdirAll(cacheDir, 0755)
	}

	if ytDlpBin == "" {
		ytDlpBin = "yt-dlp"
	}

	return &Handler{
		db:        db,
		ytClient:  ytClient,
		generator: NewGenerator(baseURL),
		templates: templates,
		baseURL:   baseURL,
		cacheDir:  cacheDir,
		ytDlpBin:  ytDlpBin,
	}
}

func (h *Handler) RegisterRoutes(r *chi.Mux) {
	r.Get("/", h.handleIndex)
	r.Post("/", h.handleCreate)
	r.Get("/feed/{id}", h.handleFeedInfo)
	r.Get("/rss/{id}", h.handleRSS)
	r.Post("/feed/{id}/refresh", h.handleRefresh)
	r.Post("/feed/{id}/delete", h.handleDelete)
	r.Get("/stream/{feedID}/{videoID}", h.handleStream)
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	feeds, err := h.db.ListFeeds()
	if err != nil {
		http.Error(w, "Failed to load feeds", http.StatusInternalServerError)
		return
	}

	data := struct {
		Feeds []FeedView
		Error string
	}{
		Feeds: make([]FeedView, 0, len(feeds)),
		Error: r.URL.Query().Get("error"),
	}

	for _, f := range feeds {
		videos, _ := h.db.GetVideos(f.ID, 5)
		fv := FeedView{
			ID:         f.ID,
			Title:      f.Title,
			YoutubeURL: f.YoutubeURL,
			VideoCount: len(videos),
			CreatedAt:  f.CreatedAt,
			UpdatedAt:  f.UpdatedAt,
			RSSURL:     h.getBaseURL(r) + "/rss/" + f.ID,
		}
		data.Feeds = append(data.Feeds, fv)
	}

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/?error=invalid+form", http.StatusSeeOther)
		return
	}

	urlStr := strings.TrimSpace(r.PostFormValue("url"))
	if urlStr == "" {
		http.Redirect(w, r, "/?error=url+required", http.StatusSeeOther)
		return
	}

	if _, err := h.db.GetFeedByURL(urlStr); err == nil {
		http.Redirect(w, r, "/?error=feed+already+exists", http.StatusSeeOther)
		return
	}

	feedID := uuid.New().String()
	feed, videos, err := h.ytClient.FetchFeed(urlStr, feedID, 50)
	if err != nil {
		log.Printf("Failed to fetch feed: %v", err)
		http.Redirect(w, r, "/?error=fetch+failed:+"+err.Error(), http.StatusSeeOther)
		return
	}

	if err := h.db.CreateFeed(feed); err != nil {
		log.Printf("Failed to create feed: %v", err)
		http.Redirect(w, r, "/?error=db+error", http.StatusInternalServerError)
		return
	}

	for _, video := range videos {
		if err := h.db.UpsertVideo(&video); err != nil {
			log.Printf("Failed to insert video: %v", err)
		}
	}

	http.Redirect(w, r, "/feed/"+feed.ID, http.StatusSeeOther)
}

func (h *Handler) handleFeedInfo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	feed, err := h.db.GetFeed(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	videos, err := h.db.GetVideos(id, 100)
	if err != nil {
		http.Error(w, "Failed to load videos", http.StatusInternalServerError)
		return
	}

	data := struct {
		Feed   FeedView
		Videos []VideoView
	}{
		Feed: FeedView{
			ID:          feed.ID,
			Title:       feed.Title,
			YoutubeURL:  feed.YoutubeURL,
			Description: feed.Description,
			VideoCount:  len(videos),
			CreatedAt:   feed.CreatedAt,
			UpdatedAt:   feed.UpdatedAt,
			RSSURL:      h.getBaseURL(r) + "/rss/" + feed.ID,
		},
		Videos: make([]VideoView, 0, len(videos)),
	}

	for _, v := range videos {
		data.Videos = append(data.Videos, VideoView{
			Title:        v.Title,
			YoutubeID:    v.YoutubeID,
			ThumbnailURL: v.ThumbnailURL,
			PublishedAt:  v.PublishedAt,
			Duration:     formatDuration(v.Duration),
			YouTubeURL:   "https://www.youtube.com/watch?v=" + v.YoutubeID,
		})
	}

	h.templates.ExecuteTemplate(w, "feed.html", data)
}

func (h *Handler) handleRSS(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	feed, err := h.db.GetFeed(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	age := time.Since(feed.UpdatedAt)
	if age > 30*time.Minute {
		go h.refreshFeed(feed)
	}

	videos, err := h.db.GetVideos(id, 50)
	if err != nil {
		http.Error(w, "Failed to load videos", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")

	baseURL := h.getBaseURL(r)
	rss, err := h.generator.GenerateWithBaseURL(feed, videos, baseURL)
	if err != nil {
		http.Error(w, "Failed to generate RSS", http.StatusInternalServerError)
		return
	}

	w.Write(rss)
}

func (h *Handler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	feed, err := h.db.GetFeed(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if err := h.refreshFeed(feed); err != nil {
		log.Printf("Failed to refresh feed: %v", err)
		http.Redirect(w, r, "/feed/"+id+"?error=refresh+failed", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/feed/"+id, http.StatusSeeOther)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.db.DeleteFeed(id); err != nil {
		log.Printf("Failed to delete feed: %v", err)
		http.Redirect(w, r, "/?error=delete+failed", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) refreshFeed(feed *models.Feed) error {
	newFeed, videos, err := h.ytClient.FetchFeed(feed.YoutubeURL, feed.ID, 50)
	if err != nil {
		return err
	}

	feed.Title = newFeed.Title
	feed.Description = newFeed.Description
	if err := h.db.UpdateFeed(feed); err != nil {
		return err
	}

	for _, video := range videos {
		if err := h.db.UpsertVideo(&video); err != nil {
			log.Printf("Failed to upsert video: %v", err)
		}
	}

	return nil
}

type FeedView struct {
	ID          string
	Title       string
	Description string
	YoutubeURL  string
	VideoCount  int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	RSSURL      string
}

type VideoView struct {
	Title        string
	YoutubeID    string
	ThumbnailURL string
	PublishedAt  time.Time
	Duration     string
	YouTubeURL   string
}

func (h *Handler) getBaseURL(r *http.Request) string {
	if h.baseURL != "" && h.baseURL != "http://localhost:8080" {
		return h.baseURL
	}

	proto := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		proto = "https"
	}

	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}

	return proto + "://" + host
}

func (h *Handler) handleStream(w http.ResponseWriter, r *http.Request) {
	feedID := chi.URLParam(r, "feedID")
	videoID := chi.URLParam(r, "videoID")

	_, err := h.db.GetFeed(feedID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if h.cacheDir != "" {
		h.serveCachedStream(w, r, videoID)
	} else {
		h.streamVideo(w, videoID)
	}
}

func (h *Handler) serveCachedStream(w http.ResponseWriter, r *http.Request, videoID string) {
	cachePath := filepath.Join(h.cacheDir, videoID+".m4a")

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		log.Printf("Caching %s to %s", videoID, cachePath)
		if err := h.downloadAndCache(videoID, cachePath); err != nil {
			log.Printf("Cache download error: %v", err)
			http.Error(w, "Failed to download video", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "audio/mp4")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", h.getFileSize(cachePath)))
	w.Header().Set("Accept-Ranges", "bytes")

	http.ServeFile(w, r, cachePath)
}

func (h *Handler) getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func (h *Handler) downloadAndCache(videoID, cachePath string) error {
	videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)

	tmpPath := cachePath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer f.Close()
	defer os.Remove(tmpPath)

	cmd := exec.Command("sh", "-c",
		fmt.Sprintf("%s -q -f bestaudio --no-warnings -o - %s | ffmpeg -i - -c:a aac -b:a 128k -f mp4 -movflags frag_keyframe+default_base_moof pipe:1", h.ytDlpBin, videoURL))
	cmd.Stdout = f
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return err
	}

	return os.Rename(tmpPath, cachePath)
}

func (h *Handler) streamVideo(w http.ResponseWriter, videoID string) {
	videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)

	w.Header().Set("Content-Type", "audio/mp4")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.m4a\"", videoID))
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	cmd := exec.Command("sh", "-c",
		fmt.Sprintf("%s -q -f bestaudio --no-warnings -o - %s | ffmpeg -i - -c:a aac -b:a 128k -f mp4 -movflags frag_keyframe+default_base_moof pipe:1", h.ytDlpBin, videoURL))

	cmd.Stdout = w
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		log.Printf("Streaming error: %v", err)
	}
}

func (h *Handler) StreamVideoWithWriter(w io.Writer, videoID string) error {
	videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)

	cmd := exec.Command("sh", "-c",
		fmt.Sprintf("%s -q -f bestaudio --no-warnings -o - %s | ffmpeg -i - -c:a aac -b:a 128k -f mp4 -movflags frag_keyframe+default_base_moof pipe:1", h.ytDlpBin, videoURL))
	cmd.Stdout = w
	cmd.Stderr = nil

	return cmd.Run()
}
