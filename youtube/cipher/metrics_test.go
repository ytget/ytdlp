package cipher

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestMetrics(t *testing.T) {
	// Reset metrics before test
	metrics = struct {
		totalRequests     int64
		cacheHits         int64
		cacheMisses       int64
		avgDecipherTime   time.Duration
		totalDecipherTime time.Duration
		mu                sync.Mutex
	}{}

	// Mock data
	playerJSURL := "https://example.com/player.js"
	signature := "test_signature"

	// Create test HTTP client that always returns error
	httpClient := &http.Client{
		Transport: &mockTransport{},
	}

	// First request - should be a cache miss
	_, _ = Decipher(httpClient, playerJSURL, signature)

	if metrics.totalRequests != 1 {
		t.Errorf("Expected totalRequests = 1, got %d", metrics.totalRequests)
	}
	if metrics.cacheMisses != 1 {
		t.Errorf("Expected cacheMisses = 1, got %d", metrics.cacheMisses)
	}
	if metrics.cacheHits != 0 {
		t.Errorf("Expected cacheHits = 0, got %d", metrics.cacheHits)
	}

	// Add signature to cache
	signatureCacheMu.Lock()
	signatureCache[signature] = signatureCacheEntry{
		value: "deciphered",
		expAt: time.Now().Add(time.Hour),
	}
	signatureCacheMu.Unlock()

	// Second request - should be a cache hit
	_, _ = Decipher(httpClient, playerJSURL, signature)

	if metrics.totalRequests != 2 {
		t.Errorf("Expected totalRequests = 2, got %d", metrics.totalRequests)
	}
	if metrics.cacheMisses != 1 {
		t.Errorf("Expected cacheMisses = 1, got %d", metrics.cacheMisses)
	}
	if metrics.cacheHits != 1 {
		t.Errorf("Expected cacheHits = 1, got %d", metrics.cacheHits)
	}

	// Verify timing metrics
	// Note: timing metrics might be zero in fast tests
	t.Logf("Total decipher time: %v", metrics.totalDecipherTime)
	t.Logf("Average decipher time: %v", metrics.avgDecipherTime)
}

// mockTransport always returns error
type mockTransport struct{}

func (m *mockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("mock error")
}
