package ytdlp

import (
	"testing"

	"github.com/ytget/ytdlp/types"
	"github.com/ytget/ytdlp/youtube/formats"
)

func TestExtractVideoID(t *testing.T) {
	cases := map[string]string{
		"https://www.youtube.com/watch?v=abc123": "abc123",
		"https://youtu.be/xyz789":                "xyz789",
	}
	for u, want := range cases {
		got, err := extractVideoID(u)
		if err != nil || got != want {
			t.Fatalf("%s -> %s (want %s, err %v)", u, got, want, err)
		}
	}
}

func TestExtractVideoID_Invalid(t *testing.T) {
	cases := []string{
		"https://www.youtube.com/shorts/abc123",
		"https://www.youtube.com/watch?foo=bar",
		"https://example.com/",
		"not a url",
	}
	for _, u := range cases {
		got, _ := extractVideoID(u)
		if got != "" {
			t.Fatalf("%s -> got=%q; want empty id", u, got)
		}
	}

	// Accept watch variants with extra params
	if got, err := extractVideoID("https://youtube.com/watch?app=desktop&v=abc123&feature=youtu.be"); err != nil || got != "abc123" {
		t.Fatalf("watch variants -> got=%q err=%v; want abc123, nil", got, err)
	}
	// Accept youtu.be with query params
	if got, err := extractVideoID("https://youtu.be/xyz789?si=token"); err != nil || got != "xyz789" {
		t.Fatalf("youtu.be with query -> got=%q err=%v; want xyz789, nil", got, err)
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
