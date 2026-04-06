package youtube

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/dnhn/yt2rss/internal/models"
	"github.com/google/uuid"
)

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

type ytdlpPlaylistInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Uploader    string `json:"uploader"`
	UploaderID  string `json:"uploader_id"`
	Entries     []struct {
		ID          string  `json:"id"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Uploader    string  `json:"uploader"`
		UploaderID  string  `json:"uploader_id"`
		Timestamp   int64   `json:"timestamp"`
		Thumbnail   string  `json:"thumbnail"`
		Duration    float64 `json:"duration"`
	} `json:"entries"`
}

type ytdlpChannelInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ID          string `json:"id"`
	Thumbnail   string `json:"thumbnail"`
	Uploader    string `json:"uploader"`
	UploaderID  string `json:"uploader_id"`
	Entries     []struct {
		ID          string  `json:"id"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Timestamp   int64   `json:"timestamp"`
		Thumbnail   string  `json:"thumbnail"`
		Duration    float64 `json:"duration"`
	} `json:"entries"`
}

func (c *Client) parseURL(inputURL string) (urlType string, id string, err error) {
	re := regexp.MustCompile(`youtu\.be/([a-zA-Z0-9_-]{11})`)
	if matches := re.FindStringSubmatch(inputURL); len(matches) > 1 {
		return "video", matches[1], nil
	}

	re = regexp.MustCompile(`[?&]v=([a-zA-Z0-9_-]{11})`)
	if matches := re.FindStringSubmatch(inputURL); len(matches) > 1 {
		return "video", matches[1], nil
	}

	if strings.Contains(inputURL, "/channel/") {
		re = regexp.MustCompile(`/channel/([^/?]+)`)
		if matches := re.FindStringSubmatch(inputURL); len(matches) > 1 {
			return "channel", matches[1], nil
		}
	}

	if strings.Contains(inputURL, "/@") {
		re = regexp.MustCompile(`/@([^/?]+)`)
		if matches := re.FindStringSubmatch(inputURL); len(matches) > 1 {
			return "handle", matches[1], nil
		}
	}

	if strings.Contains(inputURL, "/playlist") {
		re = regexp.MustCompile(`[?&]list=([^&]+)`)
		if matches := re.FindStringSubmatch(inputURL); len(matches) > 1 {
			return "playlist", matches[1], nil
		}
	}

	if strings.Contains(inputURL, "/shorts/") {
		re = regexp.MustCompile(`/shorts/([a-zA-Z0-9_-]{11})`)
		if matches := re.FindStringSubmatch(inputURL); len(matches) > 1 {
			return "video", matches[1], nil
		}
	}

	return "", "", fmt.Errorf("unsupported URL format")
}

func (c *Client) runYtDlp(args ...string) ([]byte, error) {
	defaultArgs := []string{"--no-warnings", "--no-progress", "-J"}
	args = append(defaultArgs, args...)
	cmd := exec.Command("yt-dlp", args...)
	return cmd.Output()
}

func (c *Client) GetPlaylistVideos(playlistURL string, limit int) (*models.Feed, []models.Video, error) {
	data, err := c.runYtDlp("--flat-playlist", playlistURL)
	if err != nil {
		return nil, nil, fmt.Errorf("yt-dlp error: %w", err)
	}

	var info ytdlpPlaylistInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, nil, fmt.Errorf("parse error: %w", err)
	}

	feed := &models.Feed{
		Title:       info.Title,
		Description: info.Description,
		ChannelID:   info.UploaderID,
	}

	videos := make([]models.Video, 0, limit)
	for i, entry := range info.Entries {
		if i >= limit {
			break
		}
		videos = append(videos, models.Video{
			ID:           uuid.New().String(),
			YoutubeID:    entry.ID,
			Title:        entry.Title,
			Description:  entry.Description,
			PublishedAt:  time.Unix(entry.Timestamp, 0),
			ThumbnailURL: entry.Thumbnail,
			Duration:     int(entry.Duration),
		})
	}

	return feed, videos, nil
}

func (c *Client) GetChannelVideos(channelURL string, limit int) (*models.Feed, []models.Video, error) {
	data, err := c.runYtDlp("--flat-playlist", channelURL)
	if err != nil {
		return nil, nil, fmt.Errorf("yt-dlp error: %w", err)
	}

	var info ytdlpChannelInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, nil, fmt.Errorf("parse error: %w", err)
	}

	feed := &models.Feed{
		Title:       info.Title,
		Description: info.Description,
		ChannelID:   info.UploaderID,
	}

	videos := make([]models.Video, 0, limit)
	for i, entry := range info.Entries {
		if i >= limit {
			break
		}
		videos = append(videos, models.Video{
			ID:           uuid.New().String(),
			YoutubeID:    entry.ID,
			Title:        entry.Title,
			Description:  entry.Description,
			PublishedAt:  time.Unix(entry.Timestamp, 0),
			ThumbnailURL: entry.Thumbnail,
			Duration:     int(entry.Duration),
		})
	}

	return feed, videos, nil
}

func (c *Client) FetchFeed(urlStr string, feedID string, limit int) (*models.Feed, []models.Video, error) {
	urlType, _, err := c.parseURL(urlStr)
	if err != nil {
		return nil, nil, err
	}

	var feed *models.Feed
	var videos []models.Video

	switch urlType {
	case "video", "playlist":
		feed, videos, err = c.GetPlaylistVideos(urlStr, limit)
	case "channel", "handle":
		feed, videos, err = c.GetChannelVideos(urlStr, limit)
	default:
		return nil, nil, fmt.Errorf("unsupported URL type: %s", urlType)
	}

	if err != nil {
		return nil, nil, err
	}

	feed.ID = feedID
	feed.YoutubeURL = urlStr

	for i := range videos {
		videos[i].FeedID = feedID
	}

	return feed, videos, nil
}
