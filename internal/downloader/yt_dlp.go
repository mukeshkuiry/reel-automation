package downloader

import (
	"context"
	"fmt"
	"os/exec"
)

type Downloader interface {
	Download(ctx context.Context, url string, outputPath string) error
}

type YTDLP struct{}

func NewYTDLP() *YTDLP {
	return &YTDLP{}
}

func (d *YTDLP) Download(ctx context.Context, videoURL string, outputPath string) error {
	cmd := exec.CommandContext(ctx, "yt-dlp", videoURL, "-o", outputPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("yt-dlp failed: %w: %s", err, string(out))
	}
	return nil
}
