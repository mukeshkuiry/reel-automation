package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Subreddits             []string
	CronExpr               string
	WorkDir                string
	RedditClientID         string
	RedditClientSecret     string
	RedditUsername         string
	RedditPassword         string
	RedditUserAgent        string
	RedditTopWindow        string
	MaxPostAge             time.Duration
	InstagramAccessToken   string
	InstagramBusinessID    string
	FetchPerSubreddit      int
	MaxPipelineRetries     int
	FormatRetries          int
	UploadRetries          int
	RetryBackoff           time.Duration
	DownloadTimeout        time.Duration
	FormatTimeout          time.Duration
	EnableScheduler        bool
	WorkerConcurrency      int
	UploadSourceVideoIsURL bool
}

func Load() Config {
	return Config{
		Subreddits:             splitCSV(getenv("SUBREDDITS", "funny,interestingasfuck,damnthatsinteresting,nextfuckinglevel")),
		CronExpr:               getenv("CRON_EXPRESSION", "0 */2 * * *"),
		WorkDir:                getenv("WORK_DIR", "tmp/videos"),
		RedditClientID:         os.Getenv("REDDIT_CLIENT_ID"),
		RedditClientSecret:     os.Getenv("REDDIT_CLIENT_SECRET"),
		RedditUsername:         os.Getenv("REDDIT_USERNAME"),
		RedditPassword:         os.Getenv("REDDIT_PASSWORD"),
		RedditUserAgent:        getenv("REDDIT_USER_AGENT", "reel-automation/1.0"),
		RedditTopWindow:        getenv("REDDIT_TOP_WINDOW", "hour"),
		MaxPostAge:             time.Duration(getenvInt("MAX_POST_AGE_MINUTES", 120)) * time.Minute,
		InstagramAccessToken:   os.Getenv("INSTAGRAM_ACCESS_TOKEN"),
		InstagramBusinessID:    os.Getenv("INSTAGRAM_BUSINESS_ID"),
		FetchPerSubreddit:      getenvInt("FETCH_LIMIT", 25),
		MaxPipelineRetries:     getenvInt("MAX_PIPELINE_RETRIES", 3),
		FormatRetries:          getenvInt("FORMAT_RETRIES", 2),
		UploadRetries:          getenvInt("UPLOAD_RETRIES", 5),
		RetryBackoff:           time.Duration(getenvInt("RETRY_BACKOFF_SECONDS", 3)) * time.Second,
		DownloadTimeout:        time.Duration(getenvInt("DOWNLOAD_TIMEOUT_SECONDS", 120)) * time.Second,
		FormatTimeout:          time.Duration(getenvInt("FORMAT_TIMEOUT_SECONDS", 120)) * time.Second,
		EnableScheduler:        getenvBool("ENABLE_SCHEDULER", false),
		WorkerConcurrency:      getenvInt("WORKER_CONCURRENCY", 2),
		UploadSourceVideoIsURL: getenvBool("UPLOAD_SOURCE_VIDEO_IS_URL", true),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvBool(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	return v == "1" || v == "true" || v == "yes"
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
