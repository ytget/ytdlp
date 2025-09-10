package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ytget/ytdlp/internal/logger"
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
	headerUserAgent               = "User-Agent"
	headerAccept                  = "Accept"
	headerAcceptLanguage          = "Accept-Language"
	headerAcceptEncoding          = "Accept-Encoding"
	headerReferer                 = "Referer"
	headerOrigin                  = "Origin"
	headerConnection              = "Connection"
	headerCacheControl            = "Cache-Control"
	headerDNT                     = "DNT"
	headerUpgradeInsecureRequests = "Upgrade-Insecure-Requests"
	headerSecFetchDest            = "Sec-Fetch-Dest"
	headerSecFetchMode            = "Sec-Fetch-Mode"
	headerSecFetchSite            = "Sec-Fetch-Site"
	headerSecFetchUser            = "Sec-Fetch-User"
	headerSecChUa                 = "Sec-Ch-Ua"
	headerSecChUaMobile           = "Sec-Ch-Ua-Mobile"
	headerSecChUaPlatform         = "Sec-Ch-Ua-Platform"
	successMinHTTPStatusCode      = 200
	successMaxHTTPStatusExclusive = 400

	userAgentValue = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"
)

var (
	// downloaderLogger is the logger for the downloader component
	downloaderLogger = logger.WithComponent(logger.ComponentDownloader)
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

func isGoogleVideoHost(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	h := strings.ToLower(u.Host)
	return strings.HasSuffix(h, ".googlevideo.com") || h == "googlevideo.com"
}

// detectTotalSize tries HEAD first, then GET range 0-0 to infer total size.
func (d *Downloader) detectTotalSize(ctx context.Context, urlStr string) (int64, error) {
	if isGoogleVideoHost(urlStr) {
		// Skip HEAD for googlevideo; perform GET bytes=0-1 directly
		getReq, _ := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
		getReq.Header.Set(headerUserAgent, userAgentValue)
		getReq.Header.Set(headerAccept, "*/*")
		getReq.Header.Set(headerAcceptEncoding, "identity")
		getReq.Header.Set(headerConnection, "keep-alive")
		getReq.Header.Set(headerCacheControl, "no-cache")
		getReq.Header.Set(headerRange, "bytes=0-1")

		downloaderLogger.Trace("GET range request headers", map[string]interface{}{
			"headers": getReq.Header,
		})
		getResp, err := d.Client.Do(getReq)
		if err != nil {
			return 0, err
		}
		defer func() { _ = getResp.Body.Close() }()
		downloaderLogger.Trace("GET range response", map[string]interface{}{
			"status_code": getResp.StatusCode,
			"headers":     getResp.Header,
		})
		cr := getResp.Header.Get(headerContentRange)
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

	// Non-googlevideo: attempt HEAD first
	headReq, _ := http.NewRequestWithContext(ctx, "HEAD", urlStr, nil)
	headReq.Header.Set(headerUserAgent, userAgentValue)
	headReq.Header.Set(headerAccept, "*/*")
	headReq.Header.Set(headerAcceptLanguage, "en-US,en;q=0.9")
	headReq.Header.Set(headerAcceptEncoding, "identity")
	headReq.Header.Set(headerConnection, "keep-alive")
	headReq.Header.Set(headerCacheControl, "no-cache")
	headReq.Header.Set(headerRange, "bytes=0-1")

	downloaderLogger.Trace("HEAD request headers", map[string]interface{}{
		"headers": headReq.Header,
	})
	headResp, err := d.Client.Do(headReq)
	if err == nil && headResp != nil {
		defer func() { _ = headResp.Body.Close() }()
		downloaderLogger.Trace("HEAD response", map[string]interface{}{
			"status_code": headResp.StatusCode,
			"headers":     headResp.Header,
		})
		if cr := headResp.Header.Get(headerContentRange); cr != "" {
			parts := strings.Split(cr, "/")
			if len(parts) == 2 {
				if v, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					return v, nil
				}
			}
		}
		if cl := headResp.Header.Get(headerContentLength); cl != "" {
			if v, err := strconv.ParseInt(cl, 10, 64); err == nil {
				return v, nil
			}
		}
	}

	// Fallback: GET bytes=0-1
	getReq, _ := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	getReq.Header.Set(headerUserAgent, userAgentValue)
	getReq.Header.Set(headerAccept, "*/*")
	getReq.Header.Set(headerAcceptLanguage, "en-US,en;q=0.9")
	getReq.Header.Set(headerAcceptEncoding, "identity")
	getReq.Header.Set(headerConnection, "keep-alive")
	getReq.Header.Set(headerCacheControl, "no-cache")
	getReq.Header.Set(headerRange, "bytes=0-1")

	downloaderLogger.Trace("GET range request headers", map[string]interface{}{
		"headers": getReq.Header,
	})
	getResp, err := d.Client.Do(getReq)
	if err != nil {
		return 0, err
	}
	defer func() { _ = getResp.Body.Close() }()
	downloaderLogger.Trace("GET range response", map[string]interface{}{
		"status_code": getResp.StatusCode,
		"headers":     getResp.Header,
	})
	cr := getResp.Header.Get(headerContentRange)
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
func (d *Downloader) Download(ctx context.Context, urlStr string, outputPath string) error {
	downloaderLogger.Info("Starting download", map[string]interface{}{
		"output_path": outputPath,
	})

	tmpPath := outputPath + temporaryFileSuffix
	var outFile *os.File
	var err error
	if _, statErr := os.Stat(tmpPath); statErr == nil {
		outFile, err = os.OpenFile(tmpPath, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open tmp for append: %v", err)
		}
		downloaderLogger.Debug("Resuming from existing temp file")
	} else {
		outFile, err = os.Create(tmpPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		downloaderLogger.Debug("Created new temp file")
	}
	defer func() { _ = outFile.Close() }()

	currentInfo, _ := outFile.Stat()
	downloaded := currentInfo.Size()
	downloaderLogger.Debug("Already downloaded", map[string]interface{}{
		"bytes": downloaded,
	})

	downloaderLogger.Debug("Detecting total file size")
	totalSize, err := d.detectTotalSize(ctx, urlStr)
	if err != nil {
		downloaderLogger.Warn("Could not determine total size", map[string]interface{}{
			"error": err,
		})
		downloaderLogger.Debug("Will download without size information")
		totalSize = 0
	} else {
		downloaderLogger.Debug("Total size detected", map[string]interface{}{
			"bytes": totalSize,
		})
	}

	for downloaded < totalSize || totalSize == 0 {
		start := downloaded
		end := int64(0)
		if totalSize > 0 {
			end = start + d.chunkSize - 1
			if end >= totalSize {
				end = totalSize - 1
			}
		} else {
			// When size unknown, prefer bounded first-chunk to probe and avoid 403
			end = start + d.chunkSize - 1
		}

		var resp *http.Response
		var lastErr error
		backoff := initialBackoffDuration
		for attempt := 0; attempt < d.maxRetries; attempt++ {
			req, _ := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
			req.Header.Set(headerUserAgent, userAgentValue)
			req.Header.Set(headerAccept, "*/*")
			req.Header.Set(headerAcceptEncoding, "identity")
			req.Header.Set(headerConnection, "keep-alive")
			req.Header.Set(headerCacheControl, "no-cache")
			if !isGoogleVideoHost(urlStr) {
				req.Header.Set(headerAcceptLanguage, "en-US,en;q=0.9")
			}

			rangeVal := fmt.Sprintf("bytes=%d-%d", start, end)
			req.Header.Set(headerRange, rangeVal)
			downloaderLogger.Trace("Requesting range", map[string]interface{}{
				"range": rangeVal,
			})

			downloaderLogger.Trace("Request headers", map[string]interface{}{
				"headers": req.Header,
			})

			resp, lastErr = d.Client.Do(req)
			if lastErr == nil && resp != nil && resp.StatusCode >= successMinHTTPStatusCode && resp.StatusCode < successMaxHTTPStatusExclusive {
				downloaderLogger.Trace("Request successful", map[string]interface{}{
					"status_code": resp.StatusCode,
					"headers":     resp.Header,
				})
				break
			}
			if resp != nil {
				downloaderLogger.Warn("Request failed", map[string]interface{}{
					"status_code": resp.StatusCode,
					"headers":     resp.Header,
				})
				if resp.Body != nil {
					body, _ := io.ReadAll(resp.Body)
					downloaderLogger.Trace("Response body", map[string]interface{}{
						"body": string(body),
					})
					_ = resp.Body.Close()
				}
				lastErr = fmt.Errorf("HTTP status %d", resp.StatusCode)
			}
			downloaderLogger.Debug("Request failed, retrying", map[string]interface{}{
				"attempt": attempt + 1,
				"error":   lastErr,
			})
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

		downloaderLogger.Trace("Starting to copy response body")
		buf := make([]byte, copyBufferSizeBytes)
		totalRead := int64(0)
		for {
			n, rerr := resp.Body.Read(buf)
			if n > 0 {
				if _, werr := outFile.Write(buf[:n]); werr != nil {
					_ = resp.Body.Close()
					return fmt.Errorf("failed to write chunk: %v", werr)
				}
				downloaded += int64(n)
				totalRead += int64(n)
				if d.ProgressFunc != nil {
					p := Progress{TotalSize: totalSize, DownloadedSize: downloaded}
					if totalSize > 0 {
						p.Percent = float64(downloaded) / float64(totalSize) * 100
					}
					d.ProgressFunc(p)
				}
				d.sleepForRate(int64(n))
			}
			if rerr == io.EOF {
				downloaderLogger.Trace("Response body completed", map[string]interface{}{
					"bytes_read": totalRead,
				})
				break
			}
			if rerr != nil {
				_ = resp.Body.Close()
				return fmt.Errorf("failed to read response body: %v", rerr)
			}
		}
		_ = resp.Body.Close()

		if totalSize == 0 {
			// We do not know size; continue bounded chunks until server closes or 206 signals end
			continue
		}
		if downloaded >= totalSize {
			break
		}
	}

	if fi, err := os.Stat(tmpPath); err == nil {
		if fi.Size() == 0 {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("empty download: 0 bytes written")
		}
	}

	return os.Rename(tmpPath, outputPath)
}
