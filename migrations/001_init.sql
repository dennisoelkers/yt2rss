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

CREATE INDEX idx_videos_feed_id ON videos(feed_id);
CREATE INDEX idx_videos_published_at ON videos(published_at DESC);
