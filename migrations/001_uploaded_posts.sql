CREATE TABLE IF NOT EXISTS uploaded_posts (
    id SERIAL PRIMARY KEY,
    reddit_post_id TEXT UNIQUE NOT NULL,
    subreddit TEXT NOT NULL,
    title TEXT,
    permalink TEXT,
    score DOUBLE PRECISION,
    video_url TEXT,
    local_video_path TEXT,
    instagram_media_id TEXT,
    upload_status TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    uploaded_at TIMESTAMP
);
