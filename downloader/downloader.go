package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultChunkSizeBytes         = 1 << 20 // 1MB
	defaultMaxRetries             = 3       // chunk retries
	temporaryFileSuffix           = ".tmp"  // suffix for temp download
	initialBackoffDuration        = 200 * time.Millisecond
	maxBackoffDuration            = 3 * time.Second
	copyBufferSizeBytes           = 32 * 1024 // 32KB
	headerRange                   = "Range"
	headerContentRange            = "Content-Range"
	headerContentLength           = "Content-Length"
	successMinHTTPStatusCode      = 200
	successMaxHTTPStatusExclusive = 400
)

// Progress holds information about download progress.
type Progress struct {
	TotalSize      int64
	DownloadedSize int64
	Percent        float64
}

// Downloader is responsible for downloading media files with chunked HTTP
// requests, simple retry/backoff, and optional rate limiting.
type Downloader struct {
	Client       *http.Client
	ProgressFunc func(Progress)

	chunkSize    int64
	maxRetries   int
	rateLimitBps int64
}

// New creates a new downloader instance with sane defaults.
// If client is nil, a default http.Client is used. rateLimitBps=0 disables limiting.
func New(client *http.Client, progressFunc func(Progress), rateLimitBps int64) *Downloader {
	if client == nil {
		client = &http.Client{}
	}
	return &Downloader{
		Client:       client,
		ProgressFunc: progressFunc,
		chunkSize:    defaultChunkSizeBytes,
		maxRetries:   defaultMaxRetries,
		rateLimitBps: rateLimitBps,
	}
}

// detectTotalSize tries HEAD first, then GET range 0-0 to infer total size.
func (d *Downloader) detectTotalSize(ctx context.Context, url string) (int64, error) {
	// Try HEAD
	headReq, _ := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	headResp, err := d.Client.Do(headReq)
	if err == nil && headResp != nil {
		defer func() { _ = headResp.Body.Close() }()
		if cl := headResp.Header.Get(headerContentLength); cl != "" {
			if v, err := strconv.ParseInt(cl, 10, 64); err == nil {
				return v, nil
			}
		}
	}
	// Fallback: GET first byte
	getReq, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	getReq.Header.Set(headerRange, "bytes=0-0")
	getResp, err := d.Client.Do(getReq)
	if err != nil {
		return 0, err
	}
	defer func() { _ = getResp.Body.Close() }()
	cr := getResp.Header.Get(headerContentRange) // e.g., bytes 0-0/12345
	if cr != "" {
		parts := strings.Split(cr, "/")
		if len(parts) == 2 {
			if v, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				return v, nil
			}
		}
	}
	if cl := getResp.Header.Get(headerContentLength); cl != "" {
		if v, err := strconv.ParseInt(cl, 10, 64); err == nil {
			return v, nil
		}
	}
	return 0, errors.New("cannot determine total size")
}

// sleepForRate enforces simple rate limit based on bytes written in this step.
func (d *Downloader) sleepForRate(written int64) {
	if d.rateLimitBps <= 0 || written <= 0 {
		return
	}
	dur := time.Duration(int64(time.Second) * written / d.rateLimitBps)
	if dur > 0 {
		time.Sleep(dur)
	}
}

// Download downloads a file by URL and saves it to outputPath. It supports
// resuming from an existing temporary file and reports progress periodically.
func (d *Downloader) Download(ctx context.Context, url string, outputPath string) error {
	// Create/open temporary file for appending
	tmpPath := outputPath + temporaryFileSuffix
	var outFile *os.File
	var err error
	if _, statErr := os.Stat(tmpPath); statErr == nil {
		outFile, err = os.OpenFile(tmpPath, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open tmp for append: %v", err)
		}
	} else {
		outFile, err = os.Create(tmpPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
	}
	defer func() { _ = outFile.Close() }()

	// Determine already downloaded size
	currentInfo, _ := outFile.Stat()
	downloaded := currentInfo.Size()

	totalSize, _ := d.detectTotalSize(ctx, url)
	// Main chunk loop
	for downloaded < totalSize || totalSize == 0 {
		// Calculate range
		start := downloaded
		end := int64(0)
		if totalSize > 0 {
			end = start + d.chunkSize - 1
			if end >= totalSize {
				end = totalSize - 1
			}
		}

		// Perform request with retries
		var resp *http.Response
		var lastErr error
		backoff := initialBackoffDuration
		for attempt := 0; attempt < d.maxRetries; attempt++ {
			req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
			if totalSize > 0 {
				rangeVal := fmt.Sprintf("bytes=%d-%d", start, end)
				req.Header.Set(headerRange, rangeVal)
			}
			resp, lastErr = d.Client.Do(req)
			if lastErr == nil && resp != nil && resp.StatusCode >= successMinHTTPStatusCode && resp.StatusCode < successMaxHTTPStatusExclusive {
				break
			}
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoffDuration {
				backoff = maxBackoffDuration
			}
		}
		if lastErr != nil {
			return fmt.Errorf("download chunk failed: %v", lastErr)
		}
		if resp == nil {
			return fmt.Errorf("empty response")
		}

		// Copy response body to file
		buf := make([]byte, copyBufferSizeBytes)
		for {
			n, rerr := resp.Body.Read(buf)
			if n > 0 {
				if _, werr := outFile.Write(buf[:n]); werr != nil {
					_ = resp.Body.Close()
					return fmt.Errorf("failed to write chunk: %v", werr)
				}
				downloaded += int64(n)
				// Progress
				if d.ProgressFunc != nil {
					p := Progress{TotalSize: totalSize, DownloadedSize: downloaded}
					if totalSize > 0 {
						p.Percent = float64(downloaded) / float64(totalSize) * 100
					}
					d.ProgressFunc(p)
				}
				// Rate limiting
				d.sleepForRate(int64(n))
			}
			if rerr == io.EOF {
				break
			}
			if rerr != nil {
				_ = resp.Body.Close()
				return fmt.Errorf("failed to read response body: %v", rerr)
			}
		}
		_ = resp.Body.Close()

		// If totalSize is unknown and the server closed the body — finish
		if totalSize == 0 {
			break
		}
		// If end reached
		if downloaded >= totalSize {
			break
		}
	}

	// Rename temporary file
	return os.Rename(tmpPath, outputPath)
}
