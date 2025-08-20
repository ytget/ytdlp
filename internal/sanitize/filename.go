package sanitize

import (
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// MaxFilenameLength is the maximum allowed length for the filename base.
	MaxFilenameLength = 120
	// DefaultExt is the default extension used when none is provided.
	DefaultExt = "mp4"
	// DefaultName is the replacement name when the title is empty.
	DefaultName = "video"
)

var unsafeChars = regexp.MustCompile(`[\\/:*?"<>|]+`)

// ToSafeFilename builds a cross-platform safe filename from title and extension (without dot in ext).
func ToSafeFilename(title, ext string) string {
	name := strings.TrimSpace(title)
	if name == "" {
		name = DefaultName
	}
	name = unsafeChars.ReplaceAllString(name, "_")
	name = strings.TrimSpace(name)
	if len(name) > MaxFilenameLength {
		name = name[:MaxFilenameLength]
	}
	ext = strings.TrimPrefix(strings.ToLower(ext), ".")
	if ext == "" {
		ext = DefaultExt
	}
	return filepath.Clean(name + "." + ext)
}
