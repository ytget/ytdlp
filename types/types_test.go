package types

import (
	"testing"
)

func TestFormat(t *testing.T) {
	format := Format{
		Itag:            22,
		URL:             "https://example.com/video.mp4",
		Quality:         "720p",
		MimeType:        "video/mp4",
		Bitrate:         1000000,
		Size:            50000000,
		SignatureCipher: "s=abc123",
	}

	if format.Itag != 22 {
		t.Errorf("Expected Itag 22, got %d", format.Itag)
	}

	if format.URL != "https://example.com/video.mp4" {
		t.Errorf("Expected URL 'https://example.com/video.mp4', got '%s'", format.URL)
	}

	if format.Quality != "720p" {
		t.Errorf("Expected Quality '720p', got '%s'", format.Quality)
	}

	if format.MimeType != "video/mp4" {
		t.Errorf("Expected MimeType 'video/mp4', got '%s'", format.MimeType)
	}

	if format.Bitrate != 1000000 {
		t.Errorf("Expected Bitrate 1000000, got %d", format.Bitrate)
	}

	if format.Size != 50000000 {
		t.Errorf("Expected Size 50000000, got %d", format.Size)
	}

	if format.SignatureCipher != "s=abc123" {
		t.Errorf("Expected SignatureCipher 's=abc123', got '%s'", format.SignatureCipher)
	}
}

func TestFormatZeroValues(t *testing.T) {
	format := Format{}

	if format.Itag != 0 {
		t.Errorf("Expected Itag 0, got %d", format.Itag)
	}

	if format.URL != "" {
		t.Errorf("Expected empty URL, got '%s'", format.URL)
	}

	if format.Quality != "" {
		t.Errorf("Expected empty Quality, got '%s'", format.Quality)
	}

	if format.MimeType != "" {
		t.Errorf("Expected empty MimeType, got '%s'", format.MimeType)
	}

	if format.Bitrate != 0 {
		t.Errorf("Expected Bitrate 0, got %d", format.Bitrate)
	}

	if format.Size != 0 {
		t.Errorf("Expected Size 0, got %d", format.Size)
	}

	if format.SignatureCipher != "" {
		t.Errorf("Expected empty SignatureCipher, got '%s'", format.SignatureCipher)
	}
}

func TestPlaylistItem(t *testing.T) {
	item := PlaylistItem{
		VideoID: "abc123",
		Title:   "Test Video",
		Index:   1,
	}

	if item.VideoID != "abc123" {
		t.Errorf("Expected VideoID 'abc123', got '%s'", item.VideoID)
	}

	if item.Title != "Test Video" {
		t.Errorf("Expected Title 'Test Video', got '%s'", item.Title)
	}

	if item.Index != 1 {
		t.Errorf("Expected Index 1, got %d", item.Index)
	}
}

func TestPlaylistItemZeroValues(t *testing.T) {
	item := PlaylistItem{}

	if item.VideoID != "" {
		t.Errorf("Expected empty VideoID, got '%s'", item.VideoID)
	}

	if item.Title != "" {
		t.Errorf("Expected empty Title, got '%s'", item.Title)
	}

	if item.Index != 0 {
		t.Errorf("Expected Index 0, got %d", item.Index)
	}
}
