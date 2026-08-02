package captions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ytget/ytdlp/v2/types"
	"github.com/ytget/ytdlp/v2/youtube/innertube"
)

func TestParseTracksAndSelect(t *testing.T) {
	pr := &innertube.PlayerResponse{}
	pr.Captions.PlayerCaptionsTracklistRenderer.CaptionTracks = []innertube.CaptionTrackRaw{
		{
			BaseURL:      "https://example/a",
			LanguageCode: "ru",
			Name:         innertube.CaptionName{Runs: []struct{ Text string `json:"text"` }{{Text: "Russian"}}},
			Kind:         "asr",
			VssID:        "a.ru",
		},
		{
			BaseURL:      "https://example/b",
			LanguageCode: "en",
			Name:         innertube.CaptionName{SimpleText: "English"},
			Kind:         "",
			VssID:        ".en",
		},
	}
	tracks := ParseTracks(pr)
	if len(tracks) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(tracks))
	}
	if !tracks[0].IsAuto {
		t.Fatal("first track should be auto")
	}
	if tracks[0].LanguageName != "Russian" {
		t.Fatalf("expected runs name, got %q", tracks[0].LanguageName)
	}
	sel, ok := SelectTrack(tracks, "en")
	if !ok || sel.LanguageCode != "en" || sel.IsAuto {
		t.Fatalf("expected manual en, got %+v ok=%v", sel, ok)
	}
	sel, ok = SelectTrack(tracks, "ru")
	if !ok || sel.LanguageCode != "ru" || !sel.IsAuto {
		t.Fatalf("expected auto ru, got %+v", sel)
	}
}

func TestDownloadPrefersSRV1(t *testing.T) {
	var saw []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt := r.URL.Query().Get("fmt")
		saw = append(saw, fmt)
		switch fmt {
		case "srv1":
			_, _ = w.Write([]byte(`<?xml version="1.0"?><transcript><text start="0" dur="1.5">Hello world</text><text start="1.6" dur="1">There</text></transcript>`))
		case "json3":
			_, _ = w.Write([]byte(`{"events":[{"tStartMs":0,"dDurationMs":1500,"segs":[{"utf8":"Hello"},{"utf8":" world"}]}]}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	caps, err := Download(context.Background(), srv.Client(), types.CaptionTrack{
		LanguageCode: "en",
		BaseURL:      srv.URL + "/timedtext?v=x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(caps.Cues) != 2 {
		t.Fatalf("cues=%d", len(caps.Cues))
	}
	if caps.Cues[0].Text != "Hello world" {
		t.Fatalf("cue0=%q", caps.Cues[0].Text)
	}
	if len(saw) == 0 || saw[0] != "srv1" {
		t.Fatalf("expected srv1 first, saw=%v", saw)
	}
}

func TestParseXMLClassicEntities(t *testing.T) {
	body := []byte(`<?xml version="1.0"?><transcript><text start="1.5" dur="2.0">we&amp;#39;re in an AI</text></transcript>`)
	cues, err := parseXMLCaptions(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 1 || !strings.Contains(cues[0].Text, "we're") {
		t.Fatalf("unexpected %+v", cues)
	}
	if cues[0].StartMs != 1500 {
		t.Fatalf("start=%d", cues[0].StartMs)
	}
}

func TestParseJSON3(t *testing.T) {
	body := []byte(`{"events":[{"tStartMs":0,"dDurationMs":1500,"segs":[{"utf8":"Hello"},{"utf8":" world"}]},{"tStartMs":1600,"dDurationMs":1000,"segs":[{"utf8":"\nThere"}]},{"tStartMs":0,"dDurationMs":9999}]}`)
	cues, err := parseJSON3(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 2 {
		t.Fatalf("cues=%d", len(cues))
	}
	if cues[0].Text != "Hello world" || cues[1].Text != "There" {
		t.Fatalf("%+v", cues)
	}
}
