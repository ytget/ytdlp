package ytdlp

import (
	"testing"

	"github.com/ytget/ytdlp/v2/types"
	"github.com/ytget/ytdlp/v2/youtube/formats"
)

func TestExtractVideoID(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		// Example.com URLs (existing)
		{"https://example.com/watch?v=abc123", "abc123"},
		{"https://ex.be/xyz789", "xyz789"},
		{"https://example.com/shorts/brZCOVlyPPo", "brZCOVlyPPo"},
		{"https://example.com/shorts/abc123", "abc123"},
		{"https://example.com/shorts/xyz789?si=3E6i4QoYvnJjqS_b", "xyz789"},
		{"https://example.com/watch?app=desktop&v=def456&feature=ex.be", "def456"},
		{"https://ex.be/ghi789?si=token", "ghi789"},
		
		// YouTube URLs (new)
		{"https://youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://youtu.be/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://www.youtube.com/shorts/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://youtube.com/shorts/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://youtu.be/dQw4w9WgXcQ?si=token", "dQw4w9WgXcQ"},
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=10s", "dQw4w9WgXcQ"},
		
		// Direct video IDs (new)
		{"dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"abc123def45", "abc123def45"},
	}
	for _, tc := range cases {
		got, err := extractVideoID(tc.url)
		if err != nil {
			t.Fatalf("%s -> error: %v (want %s)", tc.url, err, tc.want)
		}
		if got != tc.want {
			t.Fatalf("%s -> got %s (want %s)", tc.url, got, tc.want)
		}
	}
}

func TestExtractVideoID_Invalid(t *testing.T) {
	cases := []string{
		"https://example.com/watch?foo=bar",
		"https://other.com/",
		"not a url",
		"https://example.com/playlist?list=PLxxxx",
		"https://example.com/channel/UCxxxx",
	}
	for _, u := range cases {
		got, err := extractVideoID(u)
		if got != "" || err == nil {
			t.Fatalf("%s -> got=%q err=%v; want empty id and error", u, got, err)
		}
	}
}

func TestSelectFormatByExt(t *testing.T) {
	list := []types.Format{
		{Itag: 18, MimeType: "video/mp4", URL: "u1"},
		{Itag: 22, MimeType: "video/mp4", URL: "u2"},
		{Itag: 100, MimeType: "video/webm", URL: "u3"},
	}
	if f := formats.SelectFormat(list, "", "webm"); f == nil || f.URL != "u3" {
		t.Fatalf("want webm u3, got %+v", f)
	}
	if f := formats.SelectFormat(list, "", ""); f == nil || f.URL != "u2" {
		t.Fatalf("want itag 22 u2, got %+v", f)
	}
}

func TestSelectFormat_ItagAndHeight(t *testing.T) {
	list := []types.Format{
		{Itag: 18, MimeType: "video/mp4", URL: "u1", Quality: "360p", Bitrate: 1000},
		{Itag: 22, MimeType: "video/mp4", URL: "u2", Quality: "720p", Bitrate: 2000},
		{Itag: 100, MimeType: "video/webm", URL: "u3", Quality: "480p", Bitrate: 1500},
	}

	// itag selector
	if f := formats.SelectFormat(list, "itag=18", ""); f == nil || f.Itag != 18 {
		t.Fatalf("itag=18 -> got %+v", f)
	}

	// height<=480 should select within constraint (360p or 480p per heuristic)
	if f := formats.SelectFormat(list, "height<=480", ""); f == nil || (f.Quality != "480p" && f.Quality != "360p") {
		t.Fatalf("height<=480 -> want 360p/480p, got %+v", f)
	}

	// case-insensitive extension with dot
	if f := formats.SelectFormat(list, "", ".WEBM"); f == nil || f.URL != "u3" {
		t.Fatalf("ext .WEBM -> want u3, got %+v", f)
	}
}
