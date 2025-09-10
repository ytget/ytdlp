package cipher

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	// Reset caches before test
	playerJSCache = make(map[string]playerJSCacheEntry)
	signatureCache = make(map[string]signatureCacheEntry)

	// Test data
	now := time.Now()
	expiredTime := now.Add(-time.Hour)
	validTime := now.Add(time.Hour)

	// Add test entries
	playerJSCache["valid"] = playerJSCacheEntry{
		body:  []byte("valid"),
		expAt: validTime,
	}
	playerJSCache["expired"] = playerJSCacheEntry{
		body:  []byte("expired"),
		expAt: expiredTime,
	}

	signatureCache["valid"] = signatureCacheEntry{
		value: "valid",
		expAt: validTime,
	}
	signatureCache["expired"] = signatureCacheEntry{
		value: "expired",
		expAt: expiredTime,
	}

	// Run cleanup
	cleanupCaches()

	// Check player.js cache
	if _, ok := playerJSCache["valid"]; !ok {
		t.Error("Valid player.js entry was removed")
	}
	if _, ok := playerJSCache["expired"]; ok {
		t.Error("Expired player.js entry was not removed")
	}

	// Check signature cache
	if _, ok := signatureCache["valid"]; !ok {
		t.Error("Valid signature entry was removed")
	}
	if _, ok := signatureCache["expired"]; ok {
		t.Error("Expired signature entry was not removed")
	}
}

func TestCacheTTL(t *testing.T) {
	if playerJSTTL <= 0 {
		t.Error("playerJSTTL should be positive")
	}
	if signatureTTL <= 0 {
		t.Error("signatureTTL should be positive")
	}
	if cleanupInterval <= 0 {
		t.Error("cleanupInterval should be positive")
	}
	if cleanupInterval >= playerJSTTL {
		t.Error("cleanupInterval should be less than playerJSTTL")
	}
	if cleanupInterval >= signatureTTL {
		t.Error("cleanupInterval should be less than signatureTTL")
	}
}

// Helper function to run cache cleanup
func cleanupCaches() {
	now := time.Now()

	playerJSCacheMu.Lock()
	for url, entry := range playerJSCache {
		if now.After(entry.expAt) {
			delete(playerJSCache, url)
		}
	}
	playerJSCacheMu.Unlock()

	signatureCacheMu.Lock()
	for sig, entry := range signatureCache {
		if now.After(entry.expAt) {
			delete(signatureCache, sig)
		}
	}
	signatureCacheMu.Unlock()
}
