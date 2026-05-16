package workers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mukeshkuiry/reel-automation/internal/caption"
	"github.com/mukeshkuiry/reel-automation/internal/dedup"
	"github.com/mukeshkuiry/reel-automation/internal/downloader"
	"github.com/mukeshkuiry/reel-automation/internal/formatter"
	"github.com/mukeshkuiry/reel-automation/internal/instagram"
	"github.com/mukeshkuiry/reel-automation/internal/ranking"
	"github.com/mukeshkuiry/reel-automation/internal/reddit"
	"github.com/mukeshkuiry/reel-automation/internal/storage"
)

type Pipeline struct {
	fetcher                reddit.Fetcher
	deduper                *dedup.Service
	downloader             downloader.Downloader
	formatter              formatter.Formatter
	publisher              instagram.Publisher
	store                  storage.Store
	workDir                string
	retries                int
	formatRetries          int
	uploadRetries          int
	retryBackoff           time.Duration
	downloadTimeout        time.Duration
	formatTimeout          time.Duration
	uploadSourceVideoIsURL bool
}

func NewPipeline(
	fetcher reddit.Fetcher,
	deduper *dedup.Service,
	downloader downloader.Downloader,
	formatter formatter.Formatter,
	publisher instagram.Publisher,
	store storage.Store,
	workDir string,
	retries int,
	formatRetries int,
	uploadRetries int,
	retryBackoff time.Duration,
	downloadTimeout time.Duration,
	formatTimeout time.Duration,
	uploadSourceVideoIsURL bool,
) *Pipeline {
	if retries < 1 {
		retries = 1
	}
	if retryBackoff <= 0 {
		retryBackoff = 3 * time.Second
	}
	if formatRetries < 1 {
		formatRetries = 2
	}
	if uploadRetries < 1 {
		uploadRetries = 5
	}
	return &Pipeline{
		fetcher:                fetcher,
		deduper:                deduper,
		downloader:             downloader,
		formatter:              formatter,
		publisher:              publisher,
		store:                  store,
		workDir:                workDir,
		retries:                retries,
		formatRetries:          formatRetries,
		uploadRetries:          uploadRetries,
		retryBackoff:           retryBackoff,
		downloadTimeout:        downloadTimeout,
		formatTimeout:          formatTimeout,
		uploadSourceVideoIsURL: uploadSourceVideoIsURL,
	}
}

func (p *Pipeline) Run(ctx context.Context, subreddits []string, fetchLimit int, workerConcurrency int) error {
	if len(subreddits) == 0 {
		return fmt.Errorf("no subreddits configured")
	}
	if workerConcurrency < 1 {
		workerConcurrency = 1
	}
	if err := os.MkdirAll(p.workDir, 0o755); err != nil {
		return fmt.Errorf("create work dir: %w", err)
	}

	posts, err := p.fetchAndRank(ctx, subreddits, fetchLimit, workerConcurrency)
	if err != nil {
		return err
	}
	for _, ranked := range posts {
		seen, err := p.deduper.AlreadyPosted(ctx, ranked.Post.ID)
		if err != nil {
			return fmt.Errorf("dedup check failed for %s: %w", ranked.Post.ID, err)
		}
		if seen {
			continue
		}
		if err := p.processPost(ctx, ranked); err != nil {
			continue
		}
		return nil
	}
	return fmt.Errorf("no eligible posts available")
}

func (p *Pipeline) fetchAndRank(ctx context.Context, subreddits []string, fetchLimit int, workerConcurrency int) ([]ranking.RankedPost, error) {
	type result struct {
		posts []reddit.Post
		err   error
	}

	workCh := make(chan string)
	resultCh := make(chan result, len(subreddits))
	var wg sync.WaitGroup

	for i := 0; i < workerConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for subreddit := range workCh {
				posts, err := p.fetcher.FetchTrending(ctx, subreddit, fetchLimit)
				resultCh <- result{posts: posts, err: err}
			}
		}()
	}
	go func() {
		for _, s := range subreddits {
			workCh <- s
		}
		close(workCh)
		wg.Wait()
		close(resultCh)
	}()

	collected := make([]reddit.Post, 0, len(subreddits)*fetchLimit)
	var seenSuccess bool
	for r := range resultCh {
		if r.err != nil {
			continue
		}
		seenSuccess = true
		collected = append(collected, r.posts...)
	}
	if !seenSuccess {
		return nil, fmt.Errorf("all subreddit fetches failed")
	}
	return ranking.Rank(collected, time.Now().UTC()), nil
}

func (p *Pipeline) processPost(ctx context.Context, ranked ranking.RankedPost) error {
	rawPath := filepath.Join(p.workDir, ranked.Post.ID+"_raw.mp4")
	reelPath := filepath.Join(p.workDir, ranked.Post.ID+"_reel.mp4")

	downloadCtx := ctx
	var cancelDownload context.CancelFunc
	if p.downloadTimeout > 0 {
		downloadCtx, cancelDownload = context.WithTimeout(ctx, p.downloadTimeout)
		defer cancelDownload()
	}
	if err := retry(p.retries, p.retryBackoff, func() error {
		return p.downloader.Download(downloadCtx, ranked.Post.VideoURL, rawPath)
	}); err != nil {
		_ = p.saveRecord(ctx, ranked, rawPath, "", "download_failed", "")
		return err
	}

	formatCtx := ctx
	var cancelFormat context.CancelFunc
	if p.formatTimeout > 0 {
		formatCtx, cancelFormat = context.WithTimeout(ctx, p.formatTimeout)
		defer cancelFormat()
	}
	if err := retry(p.formatRetries, p.retryBackoff, func() error {
		return p.formatter.ToReel(formatCtx, rawPath, reelPath)
	}); err != nil {
		_ = p.saveRecord(ctx, ranked, rawPath, "", "format_failed", "")
		return err
	}

	c := caption.Generate(ranked.Post)
	uploadVideoSource := ranked.Post.VideoURL
	if !p.uploadSourceVideoIsURL {
		uploadVideoSource = reelPath
	}
	var mediaID string
	err := retry(p.uploadRetries, p.retryBackoff, func() error {
		publishedID, err := p.publisher.Publish(ctx, uploadVideoSource, c)
		if err != nil {
			return err
		}
		mediaID = publishedID
		return nil
	})
	if err != nil {
		_ = p.saveRecord(ctx, ranked, rawPath, reelPath, "upload_failed", "")
		return err
	}

	return p.saveRecord(ctx, ranked, rawPath, reelPath, "uploaded", mediaID)
}

func (p *Pipeline) saveRecord(ctx context.Context, ranked ranking.RankedPost, rawPath, reelPath, status, mediaID string) error {
	record := storage.UploadRecord{
		RedditPostID:     ranked.Post.ID,
		Subreddit:        ranked.Post.Subreddit,
		Title:            ranked.Post.Title,
		Permalink:        ranked.Post.Permalink,
		Score:            ranked.Score,
		VideoURL:         ranked.Post.VideoURL,
		LocalVideoPath:   reelPath,
		InstagramMediaID: mediaID,
		UploadStatus:     status,
		CreatedAt:        time.Now().UTC(),
	}
	if status == "uploaded" {
		now := time.Now().UTC()
		record.UploadedAt = &now
	}
	if record.LocalVideoPath == "" {
		record.LocalVideoPath = rawPath
	}
	return p.store.Save(ctx, record)
}

func retry(attempts int, backoff time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if i < attempts-1 {
			time.Sleep(backoff)
		}
	}
	return err
}
