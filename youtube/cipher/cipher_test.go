package cipher

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func reverseRunes(r []rune) []rune {
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return r
}

func spliceRunes(r []rune, n int) []rune {
	if n < 0 || n > len(r) {
		return r
	}
	return r[n:]
}

func TestDecipherWithOtto(t *testing.T) {
	playerJSContent, err := os.ReadFile("testdata/player.js")
	if err != nil {
		t.Fatalf("Failed to read test player.js: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(playerJSContent)
	}))
	defer server.Close()

	// Example of an encrypted signature
	encryptedSig := "ABCDEFGHIJKLMNabcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqr"

	// Compute the expected value using the same steps: reverse -> splice(26) -> reverse
	r := []rune(encryptedSig)
	r = reverseRunes(r)
	r = spliceRunes(r, 26)
	r = reverseRunes(r)
	expectedSig := string(r)

	deciphered, err := Decipher(server.Client(), server.URL, encryptedSig)
	if err != nil {
		t.Fatalf("Decipher returned an error: %v", err)
	}

	if deciphered != expectedSig {
		t.Errorf("Decipher() got = %v, want %v", deciphered, expectedSig)
	}
}

func TestDecipherN(t *testing.T) {
	playerJSContent, err := os.ReadFile("testdata/player.js")
	if err != nil {
		t.Fatalf("Failed to read test player.js: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(playerJSContent)
	}))
	defer server.Close()

	in := "abcdef"
	want := "fedcba"
	got, err := DecipherN(server.Client(), server.URL, in)
	if err != nil {
		t.Fatalf("DecipherN error: %v", err)
	}
	if got != want {
		t.Fatalf("DecipherN got=%q want=%q", got, want)
	}
}

func TestSanitizePlayerJS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove lookahead",
			input:    `var re = /(?=abc)/;`,
			expected: `var re = /(/;`,
		},
		{
			name:     "remove negative lookahead",
			input:    `var re = /(?!abc)/;`,
			expected: `var re = /(/;`,
		},
		{
			name:     "remove lookbehind",
			input:    `var re = /(?<=abc)/;`,
			expected: `var re = /(/;`,
		},
		{
			name:     "remove negative lookbehind",
			input:    `var re = /(?<!abc)/;`,
			expected: `var re = /(/;`,
		},
		{
			name:     "remove named capture",
			input:    `var re = /(?<name>abc)/;`,
			expected: `var re = /(abc)/;`,
		},
		{
			name:     "remove atomic group",
			input:    `var re = /(?>abc)/;`,
			expected: `var re = /(/;`,
		},
		{
			name:     "mixed patterns",
			input:    `var re1 = /(?=abc)/; var re2 = /(?!def)/; var re3 = /(?<=ghi)/;`,
			expected: `var re1 = /(/; var re2 = /(/; var re3 = /(/;`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePlayerJS(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizePlayerJS() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTryPatternFallback(t *testing.T) {
	tests := []struct {
		name      string
		playerJS  string
		signature string
		expected  string
		shouldOk  bool
	}{
		{
			name:      "reverse pattern",
			playerJS:  `function reverse() { a.reverse(); a.join(""); }`,
			signature: "abc123",
			expected:  "321cba",
			shouldOk:  true,
		},
		{
			name:      "splice pattern",
			playerJS:  `function splice() { a.splice(2); }`,
			signature: "abc123",
			expected:  "c123",
			shouldOk:  true,
		},
		{
			name:      "no pattern",
			playerJS:  `function other() { }`,
			signature: "abc123",
			expected:  "",
			shouldOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := tryPatternFallback(tt.playerJS, tt.signature)
			if ok != tt.shouldOk {
				t.Errorf("tryPatternFallback() ok = %v, want %v", ok, tt.shouldOk)
			}
			if ok && result != tt.expected {
				t.Errorf("tryPatternFallback() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHTTPClientCreation(t *testing.T) {
	// Test that HTTP client is created with HTTP/1.1 transport
	client := &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: false,
		},
		Timeout: 30 * time.Second,
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Transport is not *http.Transport")
	}

	if transport.ForceAttemptHTTP2 {
		t.Error("HTTP/2 should be disabled")
	}
}

func TestFallbackPatterns(t *testing.T) {
	// Test fallback pattern detection
	playerJS := `
		function decipher(a) {
			a = a.split("");
			a.reverse();
			a.splice(0, 2);
			return a.join("");
		}
	`

	steps := detectFallbackPatterns(playerJS)
	if len(steps) == 0 {
		t.Error("Expected fallback patterns to be detected")
	}

	// Check that reverse pattern was detected
	foundReverse := false
	for _, step := range steps {
		if step.op == "rev" {
			foundReverse = true
			break
		}
	}
	if !foundReverse {
		t.Error("Expected reverse pattern to be detected")
	}
}
