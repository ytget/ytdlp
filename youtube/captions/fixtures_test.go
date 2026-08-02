package captions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ytget/ytdlp/v2/youtube/innertube"
)

func TestFixtureSRV1(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "manual_en.srv1.xml"))
	if err != nil {
		t.Fatal(err)
	}
	cues, err := parseXMLCaptions(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 2 || cues[0].Text != "Hello from fixture" {
		t.Fatalf("%+v", cues)
	}
}

func TestFixtureJSON3(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "asr_ru.json3"))
	if err != nil {
		t.Fatal(err)
	}
	cues, err := parseJSON3(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 2 || cues[0].Text != "Привет" {
		t.Fatalf("%+v", cues)
	}
}

func TestFixturePlayerTracksMultiAndNone(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "player_tracks_multi.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pr innertube.PlayerResponse
	if err := json.Unmarshal(raw, &pr); err != nil {
		t.Fatal(err)
	}
	tracks := ParseTracks(&pr)
	if len(tracks) != 3 {
		t.Fatalf("tracks=%d", len(tracks))
	}
	sel, ok := SelectTrack(tracks, "en")
	if !ok || sel.IsAuto || sel.LanguageCode != "en" {
		t.Fatalf("want manual en, got %+v", sel)
	}
	sel, ok = SelectTrack(tracks, "ru")
	if !ok || sel.LanguageCode != "ru" {
		t.Fatalf("want ru, got %+v", sel)
	}

	rawNone, err := os.ReadFile(filepath.Join("testdata", "player_tracks_none.json"))
	if err != nil {
		t.Fatal(err)
	}
	var prNone innertube.PlayerResponse
	if err := json.Unmarshal(rawNone, &prNone); err != nil {
		t.Fatal(err)
	}
	if len(ParseTracks(&prNone)) != 0 {
		t.Fatal("expected no tracks")
	}
}
