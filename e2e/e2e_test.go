//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"

	"github.com/ytget/ytdlp/v2"
)

// TestE2E_Download tests the complete download workflow with real URLs.
// To run: go test -tags=e2e -v ./e2e
// Set YTDLP_E2E=1 to enable, YTDLP_E2E_URL to specify test URL
func TestE2E_Download(t *testing.T) {
	if os.Getenv("YTDLP_E2E") == "" {
		t.Skip("YTDLP_E2E not set - skipping e2e tests")
	}

	url := os.Getenv("YTDLP_E2E_URL")
	if url == "" {
		url = "https://example.com/watch?v=test123"
	}

	dl := ytdlp.New().WithOutputPath("")
	ctx := context.Background()

	_, err := dl.Download(ctx, url)
	if err != nil {
		t.Fatalf("e2e download failed: %v", err)
	}

	t.Logf("Successfully downloaded from: %s", url)
}
