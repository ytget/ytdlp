package mimeext

import "testing"

func TestExtFromMime(t *testing.T) {
	cases := map[string]string{
		"video/mp4":                  "mp4",
		"audio/mp4":                  "m4a",
		"video/webm":                 "webm",
		"audio/webm":                 "webm",
		"video/unknown":              "unknown",
		"":                           "mp4",
		"video/mp4; codecs=\"avc1\"": "mp4",
	}
	for in, want := range cases {
		if got := ExtFromMime(in); got != want {
			t.Fatalf("%q -> %q (want %q)", in, got, want)
		}
	}
}
