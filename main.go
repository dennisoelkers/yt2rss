package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dnhn/yt2rss/internal/db"
	"github.com/dnhn/yt2rss/internal/feed"
	"github.com/dnhn/yt2rss/internal/youtube"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	addr := flag.String("addr", ":8080", "Address to listen on")
	dbPath := flag.String("db", "yt2rss.db", "Path to SQLite database")
	baseURL := flag.String("base-url", "http://localhost:8080", "Base URL for RSS feed links")
	cacheDir := flag.String("cache-dir", "", "Directory to cache audio files")
	binDir := flag.String("bin-dir", "", "Directory containing yt-dlp binary")
	flag.Parse()

	execDir, _ := os.Executable()
	binPath := filepath.Join(filepath.Dir(execDir), "bin", "yt-dlp")
	if *binDir != "" {
		binPath = filepath.Join(*binDir, "yt-dlp")
	}
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		binPath = "yt-dlp"
	}

	db, err := db.New(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	ytClient := youtube.NewClient()
	handler := feed.New(db, ytClient, *baseURL, *cacheDir, binPath)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	handler.RegisterRoutes(r)

	log.Printf("Starting yt2rss on %s", *addr)
	if err := http.ListenAndServe(*addr, r); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
