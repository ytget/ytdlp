package innertube

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/ytget/ytdlp/internal/botguard"
	"github.com/ytget/ytdlp/types"
)

var (
	playerURL = "https://www.youtube.com/youtubei/v1/player"
	browseURL = "https://www.youtube.com/youtubei/v1/browse"
)

const (
	ytBase                = "https://www.youtube.com"
	userAgentValue        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"
	headerContentTypeJSON = "application/json"
	clientNameWEB         = "WEB"
	defaultClientVersion  = "2.20250312.04.00"
	browseIDPrefix        = "VL"
	defaultPlaylistLimit  = 100
	continuationLimitMax  = 1 << 20
	visitorIdMaxAge       = 10 * time.Hour
)

var (
	apiKeyRe    = regexp.MustCompile(`"INNERTUBE_API_KEY":"([^"]+)"`)
	clientVerRe = regexp.MustCompile(`"INNERTUBE_CLIENT_VERSION":"([^"]+)"`)
)

// clientCodeFromName returns X-YouTube-Client-Name numeric code for known clients
func clientCodeFromName(name string) string {
	switch strings.ToUpper(name) {
	case "WEB":
		return "1"
	case "MWEB":
		return "2"
	case "ANDROID":
		return "3"
	case "IOS":
		return "5"
	case "TVHTML5":
		return "7"
	case "WEB_EMBEDDED_PLAYER":
		return "56"
	case "WEB_CREATOR":
		return "62"
	case "WEB_REMIX":
		return "67"
	case "TVHTML5_SIMPLY":
		return "75"
	case "TVHTML5_SIMPLY_EMBEDDED_PLAYER":
		return "85"
	default:
		return ""
	}
}

// Client for interacting with the YouTube InnerTube API.
type Client struct {
	HTTPClient *http.Client
	apiKey     string
	clientVer  string
	clientName string
	visitorId  struct {
		value   string
		updated time.Time
	}
	// Optional Botguard integration
	bg struct {
		solver botguard.Solver
		mode   botguard.Mode
		cache  botguard.Cache
		ttl    time.Duration
		debug  bool
	}
}

// New creates a new InnerTube client with HTTP/1.1 transport.
func New(httpClient *http.Client) *Client {
	// Force HTTP/1.1 to avoid HTTP/2 handshake issues
	if httpClient == nil {
		httpClient = &http.Client{
			Transport: &http.Transport{
				ForceAttemptHTTP2:     true, // Enable HTTP/2
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   10,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				DisableCompression:    false, // Enable compression
				ReadBufferSize:        16 * 1024,
				WriteBufferSize:       16 * 1024,
			},
			Timeout: 30 * time.Second,
		}
	} else if httpClient.Transport != nil {
		// If custom transport exists, ensure HTTP/2 and compression are enabled
		if transport, ok := httpClient.Transport.(*http.Transport); ok {
			transport.ForceAttemptHTTP2 = true
			transport.DisableCompression = false
			transport.MaxIdleConnsPerHost = 10
			transport.TLSHandshakeTimeout = 10 * time.Second
			transport.ExpectContinueTimeout = 1 * time.Second
			transport.ResponseHeaderTimeout = 10 * time.Second
			transport.ReadBufferSize = 16 * 1024
			transport.WriteBufferSize = 16 * 1024
		}
	}

	return &Client{HTTPClient: httpClient, clientName: clientNameWEB}
}

// WithClient overrides InnerTube client name/version to shape playback URLs.
func (c *Client) WithClient(name, version string) *Client {
	if strings.TrimSpace(name) != "" {
		c.clientName = name
	}
	if strings.TrimSpace(version) != "" {
		c.clientVer = version
	}
	return c
}

// WithBotguard configures an optional Botguard solver and mode.
func (c *Client) WithBotguard(solver botguard.Solver, mode botguard.Mode, cache botguard.Cache) *Client {
	c.bg.solver = solver
	c.bg.mode = mode
	c.bg.cache = cache
	return c
}

// WithBotguardDebug enables Botguard debug logging.
func (c *Client) WithBotguardDebug(debug bool) *Client {
	c.bg.debug = debug
	return c
}

// WithBotguardTTL sets a default TTL to apply when solver does not specify ExpiresAt.
func (c *Client) WithBotguardTTL(ttl time.Duration) *Client {
	c.bg.ttl = ttl
	return c
}

// PlayerResponse represents a response from the InnerTube /player endpoint.
type PlayerResponse struct {
	StreamingData struct {
		Formats         []any `json:"formats"`
		AdaptiveFormats []any `json:"adaptiveFormats"`
	} `json:"streamingData"`
	VideoDetails struct {
		Title string `json:"title"`
	} `json:"videoDetails"`
	PlayabilityStatus struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	} `json:"playabilityStatus"`
}

func (c *Client) ensureKey(videoOrPlaylistID string, isPlaylist bool) {
	if c.apiKey != "" && c.clientVer != "" {
		return
	}

	// Try multiple sources for API key and client version
	sources := []string{}

	// Add video/playlist specific URL
	if isPlaylist {
		sources = append(sources, ytBase+"/playlist?list="+videoOrPlaylistID)
	} else {
		sources = append(sources, ytBase+"/watch?v="+videoOrPlaylistID)
	}

	// Add fallback sources
	sources = append(sources, ytBase, ytBase+"/feed/trending", ytBase+"/feed/explore")

	for _, source := range sources {
		if c.apiKey != "" && c.clientVer != "" {
			break
		}

		req, err := http.NewRequest("GET", source, nil)
		if err != nil {
			continue
		}

		// Enhanced headers for better compatibility
		req.Header.Set("User-Agent", userAgentValue)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
		req.Header.Set("Accept-Encoding", "identity")
		req.Header.Set("DNT", "1")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "none")
		req.Header.Set("Sec-Fetch-User", "?1")
		req.Header.Set("Cache-Control", "max-age=0")

		resp, err := c.HTTPClient.Do(req)
		if err != nil || resp == nil {
			continue
		}

		defer func() { _ = resp.Body.Close() }()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		// Extract API key if not found yet
		if c.apiKey == "" {
			if m := apiKeyRe.FindSubmatch(body); len(m) == 2 {
				c.apiKey = string(m[1])
			}
		}

		// Extract client version if not found yet
		if c.clientVer == "" {
			if m := clientVerRe.FindSubmatch(body); len(m) == 2 {
				c.clientVer = string(m[1])
			}
		}
	}

	// Final fallback for client version
	if c.clientVer == "" {
		c.clientVer = defaultClientVersion
	}
}

// GetPlayerResponse fetches video data for the provided video ID using the
// InnerTube /player endpoint.
func (c *Client) GetPlayerResponse(videoID string) (*PlayerResponse, error) {
	c.ensureKey(videoID, false)
	if c.apiKey == "" {
		// Try one more time with different sources
		c.ensureKey(videoID, false)
		if c.apiKey == "" {
			return nil, errors.New("innertube: api key not found after multiple attempts")
		}
	}

	name := c.clientName
	ver := c.clientVer
	// If a custom client name is set and version missing, use minimal default
	if name != clientNameWEB && ver == defaultClientVersion {
		ver = "2.0"
	}

	clientMap := map[string]any{
		"clientName":    name,
		"clientVersion": ver,
	}
	// Enrich Android client context to match yt-dlp shape
	var reqUserAgent = userAgentValue
	if strings.EqualFold(name, "ANDROID") {
		clientMap["androidSdkVersion"] = 30
		clientMap["osName"] = "Android"
		clientMap["osVersion"] = "11"
		ua := "com.google.android.youtube/" + ver + " (Linux; U; Android 11) gzip"
		clientMap["userAgent"] = ua
		reqUserAgent = ua
	}

	requestBody, err := json.Marshal(map[string]any{
		"context": map[string]any{
			"client": clientMap,
		},
		"videoId": videoID,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", playerURL+"?key="+c.apiKey, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", headerContentTypeJSON)
	req.Header.Set("User-Agent", reqUserAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Referer", "https://www.youtube.com/")
	req.Header.Set("Origin", "https://www.youtube.com")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")
	// Set numeric client code when known
	if code := clientCodeFromName(name); code != "" {
		req.Header.Set("X-YouTube-Client-Name", code)
	}
	req.Header.Set("X-YouTube-Client-Version", ver)

	// Add visitor ID if available
	if visitorId, err := c.getVisitorId(); err == nil && visitorId != "" {
		req.Header.Set("x-goog-visitor-id", visitorId)
	}
	resp, err := c.doWithBotguardRetry(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Log response status and headers for debugging
	fmt.Printf("Response status: %d\n", resp.StatusCode)
	fmt.Printf("Response headers:\n")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %s\n", k, v)
	}

	// Handle compressed response
	var reader io.Reader = resp.Body
	switch strings.ToLower(resp.Header.Get("Content-Encoding")) {
	case "gzip":
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %v", err)
		}
		defer gzReader.Close()
		reader = gzReader
	case "br":
		reader = brotli.NewReader(resp.Body)
	case "deflate":
		// deflate is raw DEFLATE data, no wrapper
		reader = resp.Body
	case "bzip2":
		reader = bzip2.NewReader(resp.Body)
	}

	// Read decompressed response
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Log decompressed response for debugging
	fmt.Printf("Decompressed response body: %s\n", string(body))

	var playerResponse PlayerResponse
	if err := json.Unmarshal(body, &playerResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v\nBody: %s", err, string(body))
	}

	return &playerResponse, nil
}

// GetPlaylistItems fetches initial playlist items (without continuations, MVP).
func (c *Client) GetPlaylistItems(playlistID string, limit int) ([]types.PlaylistItem, error) {
	c.ensureKey(playlistID, true)
	if c.apiKey == "" {
		return nil, errors.New("innertube: api key not found")
	}
	if limit <= 0 {
		limit = defaultPlaylistLimit
	}

	clientMap := map[string]any{
		"clientName":    c.clientName,
		"clientVersion": c.clientVer,
	}
	var reqUserAgent = userAgentValue
	if strings.EqualFold(c.clientName, "ANDROID") {
		clientMap["androidSdkVersion"] = 30
		clientMap["osName"] = "Android"
		clientMap["osVersion"] = "11"
		ua := "com.google.android.youtube/" + c.clientVer + " (Linux; U; Android 11) gzip"
		clientMap["userAgent"] = ua
		reqUserAgent = ua
	}

	reqBody := map[string]any{
		"context": map[string]any{
			"client": clientMap,
		},
		"browseId": browseIDPrefix + playlistID,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", browseURL+"?key="+c.apiKey, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", headerContentTypeJSON)
	req.Header.Set("User-Agent", reqUserAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Referer", "https://www.youtube.com/")
	req.Header.Set("Origin", "https://www.youtube.com")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")
	if code := clientCodeFromName(c.clientName); code != "" {
		req.Header.Set("X-YouTube-Client-Name", code)
	}
	req.Header.Set("X-YouTube-Client-Version", c.clientVer)

	// Add visitor ID if available
	if visitorId, err := c.getVisitorId(); err == nil && visitorId != "" {
		req.Header.Set("x-goog-visitor-id", visitorId)
	}
	resp, err := c.doWithBotguardRetry(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var root any
	if err := json.Unmarshal(respBody, &root); err != nil {
		return nil, err
	}
	items := make([]types.PlaylistItem, 0, 50)
	collectPlaylistVideoRenderers(root, &items, limit)
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

// GetPlaylistItemsAll loads playlist items with continuations up to the specified limit.
func (c *Client) GetPlaylistItemsAll(playlistID string, limit int) ([]types.PlaylistItem, error) {
	items, err := c.GetPlaylistItems(playlistID, limit)
	if err != nil {
		return nil, err
	}
	if len(items) >= limit {
		return items, nil
	}

	// Fetch continuation tokens iteratively
	clientMap := map[string]any{
		"clientName":    c.clientName,
		"clientVersion": c.clientVer,
	}
	if strings.EqualFold(c.clientName, "ANDROID") {
		clientMap["androidSdkVersion"] = 30
		clientMap["osName"] = "Android"
		clientMap["osVersion"] = "11"
		clientMap["userAgent"] = "com.google.android.youtube/" + c.clientVer + " (Linux; U; Android 11) gzip"
	}
	reqBody := map[string]any{
		"context": map[string]any{
			"client": clientMap,
		},
		"browseId": browseIDPrefix + playlistID,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", browseURL+"?key="+c.apiKey, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", headerContentTypeJSON)
	req.Header.Set("User-Agent", userAgentValue)
	resp, err := c.doWithBotguardRetry(req)
	if err != nil {
		return items, nil
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	var root any
	_ = json.Unmarshal(respBody, &root)

	token := findFirstContinuationToken(root)
	for token != "" && len(items) < limit {
		more, next, _ := c.getPlaylistContinuation(token)
		items = append(items, more...)
		if len(items) >= limit {
			break
		}
		token = next
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (c *Client) getPlaylistContinuation(continuation string) ([]types.PlaylistItem, string, error) {
	if c.apiKey == "" {
		return nil, "", errors.New("innertube: api key not found")
	}
	reqBody := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"clientName":    c.clientName,
				"clientVersion": c.clientVer,
			},
		},
		"continuation": continuation,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequest("POST", browseURL+"?key="+c.apiKey, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", headerContentTypeJSON)
	req.Header.Set("User-Agent", userAgentValue)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Referer", "https://www.youtube.com/")
	req.Header.Set("Origin", "https://www.youtube.com")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")
	if code := clientCodeFromName(c.clientName); code != "" {
		req.Header.Set("X-YouTube-Client-Name", code)
	}
	req.Header.Set("X-YouTube-Client-Version", c.clientVer)

	// Add visitor ID if available
	if visitorId, err := c.getVisitorId(); err == nil && visitorId != "" {
		req.Header.Set("x-goog-visitor-id", visitorId)
	}
	resp, err := c.doWithBotguardRetry(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	var root any
	if err := json.Unmarshal(respBody, &root); err != nil {
		return nil, "", err
	}
	items := make([]types.PlaylistItem, 0, 50)
	collectPlaylistVideoRenderers(root, &items, continuationLimitMax)
	next := findFirstContinuationToken(root)
	return items, next, nil
}

func collectPlaylistVideoRenderers(node any, out *[]types.PlaylistItem, limit int) {
	if len(*out) >= limit {
		return
	}
	switch v := node.(type) {
	case map[string]any:
		if r, ok := v["playlistVideoRenderer"].(map[string]any); ok {
			var it types.PlaylistItem
			if s, ok := r["videoId"].(string); ok {
				it.VideoID = s
			}
			if idx, ok := r["index"].(map[string]any); ok {
				if simple, ok := idx["simpleText"].(string); ok {
					if n, err := strconv.Atoi(simple); err == nil {
						it.Index = n
					}
				}
			}
			if title, ok := r["title"].(map[string]any); ok {
				if runs, ok := title["runs"].([]any); ok && len(runs) > 0 {
					if first, ok := runs[0].(map[string]any); ok {
						if txt, ok := first["text"].(string); ok {
							it.Title = txt
						}
					}
				}
			}
			*out = append(*out, it)
			return
		}
		for _, val := range v {
			collectPlaylistVideoRenderers(val, out, limit)
			if len(*out) >= limit {
				return
			}
		}
	case []any:
		for _, val := range v {
			collectPlaylistVideoRenderers(val, out, limit)
			if len(*out) >= limit {
				return
			}
		}
	}
}

func findFirstContinuationToken(node any) string {
	switch v := node.(type) {
	case map[string]any:
		// common places: continuationCommand.token, nextContinuationData.continuation
		if cc, ok := v["continuationCommand"].(map[string]any); ok {
			if tok, ok := cc["token"].(string); ok && tok != "" {
				return tok
			}
		}
		if nd, ok := v["nextContinuationData"].(map[string]any); ok {
			if tok, ok := nd["continuation"].(string); ok && tok != "" {
				return tok
			}
		}
		if tok, ok := v["continuation"].(string); ok && tok != "" {
			return tok
		}
		for _, val := range v {
			if t := findFirstContinuationToken(val); t != "" {
				return t
			}
		}
	case []any:
		for _, val := range v {
			if t := findFirstContinuationToken(val); t != "" {
				return t
			}
		}
	}
	return ""
}

// getVisitorId returns the current visitor ID, refreshing it if necessary
func (c *Client) getVisitorId() (string, error) {
	var err error
	if c.visitorId.value == "" || time.Since(c.visitorId.updated) > visitorIdMaxAge {
		err = c.refreshVisitorId()
	}
	return c.visitorId.value, err
}

// refreshVisitorId fetches a new visitor ID from YouTube's main page
func (c *Client) refreshVisitorId() error {
	const sep = "\nytcfg.set("

	req, err := http.NewRequest(http.MethodGet, "https://www.youtube.com", nil)
	if err != nil {
		return err
	}

	// Add standard headers
	req.Header.Set("User-Agent", userAgentValue)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	_, data1, found := strings.Cut(string(data), sep)
	if !found {
		return errors.New("visitor ID not found in YouTube response")
	}

	var value struct {
		InnertubeContext struct {
			Client struct {
				VisitorData string `json:"visitorData"`
			} `json:"client"`
		} `json:"INNERTUBE_CONTEXT"`
	}

	if err := json.NewDecoder(strings.NewReader(data1)).Decode(&value); err != nil {
		return err
	}

	c.visitorId.value = strings.ReplaceAll(value.InnertubeContext.Client.VisitorData, "%3D", "=")

	c.visitorId.updated = time.Now()
	return nil
}

// doWithBotguardRetry executes the request and, if configured in Auto/Force mode,
// attempts a single Botguard attestation on 403 to retry the same request with
// the obtained token applied as needed.
func (c *Client) doWithBotguardRetry(req *http.Request) (*http.Response, error) {
	// Fast path: Botguard disabled
	if c.bg.solver == nil || c.bg.mode == botguard.Off {
		return c.HTTPClient.Do(req)
	}

	// Optionally run preflight attestation in Force mode
	if c.bg.mode == botguard.Force {
		if c.bg.debug {
			fmt.Println("[botguard] force mode preflight attestation")
		}
		c.maybeApplyBotguard(req)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil || resp == nil || resp.StatusCode != http.StatusForbidden {
		return resp, err
	}
	_ = resp.Body.Close()

	// Auto mode: perform attestation and retry once
	if c.bg.mode == botguard.Auto || c.bg.mode == botguard.Force {
		if c.bg.debug {
			fmt.Println("[botguard] 403 detected, attempting attestation and retry")
		}
		if err := c.maybeApplyBotguard(req); err == nil {
			return c.HTTPClient.Do(req)
		}
	}
	return resp, err
}

// maybeApplyBotguard runs the solver and applies the token to request headers.
func (c *Client) maybeApplyBotguard(req *http.Request) error {
	if c.bg.solver == nil {
		return nil
	}
	// Prepare input
	visitorId := req.Header.Get("x-goog-visitor-id")
	name := c.clientName
	if strings.TrimSpace(name) == "" {
		name = clientNameWEB
	}
	in := botguard.Input{
		UserAgent:     req.Header.Get("User-Agent"),
		PageURL:       "https://www.youtube.com/", // best-effort
		ClientName:    name,
		ClientVersion: c.clientVer,
		VisitorID:     visitorId,
	}
	key := botguard.KeyFromInput(in)
	if c.bg.cache != nil {
		if out, ok := c.bg.cache.Get(key); ok && (out.ExpiresAt.IsZero() || time.Until(out.ExpiresAt) > 0) {
			if c.bg.debug {
				fmt.Println("[botguard] cache hit: applying cached token")
			}
			if out.Token != "" {
				req.Header.Set("x-goog-ext-123-botguard", out.Token)
			}
			return nil
		}
		if c.bg.debug {
			fmt.Println("[botguard] cache miss: computing token")
		}
	}
	out, err := c.bg.solver.Attest(req.Context(), in)
	if err != nil {
		if c.bg.debug {
			fmt.Printf("[botguard] attestation error: %v\n", err)
		}
		return err
	}
	if out.ExpiresAt.IsZero() && c.bg.ttl > 0 {
		out.ExpiresAt = time.Now().Add(c.bg.ttl)
	}
	if out.Token != "" {
		if c.bg.debug {
			fmt.Println("[botguard] token obtained, applying to headers")
		}
		req.Header.Set("x-goog-ext-123-botguard", out.Token)
	}
	if c.bg.cache != nil {
		c.bg.cache.Set(key, out)
	}
	return nil
}
