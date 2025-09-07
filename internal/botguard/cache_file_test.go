package botguard

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileCache_SetGet(t *testing.T) {
	dir := t.TempDir()
	fc, err := NewFileCache(dir)
	if err != nil {
		t.Fatalf("NewFileCache error: %v", err)
	}
	key := "ua|WEB|1.2.3|visitor"
	out := Output{Token: "xyz", ExpiresAt: time.Now().Add(time.Minute)}

	if _, ok := fc.Get(key); ok {
		t.Fatalf("expected empty cache miss")
	}
	fc.Set(key, out)
	if _, err := os.Stat(filepath.Join(dir)); err != nil {
		t.Fatalf("cache directory missing: %v", err)
	}
	got, ok := fc.Get(key)
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if got.Token != out.Token {
		t.Fatalf("token mismatch: got %q want %q", got.Token, out.Token)
	}
}

func TestFileCache_Expire(t *testing.T) {
	dir := t.TempDir()
	fc, _ := NewFileCache(dir)
	key := "ua|WEB|1.2.3|visitor"
	out := Output{Token: "will-expire", ExpiresAt: time.Now().Add(10 * time.Millisecond)}
	fc.Set(key, out)
	time.Sleep(20 * time.Millisecond)
	if _, ok := fc.Get(key); ok {
		t.Fatalf("expected expired entry to be a miss")
	}
}

