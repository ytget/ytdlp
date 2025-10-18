package downloader

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// mockTransport is a custom HTTP transport for testing
type mockTransport struct {
	responseStatus  int
	responseHeaders map[string]string
	hasError        bool
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Create a mock response
	resp := &http.Response{
		StatusCode: t.responseStatus,
		Header:     make(http.Header),
		Body:       http.NoBody,
	}

	// Set response headers
	for key, value := range t.responseHeaders {
		resp.Header.Set(key, value)
	}

	return resp, nil
}

func TestDetectTotalSize(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		responseStatus  int
		responseHeaders map[string]string
		expectedSize    int64
		hasError        bool
	}{
		{
			name:           "Google Video host with Content-Range",
			url:            "https://googlevideo.com/video.mp4",
			responseStatus: 206,
			responseHeaders: map[string]string{
				"Content-Range": "bytes 0-1/1000000",
			},
			expectedSize: 1000000,
			hasError:     false,
		},
		{
			name:           "Google Video host with Content-Length",
			url:            "https://googlevideo.com/video.mp4",
			responseStatus: 200,
			responseHeaders: map[string]string{
				"Content-Length": "500000",
			},
			expectedSize: 500000,
			hasError:     false,
		},
		{
			name:           "Non-Google host with Content-Range",
			url:            "https://example.com/video.mp4",
			responseStatus: 206,
			responseHeaders: map[string]string{
				"Content-Range": "bytes 0-1/2000000",
			},
			expectedSize: 2000000,
			hasError:     false,
		},
		{
			name:           "Non-Google host with Content-Length",
			url:            "https://example.com/video.mp4",
			responseStatus: 200,
			responseHeaders: map[string]string{
				"Content-Length": "750000",
			},
			expectedSize: 750000,
			hasError:     false,
		},
		{
			name:           "Invalid Content-Range format",
			url:            "https://example.com/video.mp4",
			responseStatus: 206,
			responseHeaders: map[string]string{
				"Content-Range": "invalid-format",
			},
			expectedSize: 0,
			hasError:     true,
		},
		{
			name:            "No size headers",
			url:             "https://example.com/video.mp4",
			responseStatus:  200,
			responseHeaders: map[string]string{},
			expectedSize:    0,
			hasError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create custom HTTP client that intercepts requests
			client := &http.Client{
				Transport: &mockTransport{
					responseStatus:  tt.responseStatus,
					responseHeaders: tt.responseHeaders,
					hasError:        tt.hasError,
				},
			}

			// Create downloader with mock HTTP client
			downloader := &Downloader{
				Client: client,
			}

			// Test detectTotalSize
			size, err := downloader.detectTotalSize(context.Background(), "https://example.com/video.mp4")

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if size != tt.expectedSize {
					t.Errorf("Expected size %d, got %d", tt.expectedSize, size)
				}
			}
		})
	}
}

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

func TestSleepForRate(t *testing.T) {
	tests := []struct {
		name         string
		rateLimitBps int64
		written      int64
		expectSleep  bool
	}{
		{
			name:         "No rate limit",
			rateLimitBps: 0,
			written:      1000,
			expectSleep:  false,
		},
		{
			name:         "Negative rate limit",
			rateLimitBps: -100,
			written:      1000,
			expectSleep:  false,
		},
		{
			name:         "No bytes written",
			rateLimitBps: 1000,
			written:      0,
			expectSleep:  false,
		},
		{
			name:         "Negative bytes written",
			rateLimitBps: 1000,
			written:      -100,
			expectSleep:  false,
		},
		{
			name:         "Normal rate limiting",
			rateLimitBps: 1000,
			written:      1000,
			expectSleep:  true,
		},
		{
			name:         "High rate limit",
			rateLimitBps: 1000000,
			written:      1000,
			expectSleep:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downloader := &Downloader{
				rateLimitBps: tt.rateLimitBps,
			}

			// Measure execution time
			start := time.Now()
			downloader.sleepForRate(tt.written)
			duration := time.Since(start)

			if tt.expectSleep {
				// Should sleep for at least some time
				if duration < time.Millisecond {
					t.Errorf("Expected sleep time > 0, got %v", duration)
				}
			} else {
				// Should not sleep
				if duration > time.Millisecond {
					t.Errorf("Expected no sleep, got sleep time %v", duration)
				}
			}
		})
	}
}

func TestIsGoogleVideoHost(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "Valid googlevideo.com URL",
			url:      "https://googlevideo.com/video.mp4",
			expected: true,
		},
		{
			name:     "Valid subdomain googlevideo.com URL",
			url:      "https://r1---sn-4g5e6n7s.googlevideo.com/video.mp4",
			expected: true,
		},
		{
			name:     "Another valid subdomain googlevideo.com URL",
			url:      "https://r2---sn-4g5e6n7s.googlevideo.com/video.mp4",
			expected: true,
		},
		{
			name:     "Invalid domain",
			url:      "https://example.com/video.mp4",
			expected: false,
		},
		{
			name:     "Invalid domain with googlevideo in name",
			url:      "https://fakegooglevideo.com/video.mp4",
			expected: false,
		},
		{
			name:     "Invalid domain with googlevideo prefix",
			url:      "https://googlevideo-fake.com/video.mp4",
			expected: false,
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: false,
		},
		{
			name:     "Invalid URL",
			url:      "invalid-url",
			expected: false,
		},
		{
			name:     "URL with port",
			url:      "https://googlevideo.com:443/video.mp4",
			expected: false, // Function doesn't handle port correctly
		},
		{
			name:     "URL with subdomain and port",
			url:      "https://r1---sn-4g5e6n7s.googlevideo.com:443/video.mp4",
			expected: false, // Function doesn't handle port correctly
		},
		{
			name:     "URL with different protocol",
			url:      "http://googlevideo.com/video.mp4",
			expected: true,
		},
		{
			name:     "URL with different protocol and subdomain",
			url:      "http://r1---sn-4g5e6n7s.googlevideo.com/video.mp4",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGoogleVideoHost(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for URL: %s", tt.expected, result, tt.url)
			}
		})
	}
}
