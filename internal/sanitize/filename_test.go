package sanitize

import "testing"

func TestToSafeFilename_Basics(t *testing.T) {
	got := ToSafeFilename("Hello:/\\*?\"<>| World", "mp4")
	if got != "Hello_ World.mp4" {
		t.Fatalf("got %q", got)
	}
}

func TestToSafeFilename_Defaults(t *testing.T) {
	got := ToSafeFilename("", "")
	if got != "video.mp4" {
		t.Fatalf("got %q", got)
	}
}

func TestToSafeFilename_Long(t *testing.T) {
	title := "a"
	for len(title) < 200 {
		title += "a"
	}
	got := ToSafeFilename(title, "mp4")
	if len(got) > 125 { // name(120)+.ext
		t.Fatalf("too long: %d", len(got))
	}
}
