package cipher

import (
	"encoding/json"
	"fmt"
)

// Error codes
const (
	ErrCodePlayerJSNotFound   = "PLAYER_JS_NOT_FOUND"
	ErrCodePlayerJSDownload   = "PLAYER_JS_DOWNLOAD_FAILED"
	ErrCodeSignatureDecipher  = "SIGNATURE_DECIPHER_FAILED"
	ErrCodeSignatureInvalid   = "SIGNATURE_INVALID"
	ErrCodeSignatureTimeout   = "SIGNATURE_TIMEOUT"
	ErrCodeSignatureNotFound  = "SIGNATURE_NOT_FOUND"
	ErrCodeJSExecutionFailed  = "JS_EXECUTION_FAILED"
	ErrCodeJSParsingFailed    = "JS_PARSING_FAILED"
	ErrCodeRegexParsingFailed = "REGEX_PARSING_FAILED"
)

// Error represents a structured error with code and details
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// MarshalJSON implements json.Marshaler
func (e *Error) MarshalJSON() ([]byte, error) {
	type Alias Error
	return json.Marshal(&struct {
		*Alias
		Error string `json:"error"`
	}{
		Alias: (*Alias)(e),
		Error: e.Error(),
	})
}

// NewError creates a new Error with the given code and message
func NewError(code string, message string, details ...any) *Error {
	e := &Error{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		e.Details = details[0]
	}
	return e
}

// IsTimeout returns true if the error is a timeout error
func IsTimeout(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCodeSignatureTimeout
	}
	return false
}

// IsNotFound returns true if the error is a not found error
func IsNotFound(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCodePlayerJSNotFound || e.Code == ErrCodeSignatureNotFound
	}
	return false
}

// IsInvalid returns true if the error is an invalid signature error
func IsInvalid(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCodeSignatureInvalid
	}
	return false
}

// IsJSError returns true if the error is a JavaScript execution error
func IsJSError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCodeJSExecutionFailed || e.Code == ErrCodeJSParsingFailed
	}
	return false
}

// IsRegexError returns true if the error is a regex parsing error
func IsRegexError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCodeRegexParsingFailed
	}
	return false
}
