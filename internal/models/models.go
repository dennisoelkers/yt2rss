package models

import "time"

type Feed struct {
	ID          string    `json:"id"`
	YoutubeURL  string    `json:"youtube_url"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ChannelID   string    `json:"channel_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Video struct {
	ID           string    `json:"id"`
	FeedID       string    `json:"feed_id"`
	YoutubeID    string    `json:"youtube_id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	PublishedAt  time.Time `json:"published_at"`
	ThumbnailURL string    `json:"thumbnail_url"`
	Duration     int       `json:"duration"`
}

type ChannelInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ThumbURL    string `json:"thumbnail_url"`
}

type PlaylistInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ThumbURL    string `json:"thumbnail_url"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
}
