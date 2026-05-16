package formatter

import (
	"context"
	"fmt"
	"os/exec"
)

type Formatter interface {
	ToReel(ctx context.Context, inputPath string, outputPath string) error
}

type FFmpeg struct{}

func NewFFmpeg() *FFmpeg {
	return &FFmpeg{}
}

func (f *FFmpeg) ToReel(ctx context.Context, inputPath string, outputPath string) error {
	filter := "[0:v]scale=1080:1920:force_original_aspect_ratio=increase,boxblur=20[bg];" +
		"[0:v]scale=1080:-1[fg];" +
		"[bg][fg]overlay=(W-w)/2:(H-h)/2"
	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i", inputPath,
		"-filter_complex", filter,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "23",
		"-pix_fmt", "yuv420p",
		outputPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w: %s", err, string(out))
	}
	return nil
}
