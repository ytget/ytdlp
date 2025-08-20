package formats

import (
	"testing"

	"github.com/ytget/ytdlp/types"
)

func TestHasDirectURL(t *testing.T) {
	if !hasDirectURL(types.Format{URL: "http://x"}) {
		t.Fatal("expected true for non-empty URL")
	}
	if hasDirectURL(types.Format{URL: ""}) {
		t.Fatal("expected false for empty URL")
	}
}

func TestMimeSubtypeEquals(t *testing.T) {
	f := types.Format{MimeType: "video/mp4; codecs=\"avc1.64001F\""}
	if !mimeSubtypeEquals(f, "mp4") {
		t.Fatal("mp4 should match")
	}
	if !mimeSubtypeEquals(f, ".mp4") {
		t.Fatal(".mp4 should match")
	}
	if mimeSubtypeEquals(f, "webm") {
		t.Fatal("webm should not match mp4")
	}
}

func TestWithinHeight(t *testing.T) {
	f := types.Format{Quality: "720p"}
	if !withinHeight(f, 0, 0) {
		t.Fatal("no bounds should pass")
	}
	if !withinHeight(f, 480, 1080) {
		t.Fatal("720p should be within 480..1080")
	}
	if withinHeight(f, 1080, 0) {
		t.Fatal("720p should not be >=1080")
	}
	if withinHeight(f, 0, 360) {
		t.Fatal("720p should not be <=360")
	}
}

func TestBetterByHeightThenBitrate(t *testing.T) {
	a := types.Format{Quality: "720p", Bitrate: 1}
	b := types.Format{Quality: "1080p", Bitrate: 1}
	if !betterByHeightThenBitrate(b, a) {
		t.Fatal("1080p should be better than 720p")
	}
	c := types.Format{Quality: "720p", Bitrate: 100}
	if !betterByHeightThenBitrate(c, a) {
		t.Fatal("higher bitrate should be better at same height")
	}
}
