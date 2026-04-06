package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/dnhn/yt2rss/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func New(dbPath string) (*DB, error) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		f, err := os.Create(dbPath)
		if err != nil {
			return nil, fmt.Errorf("creating db: %w", err)
		}
		f.Close()
	}

	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging db: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) Migrate() error {
	migration := `
	CREATE TABLE IF NOT EXISTS feeds (
		id TEXT PRIMARY KEY,
		youtube_url TEXT NOT NULL UNIQUE,
		title TEXT,
		description TEXT,
		channel_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS videos (
		id TEXT PRIMARY KEY,
		feed_id TEXT NOT NULL,
		youtube_id TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT,
		published_at DATETIME,
		thumbnail_url TEXT,
		duration INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE,
		UNIQUE(feed_id, youtube_id)
	);

	CREATE INDEX IF NOT EXISTS idx_videos_feed_id ON videos(feed_id);
	CREATE INDEX IF NOT EXISTS idx_videos_published_at ON videos(published_at DESC);
	`

	_, err := db.Exec(migration)
	return err
}

func (db *DB) CreateFeed(feed *models.Feed) error {
	query := `INSERT INTO feeds (id, youtube_url, title, description, channel_id) VALUES (?, ?, ?, ?, ?)`
	_, err := db.Exec(query, feed.ID, feed.YoutubeURL, feed.Title, feed.Description, feed.ChannelID)
	return err
}

func (db *DB) GetFeed(id string) (*models.Feed, error) {
	query := `SELECT id, youtube_url, title, description, channel_id, created_at, updated_at FROM feeds WHERE id = ?`
	row := db.QueryRow(query, id)

	var feed models.Feed
	err := row.Scan(&feed.ID, &feed.YoutubeURL, &feed.Title, &feed.Description, &feed.ChannelID, &feed.CreatedAt, &feed.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &feed, nil
}

func (db *DB) GetFeedByURL(url string) (*models.Feed, error) {
	query := `SELECT id, youtube_url, title, description, channel_id, created_at, updated_at FROM feeds WHERE youtube_url = ?`
	row := db.QueryRow(query, url)

	var feed models.Feed
	err := row.Scan(&feed.ID, &feed.YoutubeURL, &feed.Title, &feed.Description, &feed.ChannelID, &feed.CreatedAt, &feed.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &feed, nil
}

func (db *DB) UpdateFeed(feed *models.Feed) error {
	query := `UPDATE feeds SET title = ?, description = ?, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, feed.Title, feed.Description, time.Now(), feed.ID)
	return err
}

func (db *DB) DeleteFeed(id string) error {
	_, err := db.Exec(`DELETE FROM feeds WHERE id = ?`, id)
	return err
}

func (db *DB) ListFeeds() ([]models.Feed, error) {
	query := `SELECT id, youtube_url, title, description, channel_id, created_at, updated_at FROM feeds ORDER BY created_at DESC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []models.Feed
	for rows.Next() {
		var feed models.Feed
		if err := rows.Scan(&feed.ID, &feed.YoutubeURL, &feed.Title, &feed.Description, &feed.ChannelID, &feed.CreatedAt, &feed.UpdatedAt); err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}
	return feeds, rows.Err()
}

func (db *DB) UpsertVideo(video *models.Video) error {
	query := `INSERT INTO videos (id, feed_id, youtube_id, title, description, published_at, thumbnail_url, duration)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			  ON CONFLICT(feed_id, youtube_id) DO UPDATE SET
			  title = excluded.title, description = excluded.description, 
			  published_at = excluded.published_at, thumbnail_url = excluded.thumbnail_url,
			  duration = excluded.duration`
	_, err := db.Exec(query, video.ID, video.FeedID, video.YoutubeID, video.Title, video.Description,
		video.PublishedAt, video.ThumbnailURL, video.Duration)
	return err
}

func (db *DB) GetVideos(feedID string, limit int) ([]models.Video, error) {
	query := `SELECT id, feed_id, youtube_id, title, description, published_at, thumbnail_url, duration 
			  FROM videos WHERE feed_id = ? ORDER BY published_at DESC LIMIT ?`
	rows, err := db.Query(query, feedID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []models.Video
	for rows.Next() {
		var v models.Video
		if err := rows.Scan(&v.ID, &v.FeedID, &v.YoutubeID, &v.Title, &v.Description,
			&v.PublishedAt, &v.ThumbnailURL, &v.Duration); err != nil {
			return nil, err
		}
		videos = append(videos, v)
	}
	return videos, rows.Err()
}

func (db *DB) DeleteVideosForFeed(feedID string) error {
	_, err := db.Exec(`DELETE FROM videos WHERE feed_id = ?`, feedID)
	return err
}
