package botguard

import (
	"testing"
	"time"
)

func TestMemoryCache_SetGet(t *testing.T) {
	c := NewMemoryCache()
	key := "ua|WEB|1.2.3|visitor"
	out := Output{Token: "abc", ExpiresAt: time.Now().Add(time.Minute)}

	if _, ok := c.Get(key); ok {
		t.Fatalf("expected empty cache miss")
	}
	c.Set(key, out)
	got, ok := c.Get(key)
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if got.Token != out.Token {
		t.Fatalf("token mismatch: got %q want %q", got.Token, out.Token)
	}
}

