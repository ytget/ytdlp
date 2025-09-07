package ytdlp

import (
	"context"
	"crypto/sha1"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ytget/ytdlp/client"
	"github.com/ytget/ytdlp/downloader"
	"github.com/ytget/ytdlp/errs"
	"github.com/ytget/ytdlp/internal/botguard"
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
	FormatSelector  string
	DesiredExt      string
	OutputPath      string
	HTTPClient      *http.Client
	ProgressFunc    func(Progress)
	RateLimitBps    int64
	ITClientName    string
	ITClientVersion string
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
	bg      struct {
		solver botguard.Solver
		mode   botguard.Mode
		cache  botguard.Cache
		debug  bool
		ttl    time.Duration
	}
}

// startPprofServer starts a pprof server for debugging
func startPprofServer() {
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

		log.Printf("Starting pprof server on :6060")
		if err := http.ListenAndServe(":6060", mux); err != nil {
			log.Printf("pprof server error: %v", err)
		}
	}()
}

// New creates a new Downloader instance with default options.
func New() *Downloader {
	if os.Getenv("YTDLP_PPROF") == "1" {
		startPprofServer()
	}
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

// WithInnertubeClient sets the Innertube client name and version to use.
func (d *Downloader) WithInnertubeClient(name, version string) *Downloader {
	d.options.ITClientName = strings.TrimSpace(name)
	d.options.ITClientVersion = strings.TrimSpace(version)
	return d
}

// WithBotguard configures Botguard attestation usage.
func (d *Downloader) WithBotguard(mode botguard.Mode, solver botguard.Solver, cache botguard.Cache) *Downloader {
	d.bg.mode = mode
	d.bg.solver = solver
	d.bg.cache = cache
	return d
}

// WithBotguardDebug enables Botguard debug logging.
func (d *Downloader) WithBotguardDebug(debug bool) *Downloader {
	d.bg.debug = debug
	return d
}

// WithBotguardTTL sets default Botguard TTL when solver does not specify ExpiresAt.
func (d *Downloader) WithBotguardTTL(ttl time.Duration) *Downloader {
	d.bg.ttl = ttl
	return d
}

// ResolveURL performs the metadata fetch and URL resolution, returning the final media URL and basic info.
func (d *Downloader) ResolveURL(ctx context.Context, videoURL string) (string, *VideoInfo, error) {
	log.Printf("Starting resolve for URL: %s", videoURL)

	// Extract video ID from URL
	videoID, err := extractVideoID(videoURL)
	if err != nil {
		return "", nil, fmt.Errorf("extract video id failed: %v", err)
	}
	log.Printf("Extracted video ID: %s", videoID)

	// Create HTTP client with HTTP/1.1 transport
	httpClient := client.New()
	if d.options.HTTPClient != nil {
		httpClient.HTTPClient = d.options.HTTPClient
		if transport, ok := httpClient.HTTPClient.Transport.(*http.Transport); ok {
			transport.ForceAttemptHTTP2 = false
		}
	} else {
		httpClient.HTTPClient = &http.Client{
			Transport: &http.Transport{ForceAttemptHTTP2: false, MaxIdleConns: 100, IdleConnTimeout: 90 * time.Second},
			Timeout:   30 * time.Second,
		}
	}

	// Fetch player response via Innertube
	itClient := innertube.New(httpClient.HTTPClient)
	itClient.WithBotguard(d.bg.solver, d.bg.mode, d.bg.cache).WithBotguardDebug(d.bg.debug).WithBotguardTTL(d.bg.ttl)
	name := strings.TrimSpace(d.options.ITClientName)
	ver := strings.TrimSpace(d.options.ITClientVersion)
	if name == "" {
		name = "ANDROID"
	}
	if ver == "" {
		ver = "20.10.38"
	}
	itClient.WithClient(name, ver)
	playerResponse, err := itClient.GetPlayerResponse(videoID)
	if err != nil {
		return "", nil, fmt.Errorf("get player response failed: %v", err)
	}
	log.Printf("Video metadata received, title: %s", playerResponse.VideoDetails.Title)

	// Map playability
	s := strings.ToUpper(playerResponse.PlayabilityStatus.Status)
	reason := strings.ToLower(playerResponse.PlayabilityStatus.Reason)
	switch s {
	case "ERROR":
		if strings.Contains(reason, "geograph") || strings.Contains(reason, "available in your country") {
			return "", nil, errs.ErrGeoBlocked
		}
		if strings.Contains(reason, "rate limit") || strings.Contains(reason, "quota") {
			return "", nil, errs.ErrRateLimited
		}
		return "", nil, errs.ErrVideoUnavailable
	case "LOGIN_REQUIRED":
		return "", nil, errs.ErrAgeRestricted
	case "UNPLAYABLE":
		if strings.Contains(reason, "private") {
			return "", nil, errs.ErrPrivate
		}
		return "", nil, errs.ErrVideoUnavailable
	}

	// Parse formats and select
	availableFormats, err := formats.ParseFormats(playerResponse)
	if err != nil {
		return "", nil, fmt.Errorf("parse formats failed: %v", err)
	}
	selectedFormat := formats.SelectFormat(availableFormats, d.options.FormatSelector, d.options.DesiredExt)
	if selectedFormat == nil {
		return "", nil, fmt.Errorf("no suitable format found")
	}

	// Resolve final URL
	finalURL := selectedFormat.URL
	var playerJSURL string
	if strings.TrimSpace(finalURL) == "" || strings.Contains(finalURL, "&n=") || strings.Contains(finalURL, "?n=") {
		pjsURL, perr := cipher.FetchPlayerJS(httpClient.HTTPClient, videoURL)
		if perr != nil {
			return "", nil, fmt.Errorf("fetch player.js url failed: %v", perr)
		}
		playerJSURL = pjsURL
		// Optional debug
		if body, src, gerr := cipher.DebugGetPlayerJS(httpClient.HTTPClient, playerJSURL); gerr == nil {
			h := sha1.Sum(body)
			_ = src
			_ = h
		}
		u, rerr := formats.ResolveFormatURL(httpClient.HTTPClient, *selectedFormat, playerJSURL)
		if rerr != nil {
			return "", nil, fmt.Errorf("resolve selected format url failed: %v", rerr)
		}
		finalURL = u
	}

	info := &VideoInfo{ID: videoID, Title: playerResponse.VideoDetails.Title, Formats: availableFormats, Description: ""}
	return finalURL, info, nil
}

// Download retrieves video metadata, resolves URL, and downloads to disk.
func (d *Downloader) Download(ctx context.Context, videoURL string) (*VideoInfo, error) {
	log.Printf("Starting download for URL: %s", videoURL)

	finalURL, info, err := d.ResolveURL(ctx, videoURL)
	if err != nil {
		return nil, err
	}

	// 6. Download video
	log.Printf("Starting video download...")
	log.Printf("Final media URL: %s", finalURL)
	dl := downloader.New(d.options.HTTPClient, func(p downloader.Progress) {
		if d.options.ProgressFunc != nil {
			d.options.ProgressFunc(Progress{TotalSize: p.TotalSize, DownloadedSize: p.DownloadedSize, Percent: p.Percent})
		}
	}, d.options.RateLimitBps)
	outputPath := d.options.OutputPath
	if outputPath == "" {
		// derive extension from mime using helper
		// try to infer extension from selected format if available
		var chosen types.Format
		if len(info.Formats) > 0 {
			for _, f := range info.Formats {
				if strings.Contains(finalURL, strconv.Itoa(f.Itag)) {
					chosen = f
					break
				}
			}
		}
		ext := mimeext.ExtFromMime(chosen.MimeType)
		title := info.Title
		if strings.TrimSpace(title) == "" {
			title = "video"
		}
		outputPath = internalSanitize.ToSafeFilename(title, ext)
	} else {
		// if outputPath is a directory, derive a safe filename and join
		if fi, statErr := os.Stat(outputPath); statErr == nil && fi.IsDir() {
			var chosen types.Format
			if len(info.Formats) > 0 {
				for _, f := range info.Formats {
					if strings.Contains(finalURL, strconv.Itoa(f.Itag)) {
						chosen = f
						break
					}
				}
			}
			ext := mimeext.ExtFromMime(chosen.MimeType)
			title := info.Title
			if strings.TrimSpace(title) == "" {
				title = "video"
			}
			name := internalSanitize.ToSafeFilename(title, ext)
			outputPath = filepath.Join(outputPath, name)
		}
	}

	if err := dl.Download(ctx, finalURL, outputPath); err != nil {
		return nil, fmt.Errorf("download failed: %v", err)
	}

	return info, nil
}

// GetPlaylistItems returns minimal playlist items for a playlist ID (MVP: first page only).
func (d *Downloader) GetPlaylistItems(ctx context.Context, playlistID string, limit int) ([]types.PlaylistItem, error) {
	// Create HTTP client
	httpClient := client.New()
	if d.options.HTTPClient != nil {
		httpClient.HTTPClient = d.options.HTTPClient
	}
	itClient := innertube.New(httpClient.HTTPClient)
	itClient.WithBotguard(d.bg.solver, d.bg.mode, d.bg.cache).WithBotguardDebug(d.bg.debug).WithBotguardTTL(d.bg.ttl)
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
	itClient.WithBotguard(d.bg.solver, d.bg.mode, d.bg.cache).WithBotguardDebug(d.bg.debug).WithBotguardTTL(d.bg.ttl)
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
