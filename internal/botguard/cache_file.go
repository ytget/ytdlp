package botguard

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileCache stores Botguard outputs on disk, one file per key.
// Expired entries are treated as missing.
type FileCache struct {
	rootDir string
	mu      sync.RWMutex
}

// NewFileCache creates a file-backed cache under rootDir.
// The directory will be created if it does not exist.
func NewFileCache(rootDir string) (*FileCache, error) {
	if rootDir == "" {
		return nil, errors.New("rootDir is required")
	}
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return nil, err
	}
	return &FileCache{rootDir: rootDir}, nil
}

func (c *FileCache) filenameForKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	name := fmt.Sprintf("%x.json", sum[:])
	return filepath.Join(c.rootDir, name)
}

type fileEntry struct {
	Token     string            `json:"token"`
	ExpiresAt time.Time         `json:"expiresAt"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

func (c *FileCache) Get(key string) (Output, bool) {
	c.mu.RLock()
	fn := c.filenameForKey(key)
	c.mu.RUnlock()

	b, err := os.ReadFile(fn)
	if err != nil {
		return Output{}, false
	}
	var e fileEntry
	if err := json.Unmarshal(b, &e); err != nil {
		_ = os.Remove(fn)
		return Output{}, false
	}
	if !e.ExpiresAt.IsZero() && time.Until(e.ExpiresAt) <= 0 {
		_ = os.Remove(fn)
		return Output{}, false
	}
	return Output{Token: e.Token, ExpiresAt: e.ExpiresAt, Metadata: e.Metadata}, true
}

func (c *FileCache) Set(key string, value Output) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fn := c.filenameForKey(key)
	tmp := fn + ".tmp"
	e := fileEntry{Token: value.Token, ExpiresAt: value.ExpiresAt, Metadata: value.Metadata}
	b, _ := json.Marshal(e)
	_ = os.WriteFile(tmp, b, fs.FileMode(0o644))
	_ = os.Rename(tmp, fn)
}

