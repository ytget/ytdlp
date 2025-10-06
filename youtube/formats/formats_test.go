package formats

import (
	"testing"

	"github.com/ytget/ytdlp/v2/types"
)

func TestSelectFormat_Ext_Itag(t *testing.T) {
	list := []types.Format{
		{Itag: 18, MimeType: "video/mp4", URL: "u1", Quality: "360p", Bitrate: 500000},
		{Itag: 22, MimeType: "video/mp4", URL: "u2", Quality: "720p", Bitrate: 2000000},
		{Itag: 100, MimeType: "video/webm", URL: "u3", Quality: "1080p", Bitrate: 3000000},
	}
	if f := SelectFormat(list, "", "webm"); f == nil || f.URL != "u3" {
		t.Fatalf("ext webm -> u3, got %+v", f)
	}
	if f := SelectFormat(list, "itag=18", ""); f == nil || f.URL != "u1" {
		t.Fatalf("itag=18 -> u1, got %+v", f)
	}
}

func TestSelectFormat_BestWorst_Height(t *testing.T) {
	list := []types.Format{
		{Itag: 18, MimeType: "video/mp4", URL: "u1", Quality: "360p", Bitrate: 500000},
		{Itag: 22, MimeType: "video/mp4", URL: "u2", Quality: "720p", Bitrate: 2000000},
		{Itag: 100, MimeType: "video/webm", URL: "u3", Quality: "1080p", Bitrate: 3000000},
	}
	if f := SelectFormat(list, "best", ""); f == nil || f.URL != "u3" {
		t.Fatalf("best -> u3, got %+v", f)
	}
	if f := SelectFormat(list, "worst", ""); f == nil || f.URL != "u1" {
		t.Fatalf("worst -> u1, got %+v", f)
	}
	if f := SelectFormat(list, "height<=720", ""); f == nil || (f.URL != "u2" && f.URL != "u1") {
		t.Fatalf("height<=720 -> u1/u2, got %+v", f)
	}
}
