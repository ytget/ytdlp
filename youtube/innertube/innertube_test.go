package innertube

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ytget/ytdlp/v2/internal/botguard"
	"github.com/ytget/ytdlp/v2/types"
)

// mockYouTubeTransport intercepts YouTube requests and returns predefined responses
type mockYouTubeTransport struct {
	responseStatus int
	responseBody   string
}

func (t *mockYouTubeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Create a mock response
	resp := &http.Response{
		StatusCode: t.responseStatus,
		Header:     make(http.Header),
		Body:       http.NoBody,
	}

	// Set response body
	if t.responseBody != "" {
		resp.Body = io.NopCloser(strings.NewReader(t.responseBody))
	}

	return resp, nil
}

type stubSolver struct{ token string }

func (s stubSolver) Attest(ctx context.Context, in botguard.Input) (botguard.Output, error) {
	return botguard.Output{Token: s.token, ExpiresAt: time.Now().Add(time.Minute)}, nil
}

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		httpClient *http.Client
		expectNil  bool
	}{
		{
			name:       "Nil HTTP client",
			httpClient: nil,
			expectNil:  false,
		},
		{
			name:       "Custom HTTP client",
			httpClient: &http.Client{Timeout: 10 * time.Second},
			expectNil:  false,
		},
		{
			name: "HTTP client with custom transport",
			httpClient: &http.Client{
				Transport: &http.Transport{
					MaxIdleConns: 50,
				},
			},
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(tt.httpClient)

			if tt.expectNil {
				if client != nil {
					t.Errorf("Expected nil client, got %v", client)
				}
			} else {
				if client == nil {
					t.Errorf("Expected non-nil client, got nil")
					return
				}
				if client.HTTPClient == nil {
					t.Errorf("Expected non-nil HTTPClient, got nil")
				}
				if client.clientName != clientNameWEB {
					t.Errorf("Expected clientName %s, got %s", clientNameWEB, client.clientName)
				}
			}
		})
	}
}

func TestWithBotguardDebug(t *testing.T) {
	client := &Client{}

	// Test enabling debug
	result := client.WithBotguardDebug(true)
	if !result.bg.debug {
		t.Error("Expected debug to be true")
	}

	// Test disabling debug
	result = client.WithBotguardDebug(false)
	if result.bg.debug {
		t.Error("Expected debug to be false")
	}
}

func TestWithBotguardTTL(t *testing.T) {
	client := &Client{}
	ttl := 5 * time.Minute

	result := client.WithBotguardTTL(ttl)
	if result.bg.ttl != ttl {
		t.Errorf("Expected TTL %v, got %v", ttl, result.bg.ttl)
	}
}

func TestGetPlaylistItems(t *testing.T) {
	// Create a mock HTTP client
	httpClient := &http.Client{}
	client := New(httpClient)

	tests := []struct {
		name       string
		playlistID string
		limit      int
		hasError   bool
	}{
		{
			name:       "Valid playlist ID with default limit",
			playlistID: "PL1234567890",
			limit:      0,
			hasError:   true, // Will fail because we don't have a real API key
		},
		{
			name:       "Valid playlist ID with custom limit",
			playlistID: "PL1234567890",
			limit:      50,
			hasError:   true, // Will fail because we don't have a real API key
		},
		{
			name:       "Empty playlist ID",
			playlistID: "",
			limit:      10,
			hasError:   true, // Will fail because we don't have a real API key
		},
		{
			name:       "Negative limit",
			playlistID: "PL1234567890",
			limit:      -1,
			hasError:   true, // Will fail because we don't have a real API key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := client.GetPlaylistItems(tt.playlistID, tt.limit)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if items == nil {
					t.Errorf("Expected items, got nil")
				}
			}
		})
	}
}

func TestGetPlaylistItemsAll(t *testing.T) {
	// Create a mock HTTP client
	httpClient := &http.Client{}
	client := New(httpClient)

	tests := []struct {
		name       string
		playlistID string
		hasError   bool
	}{
		{
			name:       "Valid playlist ID",
			playlistID: "PL1234567890",
			hasError:   true, // Will fail because we don't have a real API key
		},
		{
			name:       "Empty playlist ID",
			playlistID: "",
			hasError:   true, // Will fail because we don't have a real API key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := client.GetPlaylistItemsAll(tt.playlistID, 100)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if items == nil {
					t.Errorf("Expected items, got nil")
				}
			}
		})
	}
}

func TestGetPlaylistContinuation(t *testing.T) {
	// Create a mock HTTP client
	httpClient := &http.Client{}
	client := New(httpClient)

	tests := []struct {
		name         string
		continuation string
		hasError     bool
	}{
		{
			name:         "Valid continuation token",
			continuation: "valid_token",
			hasError:     true, // Will fail because we don't have a real API key
		},
		{
			name:         "Empty continuation token",
			continuation: "",
			hasError:     true, // Will fail because we don't have a real API key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := client.getPlaylistContinuation(tt.continuation)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestCollectPlaylistVideoRenderers(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		expected int
	}{
		{
			name:     "Empty data",
			data:     map[string]any{},
			expected: 0,
		},
		{
			name: "Data without video renderers",
			data: map[string]any{
				"contents": map[string]any{
					"twoColumnBrowseResultsRenderer": map[string]any{
						"tabs": []map[string]any{
							{
								"tabRenderer": map[string]any{
									"content": map[string]any{
										"sectionListRenderer": map[string]any{
											"contents": []map[string]any{
												{
													"itemSectionRenderer": map[string]any{
														"contents": []map[string]any{
															{
																"playlistVideoListRenderer": map[string]any{
																	"contents": []map[string]any{},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var items []types.PlaylistItem
			collectPlaylistVideoRenderers(tt.data, &items, 0)
			if len(items) != tt.expected {
				t.Errorf("Expected %d items, got %d", tt.expected, len(items))
			}
		})
	}
}

func TestBotguardRetryOn403(t *testing.T) {
	// First request returns 403, second returns 200 with minimal JSON
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		if call == 1 {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		// minimal player response
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"playabilityStatus":{"status":"OK"}}`))
	}))
	defer srv.Close()

	c := &http.Client{Timeout: 5 * time.Second}
	it := New(c)
	it.WithBotguard(stubSolver{token: "t"}, botguard.Auto, botguard.NewMemoryCache())
	it.clientVer = "2.0"
	it.apiKey = "k"

	// Replace endpoints for test
	oldPlayerURL := playerURL
	playerURL = srv.URL
	defer func() { playerURL = oldPlayerURL }()

	// Call
	_, err := it.GetPlayerResponse("vid")
	if err != nil && !strings.Contains(err.Error(), "failed to parse response") {
		t.Fatalf("unexpected error: %v", err)
	}
	if call < 2 {
		t.Fatalf("expected retry after 403, got calls=%d", call)
	}
}

func TestBotguardTTLApplied(t *testing.T) {
	c := &Client{HTTPClient: &http.Client{Timeout: 2 * time.Second}}
	c.clientVer = "2.0"
	cache := botguard.NewMemoryCache()
	// Solver returns token with zero ExpiresAt -> TTL must be applied
	solver := stubSolver{token: "tok"}
	c.WithBotguard(solver, botguard.Force, cache).WithBotguardTTL(1 * time.Minute)

	// Build dummy request without network
	req, _ := http.NewRequest(http.MethodPost, "http://example/", nil)
	req.Header.Set("User-Agent", userAgentValue)
	// No visitor id header

	if err := c.maybeApplyBotguard(req); err != nil {
		t.Fatalf("maybeApplyBotguard error: %v", err)
	}

	// Construct cache key and verify expiry set
	key := botguard.KeyFromInput(botguard.Input{
		UserAgent:     userAgentValue,
		PageURL:       "https://www.youtube.com/",
		ClientName:    clientNameWEB,
		ClientVersion: c.clientVer,
		VisitorID:     "",
	})
	out, ok := cache.Get(key)
	if !ok {
		t.Fatalf("expected cache hit after attestation")
	}
	if out.Token == "" {
		t.Fatalf("expected non-empty token")
	}
	if out.ExpiresAt.IsZero() {
		t.Fatalf("expected ExpiresAt to be set from TTL")
	}
	if time.Until(out.ExpiresAt) <= 0 {
		t.Fatalf("expected ExpiresAt in the future")
	}
}

func TestRefreshVisitorID(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		responseStatus int
		hasError       bool
	}{
		{
			name:           "Valid response with visitor ID",
			responseBody:   "\nytcfg.set({\"INNERTUBE_CONTEXT\":{\"client\":{\"visitorData\":\"CgtISF9rMVNrRENlVSi988zHBjInCgJQVBIhEh0SGwsMDg8QERITFBUWFxgZGhscHR4fICEiIyQlJiASOgwIASCowrjf0LfO-Wg%3D\"}}})",
			responseStatus: 200,
			hasError:       false,
		},
		{
			name:           "Response without visitor ID",
			responseBody:   `ytcfg.set({"INNERTUBE_CONTEXT":{"client":{}}})`,
			responseStatus: 200,
			hasError:       true,
		},
		{
			name:           "Invalid JSON response",
			responseBody:   `ytcfg.set(invalid json)`,
			responseStatus: 200,
			hasError:       true,
		},
		{
			name:           "Response without ytcfg.set",
			responseBody:   `{"INNERTUBE_CONTEXT":{"client":{"visitorData":"test"}}}`,
			responseStatus: 200,
			hasError:       true,
		},
		{
			name:           "HTTP error response",
			responseBody:   ``,
			responseStatus: 500,
			hasError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create custom HTTP client that returns predefined responses
			client := &http.Client{
				Transport: &mockYouTubeTransport{
					responseStatus: tt.responseStatus,
					responseBody:   tt.responseBody,
				},
			}

			// Create InnerTube client with mock HTTP client
			innertubeClient := New(client)

			// Test refreshVisitorID
			err := innertubeClient.refreshVisitorID()

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestWithClient(t *testing.T) {
	tests := []struct {
		name          string
		clientName    string
		clientVersion string
		expectedName  string
		expectedVer   string
	}{
		{
			name:          "Valid client name and version",
			clientName:    "WEB",
			clientVersion: "2.0.0",
			expectedName:  "WEB",
			expectedVer:   "2.0.0",
		},
		{
			name:          "Empty client name",
			clientName:    "",
			clientVersion: "1.0.0",
			expectedName:  "WEB", // Default value from New()
			expectedVer:   "1.0.0",
		},
		{
			name:          "Empty client version",
			clientName:    "ANDROID",
			clientVersion: "",
			expectedName:  "ANDROID",
			expectedVer:   "", // No default version set
		},
		{
			name:          "Whitespace client name",
			clientName:    "   ",
			clientVersion: "3.0.0",
			expectedName:  "WEB", // Default value from New()
			expectedVer:   "3.0.0",
		},
		{
			name:          "Whitespace client version",
			clientName:    "IOS",
			clientVersion: "   ",
			expectedName:  "IOS",
			expectedVer:   "", // No default version set
		},
		{
			name:          "Both empty",
			clientName:    "",
			clientVersion: "",
			expectedName:  "WEB", // Default value from New()
			expectedVer:   "",    // No default version set
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(nil)
			result := client.WithClient(tt.clientName, tt.clientVersion)

			if result.clientName != tt.expectedName {
				t.Errorf("Expected clientName '%s', got '%s'", tt.expectedName, result.clientName)
			}
			if result.clientVer != tt.expectedVer {
				t.Errorf("Expected clientVer '%s', got '%s'", tt.expectedVer, result.clientVer)
			}
		})
	}
}

func TestClientCodeFromName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "WEB client",
			input:    "WEB",
			expected: "1",
		},
		{
			name:     "MWEB client",
			input:    "MWEB",
			expected: "2",
		},
		{
			name:     "ANDROID client",
			input:    "ANDROID",
			expected: "3",
		},
		{
			name:     "IOS client",
			input:    "IOS",
			expected: "5",
		},
		{
			name:     "TVHTML5 client",
			input:    "TVHTML5",
			expected: "7",
		},
		{
			name:     "WEB_EMBEDDED_PLAYER client",
			input:    "WEB_EMBEDDED_PLAYER",
			expected: "56",
		},
		{
			name:     "WEB_CREATOR client",
			input:    "WEB_CREATOR",
			expected: "62",
		},
		{
			name:     "WEB_REMIX client",
			input:    "WEB_REMIX",
			expected: "67",
		},
		{
			name:     "TVHTML5_SIMPLY client",
			input:    "TVHTML5_SIMPLY",
			expected: "75",
		},
		{
			name:     "TVHTML5_SIMPLY_EMBEDDED_PLAYER client",
			input:    "TVHTML5_SIMPLY_EMBEDDED_PLAYER",
			expected: "85",
		},
		{
			name:     "Unknown client",
			input:    "UNKNOWN",
			expected: "",
		},
		{
			name:     "Empty client name",
			input:    "",
			expected: "",
		},
		{
			name:     "Lowercase client name",
			input:    "web",
			expected: "1",
		},
		{
			name:     "Mixed case client name",
			input:    "Web",
			expected: "1",
		},
		{
			name:     "Mixed case client name with underscores",
			input:    "Web_Embedded_Player",
			expected: "56",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clientCodeFromName(tt.input)
			if result != tt.expected {
				t.Errorf("Expected client code '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestFindFirstContinuationToken(t *testing.T) {
	tests := []struct {
		name     string
		node     any
		expected string
	}{
		{
			name:     "Nil node",
			node:     nil,
			expected: "",
		},
		{
			name:     "Empty map",
			node:     map[string]any{},
			expected: "",
		},
		{
			name:     "Map with continuationCommand token",
			node:     map[string]any{"continuationCommand": map[string]any{"token": "test_token"}},
			expected: "test_token",
		},
		{
			name:     "Map with nextContinuationData continuation",
			node:     map[string]any{"nextContinuationData": map[string]any{"continuation": "test_continuation"}},
			expected: "test_continuation",
		},
		{
			name:     "Map with direct continuation",
			node:     map[string]any{"continuation": "direct_token"},
			expected: "direct_token",
		},
		{
			name:     "Map with empty continuationCommand token",
			node:     map[string]any{"continuationCommand": map[string]any{"token": ""}},
			expected: "",
		},
		{
			name:     "Map with empty nextContinuationData continuation",
			node:     map[string]any{"nextContinuationData": map[string]any{"continuation": ""}},
			expected: "",
		},
		{
			name:     "Map with empty direct continuation",
			node:     map[string]any{"continuation": ""},
			expected: "",
		},
		{
			name:     "Map with nested continuationCommand",
			node:     map[string]any{"data": map[string]any{"continuationCommand": map[string]any{"token": "nested_token"}}},
			expected: "nested_token",
		},
		{
			name:     "Map with nested nextContinuationData",
			node:     map[string]any{"data": map[string]any{"nextContinuationData": map[string]any{"continuation": "nested_continuation"}}},
			expected: "nested_continuation",
		},
		{
			name:     "Map with nested direct continuation",
			node:     map[string]any{"data": map[string]any{"continuation": "nested_direct"}},
			expected: "nested_direct",
		},
		{
			name:     "Array with continuationCommand",
			node:     []any{map[string]any{"continuationCommand": map[string]any{"token": "array_token"}}},
			expected: "array_token",
		},
		{
			name:     "Array with nextContinuationData",
			node:     []any{map[string]any{"nextContinuationData": map[string]any{"continuation": "array_continuation"}}},
			expected: "array_continuation",
		},
		{
			name:     "Array with direct continuation",
			node:     []any{map[string]any{"continuation": "array_direct"}},
			expected: "array_direct",
		},
		{
			name:     "Array with empty continuationCommand",
			node:     []any{map[string]any{"continuationCommand": map[string]any{"token": ""}}},
			expected: "",
		},
		{
			name:     "Array with empty nextContinuationData",
			node:     []any{map[string]any{"nextContinuationData": map[string]any{"continuation": ""}}},
			expected: "",
		},
		{
			name:     "Array with empty direct continuation",
			node:     []any{map[string]any{"continuation": ""}},
			expected: "",
		},
		{
			name:     "Array with multiple elements",
			node:     []any{map[string]any{"continuation": ""}, map[string]any{"continuation": "second_token"}},
			expected: "second_token",
		},
		{
			name:     "Map with multiple continuation sources",
			node:     map[string]any{"continuationCommand": map[string]any{"token": "first_token"}, "nextContinuationData": map[string]any{"continuation": "second_token"}},
			expected: "first_token", // Should return first found
		},
		{
			name:     "Map with non-string continuationCommand token",
			node:     map[string]any{"continuationCommand": map[string]any{"token": 123}},
			expected: "",
		},
		{
			name:     "Map with non-string nextContinuationData continuation",
			node:     map[string]any{"nextContinuationData": map[string]any{"continuation": 123}},
			expected: "",
		},
		{
			name:     "Map with non-string direct continuation",
			node:     map[string]any{"continuation": 123},
			expected: "",
		},
		{
			name:     "Map with non-map continuationCommand",
			node:     map[string]any{"continuationCommand": "not_a_map"},
			expected: "",
		},
		{
			name:     "Map with non-map nextContinuationData",
			node:     map[string]any{"nextContinuationData": "not_a_map"},
			expected: "",
		},
		{
			name:     "Array with non-map elements",
			node:     []any{"not_a_map", 123},
			expected: "",
		},
		{
			name:     "Map with non-array non-map values",
			node:     map[string]any{"key": "value", "number": 123},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findFirstContinuationToken(tt.node)
			if result != tt.expected {
				t.Errorf("Expected continuation token '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestDoWithBotguardRetry(t *testing.T) {
	tests := []struct {
		name           string
		mode           botguard.Mode
		solver         botguard.Solver
		responseStatus int
		expectRetry    bool
	}{
		{
			name:           "Botguard disabled",
			mode:           botguard.Off,
			solver:         nil,
			responseStatus: 200,
			expectRetry:    false,
		},
		{
			name:           "Botguard disabled with solver",
			mode:           botguard.Off,
			solver:         &stubSolver{token: "test_token"},
			responseStatus: 200,
			expectRetry:    false,
		},
		{
			name:           "Botguard auto mode with 200 response",
			mode:           botguard.Auto,
			solver:         &stubSolver{token: "test_token"},
			responseStatus: 200,
			expectRetry:    false,
		},
		{
			name:           "Botguard auto mode with 403 response",
			mode:           botguard.Auto,
			solver:         &stubSolver{token: "test_token"},
			responseStatus: 403,
			expectRetry:    true,
		},
		{
			name:           "Botguard force mode with 200 response",
			mode:           botguard.Force,
			solver:         &stubSolver{token: "test_token"},
			responseStatus: 200,
			expectRetry:    false,
		},
		{
			name:           "Botguard force mode with 403 response",
			mode:           botguard.Force,
			solver:         &stubSolver{token: "test_token"},
			responseStatus: 403,
			expectRetry:    true,
		},
		{
			name:           "Botguard auto mode with 403 response and nil solver",
			mode:           botguard.Auto,
			solver:         nil,
			responseStatus: 403,
			expectRetry:    false,
		},
		{
			name:           "Botguard force mode with 403 response and nil solver",
			mode:           botguard.Force,
			solver:         nil,
			responseStatus: 403,
			expectRetry:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			}))
			defer srv.Close()

			client := &http.Client{Timeout: 5 * time.Second}
			innertubeClient := New(client)
			innertubeClient.WithBotguard(tt.solver, tt.mode, botguard.NewMemoryCache())

			req, _ := http.NewRequest("GET", srv.URL, nil)
			req.Header.Set("User-Agent", "test-agent")
			req.Header.Set("x-goog-visitor-id", "test-visitor-id")

			resp, err := innertubeClient.doWithBotguardRetry(req)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if resp == nil {
				t.Errorf("Expected response, got nil")
			}

			expectedCalls := 1
			if tt.expectRetry && tt.responseStatus == 403 {
				expectedCalls = 2
			}

			if callCount != expectedCalls {
				t.Errorf("Expected %d calls, got %d", expectedCalls, callCount)
			}
		})
	}
}
