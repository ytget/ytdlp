//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"

	"github.com/ytget/ytdlp/v2"
)

func TestE2E_Download(t *testing.T) {
	if os.Getenv("YTDLP_E2E") == "" {
		t.Skip("YTDLP_E2E not set")
	}
	url := os.Getenv("YTDLP_E2E_URL")
	if url == "" {
		url = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	}
	dl := ytdlp.New().WithOutputPath("")
	ctx := context.Background()
	_, err := dl.Download(ctx, url)
	if err != nil {
		t.Fatalf("e2e download failed: %v", err)
	}
}
