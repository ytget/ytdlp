package downloader

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// simple range-aware handler serving a fixed byte slice
func makeServer(data []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHdr := r.Header.Get("Range")
		start := 0
		end := len(data) - 1
		if rangeHdr != "" {
			// bytes=a-b
			var a, b int
			if _, err := fmt.Sscanf(rangeHdr, "bytes=%d-%d", &a, &b); err == nil {
				start = a
				if b >= 0 {
					end = b
				}
			}
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(data)))
			w.WriteHeader(http.StatusPartialContent)
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", end-start+1))
		_, _ = w.Write(data[start : end+1])
	}))
}

func TestDownloadResume(t *testing.T) {
	data := make([]byte, 2<<20) // 2MB
	for i := range data {
		data[i] = byte(i % 251)
	}
	server := makeServer(data)
	defer server.Close()

	ctx := context.Background()
	dl := New(server.Client(), nil, 0)
	out := t.TempDir() + "/file.bin"
	tmp := out + ".tmp"

	// Pre-create partial tmp (first 1MB)
	if err := os.WriteFile(tmp, data[:1<<20], 0644); err != nil {
		t.Fatalf("precreate tmp failed: %v", err)
	}

	// Resume and complete
	if err := dl.Download(ctx, server.URL, out); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	// Verify file contents and size
	bs, err := os.ReadFile(out)
	if err != nil || int64(len(bs)) != int64(len(data)) {
		t.Fatalf("bad size/content: err=%v got=%d want=%d", err, len(bs), len(data))
	}
	if string(bs[:1024]) != string(data[:1024]) || string(bs[len(bs)-1024:]) != string(data[len(data)-1024:]) {
		t.Fatalf("content mismatch")
	}
}
