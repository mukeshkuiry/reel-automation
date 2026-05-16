package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mukeshkuiry/reel-automation/internal/config"
	"github.com/mukeshkuiry/reel-automation/internal/dedup"
	"github.com/mukeshkuiry/reel-automation/internal/downloader"
	"github.com/mukeshkuiry/reel-automation/internal/formatter"
	"github.com/mukeshkuiry/reel-automation/internal/instagram"
	"github.com/mukeshkuiry/reel-automation/internal/reddit"
	"github.com/mukeshkuiry/reel-automation/internal/scheduler"
	"github.com/mukeshkuiry/reel-automation/internal/storage"
	"github.com/mukeshkuiry/reel-automation/internal/workers"
)

func main() {
	cfg := config.Load()

	httpClient := &http.Client{}
	fetcher := reddit.NewClient(
		httpClient,
		cfg.RedditClientID,
		cfg.RedditClientSecret,
		cfg.RedditUsername,
		cfg.RedditPassword,
		cfg.RedditUserAgent,
		cfg.RedditTopWindow,
		cfg.MaxPostAge,
	)
	store := storage.NewInMemoryStore()
	deduper := dedup.New(store)
	pipeline := workers.NewPipeline(
		fetcher,
		deduper,
		downloader.NewYTDLP(),
		formatter.NewFFmpeg(),
		instagram.NewGraphPublisher(httpClient, cfg.InstagramBusinessID, cfg.InstagramAccessToken),
		store,
		cfg.WorkDir,
		cfg.MaxPipelineRetries,
		cfg.FormatRetries,
		cfg.RetryBackoff,
		cfg.DownloadTimeout,
		cfg.FormatTimeout,
		cfg.UploadSourceVideoIsURL,
	)

	run := func(ctx context.Context) error {
		return pipeline.Run(ctx, cfg.Subreddits, cfg.FetchPerSubreddit, cfg.WorkerConcurrency)
	}

	if !cfg.EnableScheduler {
		if err := run(context.Background()); err != nil {
			log.Fatal(err)
		}
		return
	}

	s := scheduler.New()
	if err := s.Register(cfg.CronExpr, run); err != nil {
		log.Fatal(err)
	}
	s.Start()
	defer s.Stop()

	log.Printf("scheduler started with cron expression %q", cfg.CronExpr)
	waitForShutdown()
}

func waitForShutdown() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
