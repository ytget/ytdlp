package innertube

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"

	"github.com/ytget/ytdlp/types"
)

const (
	playerURL = "https://www.youtube.com/youtubei/v1/player"
	browseURL = "https://www.youtube.com/youtubei/v1/browse"

	ytBase                = "https://www.youtube.com"
	userAgentValue        = "Mozilla/5.0"
	headerContentTypeJSON = "application/json"
	clientNameWEB         = "WEB"
	defaultClientVersion  = "17.36.4"
	browseIDPrefix        = "VL"
	defaultPlaylistLimit  = 100
	continuationLimitMax  = 1 << 20
)

var (
	apiKeyRe    = regexp.MustCompile(`"INNERTUBE_API_KEY":"([^"]+)"`)
	clientVerRe = regexp.MustCompile(`"INNERTUBE_CLIENT_VERSION":"([^"]+)"`)
)

// Client for interacting with the YouTube InnerTube API.
type Client struct {
	HTTPClient *http.Client
	apiKey     string
	clientVer  string
}

// New creates a new InnerTube client.
func New(httpClient *http.Client) *Client {
	return &Client{HTTPClient: httpClient}
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
	var url string
	if isPlaylist {
		url = ytBase + "/playlist?list=" + videoOrPlaylistID
	} else {
		url = ytBase + "/watch?v=" + videoOrPlaylistID
	}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", userAgentValue)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if m := apiKeyRe.FindSubmatch(body); len(m) == 2 {
		c.apiKey = string(m[1])
	}
	if m := clientVerRe.FindSubmatch(body); len(m) == 2 {
		c.clientVer = string(m[1])
	}
	if c.clientVer == "" {
		c.clientVer = defaultClientVersion
	}
}

// GetPlayerResponse fetches video data for the provided video ID using the
// InnerTube /player endpoint.
func (c *Client) GetPlayerResponse(videoID string) (*PlayerResponse, error) {
	c.ensureKey(videoID, false)

	requestBody, err := json.Marshal(map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"clientName":    clientNameWEB,
				"clientVersion": c.clientVer,
			},
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
	req.Header.Set("User-Agent", userAgentValue)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var playerResponse PlayerResponse
	if err := json.Unmarshal(body, &playerResponse); err != nil {
		return nil, err
	}

	return &playerResponse, nil
}

// GetPlaylistItems fetches initial playlist items (without continuations, MVP).
func (c *Client) GetPlaylistItems(playlistID string, limit int) ([]types.PlaylistItem, error) {
	c.ensureKey(playlistID, true)
	if limit <= 0 {
		limit = defaultPlaylistLimit
	}

	reqBody := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"clientName":    clientNameWEB,
				"clientVersion": c.clientVer,
			},
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
	req.Header.Set("User-Agent", userAgentValue)
	resp, err := c.HTTPClient.Do(req)
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
	reqBody := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"clientName":    clientNameWEB,
				"clientVersion": c.clientVer,
			},
		},
		"browseId": browseIDPrefix + playlistID,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", browseURL+"?key="+c.apiKey, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", headerContentTypeJSON)
	req.Header.Set("User-Agent", userAgentValue)
	resp, err := c.HTTPClient.Do(req)
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
	reqBody := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"clientName":    clientNameWEB,
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
	resp, err := c.HTTPClient.Do(req)
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
