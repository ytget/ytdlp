// Package formats provides utilities for working with YouTube media formats.
package formats

import (
	"strings"

	"github.com/ytget/ytdlp/v2/types"
)

// hasDirectURL returns true when the format already contains a resolvable URL.
// Formats without direct URLs need signature deciphering.
func hasDirectURL(format types.Format) bool {
	return strings.TrimSpace(format.URL) != ""
}

// mimeSubtypeEquals checks that MIME subtype (e.g., mp4, webm) equals desiredExt.
// The desiredExt is case-insensitive and may start with a dot.
// If desiredExt is empty, the function returns true (no filtering).
func mimeSubtypeEquals(format types.Format, desiredExt string) bool {
	desired := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(desiredExt)), ".")
	if desired == "" {
		return true
	}
	return getSubtype(format.MimeType) == desired
}

// itagEquals checks that format's itag matches the specified itag value.
// Returns false if itag is 0 or negative.
func itagEquals(format types.Format, itag int) bool {
	return itag > 0 && format.Itag == itag
}

// withinHeight checks whether the format's Quality label height is within [minHeight, maxHeight] constraints.
// If a bound equals 0, that constraint is ignored (e.g., minHeight=0, maxHeight=720 means "any height up to 720p").
// If both bounds are 0, all formats pass the check.
func withinHeight(format types.Format, minHeight int, maxHeight int) bool {
	if minHeight <= 0 && maxHeight <= 0 {
		return true
	}
	h := parseHeight(format.Quality)
	if minHeight > 0 && h < minHeight {
		return false
	}
	if maxHeight > 0 && h > maxHeight {
		return false
	}
	return true
}

// betterByHeightThenBitrate compares two formats and returns true when candidate is better than current
// using height as primary criterion and bitrate as a tiebreaker.
// This is used for implementing "best" and "worst" selectors.
func betterByHeightThenBitrate(candidate types.Format, current types.Format) bool {
	candidateHeight := parseHeight(candidate.Quality)
	currentHeight := parseHeight(current.Quality)
	if candidateHeight != currentHeight {
		return candidateHeight > currentHeight
	}
	return candidate.Bitrate > current.Bitrate
}
