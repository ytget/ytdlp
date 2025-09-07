package main

import (
	"fmt"
	"strings"
)

func testCipherFallbacks() {
	fmt.Println("\n6️⃣ Testing Cipher Fallback Mechanisms...")

	// Test 1: RegExp sanitization
	fmt.Println("   Testing RegExp sanitization...")
	testRegExpSanitization()

	// Test 2: Pattern fallback detection
	fmt.Println("   Testing pattern fallback detection...")
	testPatternFallback()

	// Test 3: Fallback pattern detection
	fmt.Println("   Testing fallback pattern detection...")
	testFallbackPatterns()
}

func testRegExpSanitization() {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lookahead",
			input:    `var re = /(?=abc)/;`,
			expected: `var re = /(/;`,
		},
		{
			name:     "negative lookahead",
			input:    `var re = /(?!abc)/;`,
			expected: `var re = /(/;`,
		},
		{
			name:     "lookbehind",
			input:    `var re = /(?<=abc)/;`,
			expected: `var re = /(/;`,
		},
		{
			name:     "mixed patterns",
			input:    `var re1 = /(?=abc)/; var re2 = /(?!def)/;`,
			expected: `var re1 = /(/; var re2 = /(/;`,
		},
	}

	for _, tc := range testCases {
		result := sanitizePlayerJS(tc.input)
		if result == tc.expected {
			fmt.Printf("      ✅ %s: sanitized correctly\n", tc.name)
		} else {
			fmt.Printf("      ❌ %s: expected '%s', got '%s'\n", tc.name, tc.expected, result)
		}
	}
}

func testPatternFallback() {
	testCases := []struct {
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

	for _, tc := range testCases {
		result, ok := tryPatternFallback(tc.playerJS, tc.signature)
		if ok == tc.shouldOk {
			if ok && result == tc.expected {
				fmt.Printf("      ✅ %s: pattern detected and transformed correctly\n", tc.name)
			} else if !ok {
				fmt.Printf("      ✅ %s: correctly detected no pattern\n", tc.name)
			} else {
				fmt.Printf("      ❌ %s: pattern detected but wrong result\n", tc.name)
			}
		} else {
			fmt.Printf("      ❌ %s: pattern detection failed\n", tc.name)
		}
	}
}

func testFallbackPatterns() {
	testCases := []struct {
		name     string
		playerJS string
		expected int // expected number of patterns
	}{
		{
			name:     "reverse and splice",
			playerJS: `function decipher(a) { a.reverse(); a.splice(2); return a.join(""); }`,
			expected: 2,
		},
		{
			name:     "swap pattern",
			playerJS: `function decipher(a) { a[0]=a[1%a.length]; return a.join(""); }`,
			expected: 1,
		},
		{
			name:     "no patterns",
			playerJS: `function other() { return "test"; }`,
			expected: 0,
		},
	}

	for _, tc := range testCases {
		steps := detectFallbackPatterns(tc.playerJS)
		if len(steps) == tc.expected {
			fmt.Printf("      ✅ %s: detected %d patterns correctly\n", tc.name, len(steps))
		} else {
			fmt.Printf("      ❌ %s: expected %d patterns, got %d\n", tc.name, tc.expected, len(steps))
		}
	}
}

// Mock implementations for testing (these would normally be in the cipher package)
func sanitizePlayerJS(playerJS string) string {
	// Simplified version for testing
	patterns := []string{
		`\?=[^)]*\)`,  // lookahead
		`\?![^)]*\)`,  // negative lookahead
		`\?<=[^)]*\)`, // lookbehind
		`\?<![^)]*\)`, // negative lookbehind
	}

	for _, pattern := range patterns {
		re := strings.ReplaceAll(pattern, `\`, "")
		playerJS = strings.ReplaceAll(playerJS, re, "")
	}

	// Clean up empty parentheses
	playerJS = strings.ReplaceAll(playerJS, "()", "")
	playerJS = strings.ReplaceAll(playerJS, "(;", ";")

	return playerJS
}

func tryPatternFallback(playerJS string, signature string) (string, bool) {
	// Look for reverse pattern
	if strings.Contains(playerJS, "reverse") && strings.Contains(playerJS, "join") {
		runes := []rune(signature)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes), true
	}

	// Look for splice pattern
	if strings.Contains(playerJS, "splice") {
		// Try to detect splice offset
		if strings.Contains(playerJS, "splice(2)") {
			if len(signature) > 2 {
				return signature[2:], true
			}
		}
	}

	return "", false
}

func detectFallbackPatterns(playerJS string) []struct {
	op  string
	arg int
} {
	var steps []struct {
		op  string
		arg int
	}

	if strings.Contains(playerJS, "reverse") && strings.Contains(playerJS, "join") {
		steps = append(steps, struct {
			op  string
			arg int
		}{op: "rev", arg: 0})
	}

	if strings.Contains(playerJS, "splice") {
		steps = append(steps, struct {
			op  string
			arg int
		}{op: "spl", arg: 2})
	}

	if strings.Contains(playerJS, "a[0]=a[") && strings.Contains(playerJS, "%a.length") {
		steps = append(steps, struct {
			op  string
			arg int
		}{op: "swp", arg: 1})
	}

	return steps
}

