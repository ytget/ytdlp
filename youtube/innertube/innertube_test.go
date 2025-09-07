package innertube

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ytget/ytdlp/internal/botguard"
)

type stubSolver struct{ token string }

func (s stubSolver) Attest(ctx context.Context, in botguard.Input) (botguard.Output, error) {
	return botguard.Output{Token: s.token, ExpiresAt: time.Now().Add(time.Minute)}, nil
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
