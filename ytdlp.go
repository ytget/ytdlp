package ytdlp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ytget/ytdlp/client"
	"github.com/ytget/ytdlp/downloader"
	"github.com/ytget/ytdlp/errs"
	"github.com/ytget/ytdlp/internal/mimeext"
	internalSanitize "github.com/ytget/ytdlp/internal/sanitize"
	"github.com/ytget/ytdlp/types"
	"github.com/ytget/ytdlp/youtube/cipher"
	"github.com/ytget/ytdlp/youtube/formats"
	"github.com/ytget/ytdlp/youtube/innertube"
)

// VideoInfo contains basic video metadata and the full list of available formats.
type VideoInfo struct {
	ID          string
	Title       string
	Author      string
	Duration    int
	Formats     []types.Format
	Description string
}

// Format describes an available media format.
// Deprecated: use types.Format instead.
type Format = types.Format

// DownloadOptions contains configuration for a single download invocation.
//
// Use chainable setters on Downloader to populate these options.
type DownloadOptions struct {
	FormatSelector string
	DesiredExt     string
	OutputPath     string
	HTTPClient     *http.Client
	ProgressFunc   func(Progress)
	RateLimitBps   int64
}

// Progress describes current progress of an ongoing download.
type Progress struct {
	TotalSize      int64
	DownloadedSize int64
	Percent        float64
}

// Downloader provides a high-level API for retrieving metadata and downloading
// YouTube videos using internal clients and helpers.
type Downloader struct {
	options DownloadOptions
}

// New creates a new Downloader instance with default options.
func New() *Downloader {
	return &Downloader{}
}

// WithFormat sets a format selector and optional desired extension.
// Examples: "itag=22", "best", "height<=480". Extension is case-insensitive.
func (d *Downloader) WithFormat(quality, ext string) *Downloader {
	d.options.FormatSelector = quality
	d.options.DesiredExt = strings.TrimPrefix(strings.ToLower(ext), ".")
	return d
}

// WithHTTPClient sets a custom HTTP client to be used for all network calls.
func (d *Downloader) WithHTTPClient(client *http.Client) *Downloader {
	d.options.HTTPClient = client
	return d
}

// WithProgress registers a callback that receives progress updates.
func (d *Downloader) WithProgress(f func(Progress)) *Downloader {
	d.options.ProgressFunc = f
	return d
}

// WithOutputPath sets the output file path. If empty, a safe filename is derived
// from the video title and mime extension. If a directory path is provided, a
// safe filename is derived and placed inside that directory.
func (d *Downloader) WithOutputPath(path string) *Downloader {
	d.options.OutputPath = path
	return d
}

// WithRateLimit sets a download rate limit in bytes per second. Zero disables limiting.
func (d *Downloader) WithRateLimit(bytesPerSecond int64) *Downloader {
	if bytesPerSecond < 0 {
		bytesPerSecond = 0
	}
	d.options.RateLimitBps = bytesPerSecond
	return d
}

// Download retrieves video metadata and downloads the selected format to disk.
// It performs: metadata fetch, player.js resolution, signature deciphering,
// format selection, and a chunked download with retries.
func (d *Downloader) Download(ctx context.Context, videoURL string) (*VideoInfo, error) {
	// Extract video ID from URL
	videoID, err := extractVideoID(videoURL)
	if err != nil {
		return nil, fmt.Errorf("extract video id failed: %v", err)
	}

	// 1. Create HTTP client
	httpClient := client.New()
	if d.options.HTTPClient != nil {
		httpClient.HTTPClient = d.options.HTTPClient
	}

	// 2. Fetch video metadata
	itClient := innertube.New(httpClient.HTTPClient)
	playerResponse, err := itClient.GetPlayerResponse(videoID)
	if err != nil {
		return nil, fmt.Errorf("get player response failed: %v", err)
	}

	// Check playability status and map errors
	s := strings.ToUpper(playerResponse.PlayabilityStatus.Status)
	reason := strings.ToLower(playerResponse.PlayabilityStatus.Reason)
	switch s {
	case "ERROR":
		if strings.Contains(reason, "geograph") || strings.Contains(reason, "available in your country") {
			return nil, errs.ErrGeoBlocked
		}
		if strings.Contains(reason, "rate limit") || strings.Contains(reason, "quota") {
			return nil, errs.ErrRateLimited
		}
		return nil, errs.ErrVideoUnavailable
	case "LOGIN_REQUIRED":
		return nil, errs.ErrAgeRestricted
	case "UNPLAYABLE":
		if strings.Contains(reason, "private") {
			return nil, errs.ErrPrivate
		}
		return nil, errs.ErrVideoUnavailable
	}

	// 3. Fetch player.js URL
	playerJSURL, err := cipher.FetchPlayerJS(httpClient.HTTPClient, videoURL)
	if err != nil {
		return nil, fmt.Errorf("fetch player.js url failed: %v", err)
	}

	// 4. Parse formats
	availableFormats, err := formats.ParseFormats(playerResponse)
	if err != nil {
		return nil, fmt.Errorf("parse formats failed: %v", err)
	}

	// 5. Decrypt signatures (if needed)
	if err := formats.DecryptSignatures(httpClient.HTTPClient, availableFormats, playerJSURL); err != nil {
		return nil, fmt.Errorf("decrypt signatures failed: %v", err)
	}

	// 6. Select a suitable format
	desiredExt := d.options.DesiredExt
	selectedFormat := formats.SelectFormat(availableFormats, d.options.FormatSelector, desiredExt)
	if selectedFormat == nil {
		return nil, fmt.Errorf("no suitable format found")
	}

	// 7. Download video
	dl := downloader.New(httpClient.HTTPClient, func(p downloader.Progress) {
		if d.options.ProgressFunc != nil {
			d.options.ProgressFunc(Progress{TotalSize: p.TotalSize, DownloadedSize: p.DownloadedSize, Percent: p.Percent})
		}
	}, d.options.RateLimitBps)
	outputPath := d.options.OutputPath
	if outputPath == "" {
		// derive extension from mime using helper
		ext := mimeext.ExtFromMime(selectedFormat.MimeType)
		title := playerResponse.VideoDetails.Title
		if strings.TrimSpace(title) == "" {
			title = strconv.Itoa(selectedFormat.Itag)
		}
		outputPath = internalSanitize.ToSafeFilename(title, ext)
	} else {
		// if outputPath is a directory, derive a safe filename and join
		if fi, statErr := os.Stat(outputPath); statErr == nil && fi.IsDir() {
			ext := mimeext.ExtFromMime(selectedFormat.MimeType)
			title := playerResponse.VideoDetails.Title
			if strings.TrimSpace(title) == "" {
				title = strconv.Itoa(selectedFormat.Itag)
			}
			name := internalSanitize.ToSafeFilename(title, ext)
			outputPath = filepath.Join(outputPath, name)
		}
	}

	if err := dl.Download(ctx, selectedFormat.URL, outputPath); err != nil {
		return nil, fmt.Errorf("download failed: %v", err)
	}

	// 8. Build result
	videoInfo := &VideoInfo{
		ID:          videoID,
		Title:       playerResponse.VideoDetails.Title,
		Formats:     availableFormats,
		Description: "",
	}

	return videoInfo, nil
}

// GetPlaylistItems returns minimal playlist items for a playlist ID (MVP: first page only).
func (d *Downloader) GetPlaylistItems(ctx context.Context, playlistID string, limit int) ([]types.PlaylistItem, error) {
	// Create HTTP client
	httpClient := client.New()
	if d.options.HTTPClient != nil {
		httpClient.HTTPClient = d.options.HTTPClient
	}
	itClient := innertube.New(httpClient.HTTPClient)
	items, err := itClient.GetPlaylistItems(playlistID, limit)
	return items, err
}

// GetPlaylistItemsAll returns playlist items with continuations up to the limit.
func (d *Downloader) GetPlaylistItemsAll(ctx context.Context, playlistID string, limit int) ([]types.PlaylistItem, error) {
	httpClient := client.New()
	if d.options.HTTPClient != nil {
		httpClient.HTTPClient = d.options.HTTPClient
	}
	itClient := innertube.New(httpClient.HTTPClient)
	return itClient.GetPlaylistItemsAll(playlistID, limit)
}

func extractVideoID(videoURL string) (string, error) {
	u, err := url.Parse(videoURL)
	if err != nil {
		return "", err
	}
	if u.Host == "youtu.be" {
		return strings.TrimPrefix(u.Path, "/"), nil
	}
	if u.Host == "youtube.com" || u.Host == "www.youtube.com" {
		if strings.HasPrefix(u.Path, "/watch") {
			return u.Query().Get("v"), nil
		}
	}
	return "", fmt.Errorf("invalid youtube url")
}
