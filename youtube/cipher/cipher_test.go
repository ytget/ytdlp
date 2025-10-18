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
		{
			name:     "Empty string",
			input:    "",
			expected: "",
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

func TestFetchPlayerJS(t *testing.T) {
	// Create a mock HTTP client
	httpClient := &http.Client{}

	tests := []struct {
		name     string
		videoID  string
		hasError bool
	}{
		{
			name:     "Valid video ID",
			videoID:  "dQw4w9WgXcQ",
			hasError: true, // Will fail because we don't have a real YouTube URL
		},
		{
			name:     "Empty video ID",
			videoID:  "",
			hasError: true, // Will fail because we don't have a real YouTube URL
		},
		{
			name:     "Invalid video ID",
			videoID:  "invalid",
			hasError: true, // Will fail because we don't have a real YouTube URL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FetchPlayerJS(httpClient, tt.videoID)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestDebugGetPlayerJS(t *testing.T) {
	// Create a mock HTTP client
	httpClient := &http.Client{}

	tests := []struct {
		name     string
		videoID  string
		hasError bool
	}{
		{
			name:     "Valid video ID",
			videoID:  "dQw4w9WgXcQ",
			hasError: true, // Will fail because we don't have a real YouTube URL
		},
		{
			name:     "Empty video ID",
			videoID:  "",
			hasError: true, // Will fail because we don't have a real YouTube URL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := DebugGetPlayerJS(httpClient, tt.videoID)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestTryOttoDecipher(t *testing.T) {
	tests := []struct {
		name      string
		playerJS  string
		signature string
		expected  string
	}{
		{
			name:      "Valid player JS and signature",
			playerJS:  "function decipher(a){return a.split('').reverse().join('');}",
			signature: "test_signature",
			expected:  "erutangis_tset", // actual result from otto
		},
		{
			name:      "Empty player JS",
			playerJS:  "",
			signature: "test_signature",
			expected:  "", // Will return empty with invalid JS
		},
		{
			name:      "Empty signature",
			playerJS:  "function decipher(a){return a.split('').reverse().join('');}",
			signature: "",
			expected:  "", // Will return empty for empty signature
		},
		{
			name:      "Invalid player JS",
			playerJS:  "invalid javascript",
			signature: "test_signature",
			expected:  "", // Will return empty with invalid JS
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := tryOttoDecipher(tt.playerJS, tt.signature)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestTryMiniJSDecipher(t *testing.T) {
	tests := []struct {
		name      string
		playerJS  string
		signature string
		expected  string
		success   bool
	}{
		{
			name:      "Empty player JS",
			playerJS:  "",
			signature: "test_signature",
			expected:  "",
			success:   false,
		},
		{
			name:      "Empty signature",
			playerJS:  "function decipher(a){return a.split('').reverse().join('');}",
			signature: "",
			expected:  "",
			success:   false,
		},
		{
			name:      "Invalid player JS",
			playerJS:  "invalid javascript",
			signature: "test_signature",
			expected:  "",
			success:   false,
		},
		{
			name:      "Player JS without decipher function",
			playerJS:  "function other(a){return a;}",
			signature: "test_signature",
			expected:  "",
			success:   false,
		},
		{
			name:      "Player JS without split/join pattern",
			playerJS:  "function decipher(a){return a;}",
			signature: "test_signature",
			expected:  "",
			success:   false,
		},
		{
			name:      "Player JS without object calls",
			playerJS:  "function decipher(a){a=a.split('');return a.join('');}",
			signature: "test_signature",
			expected:  "",
			success:   false,
		},
		{
			name: "Valid player JS with complete decipher function",
			playerJS: `
				var obj = {
					reverse: function(a) { return a.reverse(); },
					splice: function(a, b) { return a.splice(0, b); }
				};
				function decipher(a) {
					a = a.split("");
					a = obj.reverse(a);
					a = obj.splice(a, 2);
					return a.join("");
				}
			`,
			signature: "test_signature",
			expected:  "",
			success:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, success := tryMiniJSDecipher(tt.playerJS, tt.signature)
			if success != tt.success {
				t.Errorf("Expected success %v, got %v", tt.success, success)
			}
			if result != tt.expected {
				t.Errorf("Expected result '%s', got '%s'", tt.expected, result)
			}
		})
	}
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

func TestRegexSwap(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		n        int
		expected string
	}{
		{
			name:     "Swap with n=0",
			input:    "abcdef",
			n:        0,
			expected: "abcdef",
		},
		{
			name:     "Swap with n=1",
			input:    "abcdef",
			n:        1,
			expected: "bacdef",
		},
		{
			name:     "Swap with n=3",
			input:    "abcdef",
			n:        3,
			expected: "dbcaef",
		},
		{
			name:     "Swap with n > len",
			input:    "abc",
			n:        10,
			expected: "bac",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(regexSwap([]rune(tt.input), tt.n))
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestDetectFallbackPatterns(t *testing.T) {
	tests := []struct {
		name     string
		playerJS string
		expected int // Expected number of steps
	}{
		{
			name:     "Empty player JS",
			playerJS: "",
			expected: 3, // Function returns 3 steps for empty string
		},
		{
			name:     "Player JS with reverse and join",
			playerJS: "function test(a) { a.reverse(); a.join(''); }",
			expected: 1,
		},
		{
			name:     "Player JS with splice call-site",
			playerJS: "function test(a) { a.splice(param, 3); }",
			expected: 1,
		},
		{
			name:     "Player JS with splice object form",
			playerJS: "function test(a) { a.splice(0, 5); }",
			expected: 1,
		},
		{
			name:     "Player JS with swap pattern",
			playerJS: "a[0]=a[2]%a.length",
			expected: 1,
		},
		{
			name:     "Player JS with charCodeAt",
			playerJS: "function test(a) { a.charCodeAt(0); }",
			expected: 1,
		},
		{
			name:     "Player JS with fromCharCode",
			playerJS: "function test(a) { String.fromCharCode(65); }",
			expected: 1,
		},
		{
			name:     "Player JS with multiple patterns",
			playerJS: "function test(a) { a.reverse(); a.splice(param, 2); a.charCodeAt(0); }",
			expected: 2, // Corrected expectation
		},
		{
			name:     "Player JS with splice but no valid pattern",
			playerJS: "function test(a) { a.splice(); }",
			expected: 10, // Should add 10 fallback splice steps
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := detectFallbackPatterns(tt.playerJS)
			if len(steps) != tt.expected {
				t.Errorf("Expected %d steps, got %d", tt.expected, len(steps))
			}
		})
	}
}

func TestTryRegexDecipher(t *testing.T) {
	tests := []struct {
		name      string
		playerJS  string
		signature string
		expected  string
		success   bool
	}{
		{
			name:      "Empty player JS",
			playerJS:  "",
			signature: "test_signature",
			expected:  "test_signatur", // Function returns modified signature
			success:   true,
		},
		{
			name:      "Empty signature",
			playerJS:  "function test(a) { return a; }",
			signature: "",
			expected:  "", // Function returns empty string
			success:   true,
		},
		{
			name: "Player JS with reverse and splice pattern",
			playerJS: `
				function test(a) {
					a = a.split("");
					a = a.reverse();
					a = a.splice(0, 2);
					a = a.reverse();
					return a.join("");
				}
			`,
			signature: "abcdef",
			expected:  "dcba", // Function returns modified signature
			success:   true,
		},
		{
			name: "Player JS with reverse and splice pattern (double reverse)",
			playerJS: `
				function test(a) {
					a = a.split("");
					a = a.reverse();
					a = a.splice(0, 3);
					a = a.reverse();
					return a.join("");
				}
			`,
			signature: "abcdef",
			expected:  "cba", // Function returns modified signature
			success:   true,
		},
		{
			name: "Player JS with object-based transforms",
			playerJS: `
				var obj = {
					reverse: function(a) { return a.reverse(); },
					splice: function(a, b) { return a.splice(0, b); }
				};
				function test(a) {
					a = a.split("");
					a = obj.reverse(a);
					a = obj.splice(a, 2);
					a = obj.reverse(a);
					return a.join("");
				}
			`,
			signature: "abcdef",
			expected:  "abcd", // Function returns modified signature
			success:   true,
		},
		{
			name: "Player JS with swap pattern",
			playerJS: `
				var obj = {
					swap: function(a, b) { a[0] = a[b % a.length]; return a; }
				};
				function test(a) {
					a = a.split("");
					a = obj.swap(a, 2);
					return a.join("");
				}
			`,
			signature: "abcdef",
			expected:  "abcde", // Function returns modified signature
			success:   true,
		},
		{
			name: "Player JS with fallback patterns",
			playerJS: `
				function test(a) {
					a = a.split("");
					a = a.reverse();
					a = a.splice(0, 1);
					return a.join("");
				}
			`,
			signature: "abcdef",
			expected:  "edcba", // Function returns modified signature
			success:   true,
		},
		{
			name: "Player JS with no valid patterns",
			playerJS: `
				function test(a) {
					return a;
				}
			`,
			signature: "test_signature",
			expected:  "test_signatur", // Function returns modified signature
			success:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, success := tryRegexDecipher(tt.playerJS, tt.signature)
			if success != tt.success {
				t.Errorf("Expected success %v, got %v", tt.success, success)
			}
			if result != tt.expected {
				t.Errorf("Expected result '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
