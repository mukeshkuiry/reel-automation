# Reddit to Instagram Reels Automation

This repository contains an MVP implementation of an automated Reddit-to-Instagram Reels pipeline in Go.

## Implemented modules

- Scheduler (`internal/scheduler`) using `robfig/cron` with default `0 */2 * * *`
- Reddit fetcher (`internal/reddit`) with OAuth2 password flow and video post filters
- Trend ranking engine (`internal/ranking`) using:
  - `velocity = upvotes / age_in_minutes`
  - `score = velocity*0.7 + comments*0.3`
- Deduplication (`internal/dedup` + `internal/storage`)
- Video downloader (`internal/downloader`) via `yt-dlp`
- Reel formatter (`internal/formatter`) via `ffmpeg` (9:16 pipeline)
- Caption generator (`internal/caption`)
- Instagram publisher (`internal/instagram`) with Graph API `/media` and `/media_publish`
- Pipeline workers (`internal/workers`) with retry behavior and concurrent subreddit fetching

## Project layout

```text
cmd/reel-automation
internal/
  caption/
  config/
  dedup/
  downloader/
  formatter/
  instagram/
  ranking/
  reddit/
  scheduler/
  storage/
  workers/
migrations/
```

## Configuration

Environment variables:

```env
SUBREDDITS=Funny,interestingasfuck,damnthatsinteresting,nextfuckinglevel
CRON_EXPRESSION=0 */2 * * *
ENABLE_SCHEDULER=false

REDDIT_CLIENT_ID=
REDDIT_CLIENT_SECRET=
REDDIT_USERNAME=
REDDIT_PASSWORD=
REDDIT_USER_AGENT=reel-automation/1.0
REDDIT_TOP_WINDOW=hour
MAX_POST_AGE_MINUTES=120
UPLOAD_RETRIES=5

INSTAGRAM_ACCESS_TOKEN=
INSTAGRAM_BUSINESS_ID=
```

## Run

```bash
go run ./cmd/reel-automation
```

## Test

```bash
go test ./...
```
