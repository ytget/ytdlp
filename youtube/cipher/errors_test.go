package cipher

import (
	"encoding/json"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name: "error with details",
			err: &Error{
				Code:    ErrCodeSignatureDecipher,
				Message: "Failed to decipher",
				Details: map[string]any{"signature": "abc123"},
			},
			expected: "SIGNATURE_DECIPHER_FAILED: Failed to decipher (map[signature:abc123])",
		},
		{
			name: "error without details",
			err: &Error{
				Code:    ErrCodePlayerJSNotFound,
				Message: "Player.js not found",
			},
			expected: "PLAYER_JS_NOT_FOUND: Player.js not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestError_MarshalJSON(t *testing.T) {
	err := &Error{
		Code:    ErrCodeSignatureDecipher,
		Message: "Failed to decipher",
		Details: map[string]any{
			"signature": "abc123",
			"attempts":  []string{"regex", "miniJS"},
		},
	}

	data, err2 := json.Marshal(err)
	if err2 != nil {
		t.Fatalf("Failed to marshal error: %v", err2)
	}

	var result map[string]any
	if err2 := json.Unmarshal(data, &result); err2 != nil {
		t.Fatalf("Failed to unmarshal error: %v", err2)
	}

	// Check required fields
	if code, ok := result["code"].(string); !ok || code != ErrCodeSignatureDecipher {
		t.Errorf("Wrong code in JSON: %v", result["code"])
	}
	if msg, ok := result["message"].(string); !ok || msg != "Failed to decipher" {
		t.Errorf("Wrong message in JSON: %v", result["message"])
	}
	if errStr, ok := result["error"].(string); !ok || errStr != err.Error() {
		t.Errorf("Wrong error string in JSON: %v", result["error"])
	}

	// Check details
	details, ok := result["details"].(map[string]any)
	if !ok {
		t.Fatal("Details missing or wrong type")
	}
	if sig, ok := details["signature"].(string); !ok || sig != "abc123" {
		t.Errorf("Wrong signature in details: %v", details["signature"])
	}
	attempts, ok := details["attempts"].([]any)
	if !ok || len(attempts) != 2 {
		t.Errorf("Wrong attempts in details: %v", details["attempts"])
	}
}

func TestErrorHelpers(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		isTO    bool
		isNF    bool
		isInv   bool
		isJS    bool
		isRegex bool
	}{
		{
			name:    "timeout error",
			err:     NewError(ErrCodeSignatureTimeout, "Timeout"),
			isTO:    true,
			isNF:    false,
			isInv:   false,
			isJS:    false,
			isRegex: false,
		},
		{
			name:    "not found error (player.js)",
			err:     NewError(ErrCodePlayerJSNotFound, "Not found"),
			isTO:    false,
			isNF:    true,
			isInv:   false,
			isJS:    false,
			isRegex: false,
		},
		{
			name:    "not found error (signature)",
			err:     NewError(ErrCodeSignatureNotFound, "Not found"),
			isTO:    false,
			isNF:    true,
			isInv:   false,
			isJS:    false,
			isRegex: false,
		},
		{
			name:    "invalid error",
			err:     NewError(ErrCodeSignatureInvalid, "Invalid"),
			isTO:    false,
			isNF:    false,
			isInv:   true,
			isJS:    false,
			isRegex: false,
		},
		{
			name:    "js execution error",
			err:     NewError(ErrCodeJSExecutionFailed, "JS failed"),
			isTO:    false,
			isNF:    false,
			isInv:   false,
			isJS:    true,
			isRegex: false,
		},
		{
			name:    "js parsing error",
			err:     NewError(ErrCodeJSParsingFailed, "JS parse failed"),
			isTO:    false,
			isNF:    false,
			isInv:   false,
			isJS:    true,
			isRegex: false,
		},
		{
			name:    "regex error",
			err:     NewError(ErrCodeRegexParsingFailed, "Regex failed"),
			isTO:    false,
			isNF:    false,
			isInv:   false,
			isJS:    false,
			isRegex: true,
		},
		{
			name:    "non-Error type",
			err:     nil,
			isTO:    false,
			isNF:    false,
			isInv:   false,
			isJS:    false,
			isRegex: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTimeout(tt.err); got != tt.isTO {
				t.Errorf("IsTimeout() = %v, want %v", got, tt.isTO)
			}
			if got := IsNotFound(tt.err); got != tt.isNF {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.isNF)
			}
			if got := IsInvalid(tt.err); got != tt.isInv {
				t.Errorf("IsInvalid() = %v, want %v", got, tt.isInv)
			}
			if got := IsJSError(tt.err); got != tt.isJS {
				t.Errorf("IsJSError() = %v, want %v", got, tt.isJS)
			}
			if got := IsRegexError(tt.err); got != tt.isRegex {
				t.Errorf("IsRegexError() = %v, want %v", got, tt.isRegex)
			}
		})
	}
}




