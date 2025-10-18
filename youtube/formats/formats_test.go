package formats

import (
	"net/http"
	"testing"

	"github.com/ytget/ytdlp/v2/types"
	"github.com/ytget/ytdlp/v2/youtube/innertube"
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

func TestParseFormats(t *testing.T) {
	// Test with empty data
	data := &innertube.PlayerResponse{}
	formats, err := ParseFormats(data)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(formats) != 0 {
		t.Errorf("Expected 0 formats, got %d", len(formats))
	}

	// Test with valid data
	data = &innertube.PlayerResponse{
		StreamingData: struct {
			Formats         []any `json:"formats"`
			AdaptiveFormats []any `json:"adaptiveFormats"`
		}{
			Formats: []any{
				map[string]any{
					"itag":          float64(18),
					"mimeType":      "video/mp4",
					"qualityLabel":  "360p",
					"bitrate":       float64(500000),
					"contentLength": "1000000",
					"url":           "https://example.com/video.mp4",
				},
			},
			AdaptiveFormats: []any{
				map[string]any{
					"itag":            float64(22),
					"mimeType":        "video/mp4",
					"qualityLabel":    "720p",
					"bitrate":         float64(2000000),
					"contentLength":   "2000000",
					"signatureCipher": "s=abc123",
				},
			},
		},
	}

	formats, err = ParseFormats(data)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(formats) != 2 {
		t.Errorf("Expected 2 formats, got %d", len(formats))
	}

	// Check first format
	if formats[0].Itag != 18 {
		t.Errorf("Expected Itag 18, got %d", formats[0].Itag)
	}
	if formats[0].MimeType != "video/mp4" {
		t.Errorf("Expected MimeType 'video/mp4', got '%s'", formats[0].MimeType)
	}
	if formats[0].URL != "https://example.com/video.mp4" {
		t.Errorf("Expected URL 'https://example.com/video.mp4', got '%s'", formats[0].URL)
	}

	// Check second format
	if formats[1].Itag != 22 {
		t.Errorf("Expected Itag 22, got %d", formats[1].Itag)
	}
	if formats[1].SignatureCipher != "s=abc123" {
		t.Errorf("Expected SignatureCipher 's=abc123', got '%s'", formats[1].SignatureCipher)
	}
}

func TestParseFormatsWithInvalidData(t *testing.T) {
	data := &innertube.PlayerResponse{
		StreamingData: struct {
			Formats         []any `json:"formats"`
			AdaptiveFormats []any `json:"adaptiveFormats"`
		}{
			Formats: []any{
				"invalid format data", // Not a map
				map[string]any{
					"itag": "invalid", // Invalid itag type
				},
			},
		},
	}

	formats, err := ParseFormats(data)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	// The function still creates formats even with invalid data, but with default values
	if len(formats) != 1 {
		t.Errorf("Expected 1 format (with invalid itag), got %d", len(formats))
	}
}

func TestResolveFormatURL(t *testing.T) {
	// Create a mock HTTP client
	httpClient := &http.Client{}

	tests := []struct {
		name        string
		format      types.Format
		playerJSURL string
		expectedURL string
		hasError    bool
	}{
		{
			name: "Format with direct URL",
			format: types.Format{
				URL: "https://example.com/video.mp4",
			},
			playerJSURL: "https://example.com/player.js",
			expectedURL: "https://example.com/video.mp4?alr=yes&ratebypass=yes",
			hasError:    false,
		},
		{
			name: "Format with URL and n parameter",
			format: types.Format{
				URL: "https://example.com/video.mp4?n=encrypted",
			},
			playerJSURL: "https://example.com/player.js",
			expectedURL: "https://example.com/video.mp4?alr=yes&n=encrypted&ratebypass=yes",
			hasError:    false,
		},
		{
			name: "Format with signatureCipher",
			format: types.Format{
				SignatureCipher: "s=encrypted&sp=sig&url=https%3A%2F%2Fexample.com%2Fvideo.mp4",
			},
			playerJSURL: "https://example.com/player.js",
			expectedURL: "https://example.com/video.mp4?alr=yes&ratebypass=yes&sig=encrypte",
			hasError:    false, // Will succeed but return URL with partial signature
		},
		{
			name: "Format with empty URL and signatureCipher",
			format: types.Format{
				URL:             "",
				SignatureCipher: "",
			},
			playerJSURL: "https://example.com/player.js",
			expectedURL: "",
			hasError:    true,
		},
		{
			name: "Format with invalid URL",
			format: types.Format{
				URL: "://invalid-url",
			},
			playerJSURL: "https://example.com/player.js",
			expectedURL: "",
			hasError:    true,
		},
		{
			name: "Format with URL already having ratebypass",
			format: types.Format{
				URL: "https://example.com/video.mp4?ratebypass=yes",
			},
			playerJSURL: "https://example.com/player.js",
			expectedURL: "https://example.com/video.mp4?alr=yes&ratebypass=yes",
			hasError:    false,
		},
		{
			name: "Format with URL already having alr",
			format: types.Format{
				URL: "https://example.com/video.mp4?alr=yes",
			},
			playerJSURL: "https://example.com/player.js",
			expectedURL: "https://example.com/video.mp4?alr=yes&ratebypass=yes",
			hasError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveFormatURL(httpClient, tt.format, tt.playerJSURL)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if result != tt.expectedURL {
					t.Errorf("Expected URL '%s', got '%s'", tt.expectedURL, result)
				}
			}
		})
	}
}

func TestParseHeight(t *testing.T) {
	tests := []struct {
		name     string
		label    string
		expected int
	}{
		{
			name:     "Valid height 720p",
			label:    "720p",
			expected: 720,
		},
		{
			name:     "Valid height 1080p",
			label:    "1080p",
			expected: 1080,
		},
		{
			name:     "Valid height 480p",
			label:    "480p",
			expected: 480,
		},
		{
			name:     "Valid height 1440p",
			label:    "1440p",
			expected: 1440,
		},
		{
			name:     "Invalid height 2p",
			label:    "2p",
			expected: 0,
		},
		{
			name:     "Invalid height 12p",
			label:    "12p",
			expected: 0,
		},
		{
			name:     "No height in label",
			label:    "audio only",
			expected: 0,
		},
		{
			name:     "Empty label",
			label:    "",
			expected: 0,
		},
		{
			name:     "Height with extra text",
			label:    "720p - High quality",
			expected: 720,
		},
		{
			name:     "Height with prefix text",
			label:    "Video 1080p",
			expected: 1080,
		},
		{
			name:     "Height with multiple numbers",
			label:    "720p 1080p",
			expected: 720, // Should match first occurrence
		},
		{
			name:     "Height with invalid format",
			label:    "720",
			expected: 0,
		},
		{
			name:     "Height with lowercase p",
			label:    "720p",
			expected: 720,
		},
		{
			name:     "Height with uppercase P",
			label:    "720P",
			expected: 0, // Regex is case sensitive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHeight(tt.label)
			if result != tt.expected {
				t.Errorf("Expected height %d, got %d for label: %s", tt.expected, result, tt.label)
			}
		})
	}
}

func TestDecryptSignatures(t *testing.T) {
	// Test with empty formats
	formats := []types.Format{}
	err := DecryptSignatures(&http.Client{}, formats, "https://example.com/player.js")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test with formats that don't need decryption
	formats = []types.Format{
		{URL: "https://example.com/video.mp4"},
		{URL: "https://example.com/video2.mp4"},
	}
	err = DecryptSignatures(&http.Client{}, formats, "https://example.com/player.js")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test with formats that have signature cipher but no URL
	formats = []types.Format{
		{Itag: 22, SignatureCipher: "s=test_signature&sp=signature&url=https://example.com/video.mp4"},
		{Itag: 18, SignatureCipher: "s=test_signature2&sp=signature&url=https://example.com/video2.mp4"},
	}
	err = DecryptSignatures(&http.Client{}, formats, "https://example.com/player.js")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test with formats that have invalid signature cipher
	formats = []types.Format{
		{Itag: 22, SignatureCipher: "invalid_cipher"},
	}
	err = DecryptSignatures(&http.Client{}, formats, "https://example.com/player.js")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test with formats that have signature cipher but missing signature
	formats = []types.Format{
		{Itag: 22, SignatureCipher: "sp=signature&url=https://example.com/video.mp4"},
	}
	err = DecryptSignatures(&http.Client{}, formats, "https://example.com/player.js")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test with formats that have signature cipher but missing URL
	formats = []types.Format{
		{Itag: 22, SignatureCipher: "s=test_signature&sp=signature"},
	}
	err = DecryptSignatures(&http.Client{}, formats, "https://example.com/player.js")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test with formats that have signature cipher but invalid URL
	formats = []types.Format{
		{Itag: 22, SignatureCipher: "s=test_signature&sp=signature&url=invalid_url"},
	}
	err = DecryptSignatures(&http.Client{}, formats, "https://example.com/player.js")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
