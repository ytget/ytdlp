package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/ytget/ytdlp"
	"github.com/ytget/ytdlp/client"
	"github.com/ytget/ytdlp/internal/botguard"
)

func main() {
	// Start pprof HTTP server
	go func() {
		log.Println("Starting pprof server on :6060")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	// Enable CPU profiling
	cpuProfile, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	if err := pprof.StartCPUProfile(cpuProfile); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
	defer pprof.StopCPUProfile()

	var (
		flagFormat       string
		flagExt          string
		flagOutput       string
		flagNoProgress   bool
		flagTimeout      time.Duration
		flagRetries      int
		flagUA           string
		flagProxy        string
		flagRateLimit    string
		flagPlaylist     bool
		flagLimit        int
		flagConcurrency  int
		flagBGMode       string
		flagBGDebug      bool
		flagBGCacheMode  string
		flagBGCacheDir   string
		flagBGCacheTTL   time.Duration
		flagBGScriptPath string
		flagClientName   string
		flagClientVer    string
		flagPrintURL     bool
	)

	flag.StringVar(&flagFormat, "format", "", "Format selector (e.g., 'itag=22', 'best', 'height<=480')")
	flag.StringVar(&flagExt, "ext", "", "Desired extension (e.g., 'mp4', 'webm')")
	flag.StringVar(&flagOutput, "output", "", "Output path (file or directory). Empty derives from title + MIME")
	flag.BoolVar(&flagNoProgress, "no-progress", false, "Disable progress output")
	flag.DurationVar(&flagTimeout, "http-timeout", 30*time.Second, "HTTP timeout (e.g., 30s, 1m)")
	flag.IntVar(&flagRetries, "retries", 3, "HTTP retries for transient errors")
	flag.StringVar(&flagUA, "ua", "", "Override User-Agent header")
	flag.StringVar(&flagProxy, "proxy", "", "Proxy URL (http/https/socks)")
	flag.StringVar(&flagRateLimit, "rate-limit", "", "Download rate limit (e.g., 2MiB/s, 500KiB/s)")
	flag.BoolVar(&flagPlaylist, "playlist", false, "Treat input as playlist URL or ID")
	flag.IntVar(&flagLimit, "limit", 0, "Max items to process for playlist (0 means all)")
	flag.IntVar(&flagConcurrency, "concurrency", 1, "Parallelism for playlist downloads")
	flag.StringVar(&flagBGMode, "botguard", "off", "Botguard mode: off|auto|force")
	flag.BoolVar(&flagBGDebug, "debug-botguard", false, "Enable Botguard debug logs")
	flag.StringVar(&flagBGCacheMode, "botguard-cache", "mem", "Botguard cache mode: mem|file")
	flag.StringVar(&flagBGCacheDir, "botguard-cache-dir", "", "Botguard cache directory (for file mode)")
	flag.DurationVar(&flagBGCacheTTL, "botguard-ttl", 30*time.Minute, "Default Botguard token TTL if solver doesn't set")
	flag.StringVar(&flagBGScriptPath, "botguard-script", "", "Path to JS script implementing bgAttest(input)")
	flag.StringVar(&flagClientName, "client-name", "", "Innertube client name (default ANDROID)")
	flag.StringVar(&flagClientVer, "client-version", "", "Innertube client version (default 20.10.38)")
	flag.BoolVar(&flagPrintURL, "g", false, "Print final media URL and exit (no download)")
	flag.BoolVar(&flagPrintURL, "print-url", false, "Print final media URL and exit (no download)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <video_or_playlist_url>\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "\nFlags:")
		flag.PrintDefaults()
	}

	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(2)
	}

	input := strings.TrimSpace(args[0])

	// Build client config
	cfg := client.Config{Timeout: flagTimeout, Retries: flagRetries, UserAgent: flagUA, ProxyURL: flagProxy}
	c := client.NewWith(cfg)

	// Determine Botguard mode and initialize solver/cache
	var bgMode botguard.Mode
	switch strings.ToLower(strings.TrimSpace(flagBGMode)) {
	case "force":
		bgMode = botguard.Force
	case "auto":
		bgMode = botguard.Auto
	default:
		bgMode = botguard.Off
	}
	var solver *botguard.GojaSolver
	if strings.TrimSpace(flagBGScriptPath) != "" {
		solver = botguard.NewGojaSolverWithScript(flagBGScriptPath)
	} else {
		solver = botguard.NewGojaSolver()
	}

	var cache botguard.Cache
	switch strings.ToLower(strings.TrimSpace(flagBGCacheMode)) {
	case "file":
		root := flagBGCacheDir
		if root == "" {
			root = filepath.Join(os.TempDir(), "ytdlp-bg-cache")
		}
		if fc, err := botguard.NewFileCache(root); err == nil {
			cache = fc
		} else {
			cache = botguard.NewMemoryCache()
		}
	default:
		cache = botguard.NewMemoryCache()
	}

	if flagPlaylist {
		playlistID, err := parsePlaylistID(input)
		if err != nil || playlistID == "" {
			fmt.Fprintf(os.Stderr, "Invalid playlist input: %v\n", err)
			os.Exit(2)
		}

		// Prepare output dir
		outDir := flagOutput
		if outDir == "" {
			outDir = "."
		}
		if !isDir(outDir) {
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create output dir: %v\n", err)
				os.Exit(1)
			}
		}

		d := ytdlp.New().WithHTTPClient(c.HTTPClient).
			WithBotguard(bgMode, solver, cache).
			WithBotguardDebug(flagBGDebug).
			WithBotguardTTL(flagBGCacheTTL)
		if flagClientName != "" || flagClientVer != "" {
			d = d.WithInnertubeClient(flagClientName, flagClientVer)
		}
		items, err := d.GetPlaylistItemsAll(context.Background(), playlistID, flagLimit)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fetch playlist: %v\n", err)
			os.Exit(1)
		}
		if len(items) == 0 {
			fmt.Fprintln(os.Stderr, "No items in playlist")
			return
		}
		if flagConcurrency < 1 {
			flagConcurrency = 1
		}

		jobs := make(chan int, len(items))
		var wg sync.WaitGroup
		wg.Add(flagConcurrency)
		for w := 0; w < flagConcurrency; w++ {
			go func() {
				defer wg.Done()
				localD := ytdlp.New().WithHTTPClient(c.HTTPClient).
					WithBotguard(bgMode, solver, cache).
					WithBotguardDebug(flagBGDebug).
					WithBotguardTTL(flagBGCacheTTL)
				if flagClientName != "" || flagClientVer != "" {
					localD = localD.WithInnertubeClient(flagClientName, flagClientVer)
				}
				if flagFormat != "" || flagExt != "" {
					localD = localD.WithFormat(flagFormat, flagExt)
				}
				if bps := parseRate(flagRateLimit); bps > 0 {
					localD = localD.WithRateLimit(bps)
				}
				if !flagNoProgress && flagConcurrency == 1 {
					localD = localD.WithProgress(func(p ytdlp.Progress) {
						if p.TotalSize > 0 {
							_, _ = fmt.Fprintf(os.Stdout, "Downloaded %.1f%%\r", p.Percent)
						}
					})
				}
				for idx := range jobs {
					item := items[idx]
					videoURL := "https://www.youtube.com/watch?v=" + item.VideoID
					_, _ = fmt.Fprintf(os.Stdout, "Downloading [%d/%d] %s...\n", idx+1, len(items), item.Title)
					localOut := flagOutput
					if localOut != "" && isDir(localOut) {
						localOut = filepath.Join(localOut, "") // directory; library will derive filename
					}
					if localOut != "" {
						localD = localD.WithOutputPath(localOut)
					}
					if _, err := localD.Download(context.Background(), videoURL); err != nil {
						fmt.Fprintf(os.Stderr, "Error downloading %s: %v\n", item.Title, err)
					} else {
						_, _ = fmt.Fprintf(os.Stdout, "Done: %s\n", item.Title)
					}
				}
			}()
		}
		for i := range items {
			jobs <- i
		}
		close(jobs)
		wg.Wait()
		return
	}

	d := ytdlp.New().WithHTTPClient(c.HTTPClient).
		WithBotguard(bgMode, solver, cache).
		WithBotguardDebug(flagBGDebug).
		WithBotguardTTL(flagBGCacheTTL)
	if flagClientName != "" || flagClientVer != "" {
		d = d.WithInnertubeClient(flagClientName, flagClientVer)
	}
	if flagFormat != "" || flagExt != "" {
		d = d.WithFormat(flagFormat, flagExt)
	}
	if flagOutput != "" {
		d = d.WithOutputPath(flagOutput)
	}
	if !flagNoProgress && !flagPrintURL {
		d = d.WithProgress(func(p ytdlp.Progress) {
			if p.TotalSize > 0 {
				_, _ = fmt.Fprintf(os.Stdout, "Downloaded %.1f%%\r", p.Percent)
			}
		})
	}
	if bps := parseRate(flagRateLimit); bps > 0 {
		d = d.WithRateLimit(bps)
	}

	if flagPrintURL {
		finalURL, info, err := d.ResolveURL(context.Background(), input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		_, _ = fmt.Fprintln(os.Stdout, finalURL)
		_ = info
		return
	}

	info, err := d.Download(context.Background(), input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nSaved: %s\n", info.Title)
}

// parseRate parses strings like "2MiB/s", "500KiB/s" into bytes per second.
func parseRate(s string) int64 {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0
	}
	// Very small parser: accept numbers with optional KiB/MiB/GiB suffix and optional /S
	mul := int64(1)
	s = strings.TrimSuffix(s, "/S")
	s = strings.TrimSpace(s)
	sfx := ""
	for _, suf := range []string{"KIB", "MIB", "GIB", "KB", "MB", "GB"} {
		if strings.HasSuffix(s, suf) {
			sfx = suf
			s = strings.TrimSuffix(s, suf)
			break
		}
	}
	s = strings.TrimSpace(s)
	var val float64
	_, err := fmt.Sscanf(s, "%f", &val)
	if err != nil || val <= 0 {
		return 0
	}
	switch sfx {
	case "KIB":
		mul = 1024
	case "MIB":
		mul = 1024 * 1024
	case "GIB":
		mul = 1024 * 1024 * 1024
	case "KB":
		mul = 1000
	case "MB":
		mul = 1000 * 1000
	case "GB":
		mul = 1000 * 1000 * 1000
	}
	return int64(val * float64(mul))
}

func parsePlaylistID(input string) (string, error) {
	// Accept raw playlist IDs as-is
	if input != "" && (strings.HasPrefix(input, "PL") || strings.HasPrefix(input, "UU") || strings.HasPrefix(input, "OLAK5uy_")) {
		return input, nil
	}
	u, err := url.Parse(input)
	if err != nil {
		return "", err
	}
	if id := u.Query().Get("list"); id != "" {
		return id, nil
	}
	return "", fmt.Errorf("playlist id not found")
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
