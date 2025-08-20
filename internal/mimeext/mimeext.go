package mimeext

import (
	"strings"
)

const (
	// DefaultExt is the extension used when MIME is unknown or empty.
	DefaultExt = "mp4"

	// ExtM4A is the file extension for MP4 audio.
	ExtM4A = "m4a"
	// ExtWebM is the file extension for WebM media.
	ExtWebM = "webm"

	// MimeVideoMP4 is the MIME type for MP4 video.
	MimeVideoMP4 = "video/mp4"
	// MimeAudioMP4 is the MIME type for MP4 audio.
	MimeAudioMP4 = "audio/mp4"
	// MimeVideoWebM is the MIME type for WebM video.
	MimeVideoWebM = "video/webm"
	// MimeAudioWebM is the MIME type for WebM audio.
	MimeAudioWebM = "audio/webm"
)

// ExtFromMime returns file extension (without dot) for given mime type.
// Falls back to subtype or mp4 if unknown.
func ExtFromMime(mime string) string {
	mime = strings.TrimSpace(mime)
	if mime == "" {
		return DefaultExt
	}
	base := mime
	if i := strings.Index(mime, ";"); i >= 0 {
		base = strings.TrimSpace(mime[:i])
	}
	switch base {
	case MimeVideoMP4:
		return DefaultExt
	case MimeAudioMP4:
		return ExtM4A
	case MimeVideoWebM, MimeAudioWebM:
		return ExtWebM
	}
	// Try subtype
	parts := strings.Split(base, "/")
	if len(parts) == 2 && parts[1] != "" {
		return parts[1]
	}
	return DefaultExt
}
