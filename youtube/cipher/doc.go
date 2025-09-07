/*
Package cipher implements YouTube signature decryption.

The package provides functionality to decrypt YouTube video signatures using multiple
methods with fallback and caching support. It is designed to be efficient and reliable,
with built-in performance metrics and error handling.

# Architecture

The package uses a multi-layered approach to signature decryption:

1. Cache Layer
  - Signatures are cached with TTL to avoid repeated decryption
  - player.js content is cached to reduce network requests
  - Periodic cleanup removes expired entries

2. Decryption Layer
  - Multiple decryption methods with fallback:
    a. Regex-based parser (fast, reliable)
    b. Minimal JS environment (medium reliability)
    c. Full otto execution (fallback)
    d. Pattern-based fallback (last resort)

3. Error Handling
  - Structured errors with codes and details
  - JSON serialization support
  - Helper functions for error type checking

4. Metrics
  - Request counts
  - Cache hit/miss ratios
  - Average decryption time
  - Total decryption time

# Usage

Basic usage:

	client := &http.Client{}
	signature := "..."
	playerJSURL := "https://www.youtube.com/s/player/..."

	deciphered, err := cipher.Decipher(client, playerJSURL, signature)
	if err != nil {
		if cipher.IsTimeout(err) {
			// Handle timeout
		}
		return err
	}

Error handling:

	if err != nil {
		switch {
		case cipher.IsTimeout(err):
			// Handle timeout
		case cipher.IsNotFound(err):
			// Handle not found
		case cipher.IsJSError(err):
			// Handle JS execution error
		default:
			// Handle other errors
		}
	}

# Performance

The package includes built-in metrics that can be used to monitor performance:

- Cache hit ratio
- Average decryption time
- Total requests processed

These metrics are logged periodically and can be used for monitoring and optimization.

# Error Codes

The package defines several error codes:

- PLAYER_JS_NOT_FOUND: player.js URL not found in video page
- PLAYER_JS_DOWNLOAD_FAILED: Failed to download player.js
- SIGNATURE_DECIPHER_FAILED: All decryption methods failed
- SIGNATURE_INVALID: Invalid signature format
- SIGNATURE_TIMEOUT: Decryption timeout
- SIGNATURE_NOT_FOUND: Signature not found in video data
- JS_EXECUTION_FAILED: JavaScript execution error
- JS_PARSING_FAILED: JavaScript parsing error
- REGEX_PARSING_FAILED: Regular expression parsing error

# Caching

The package implements two levels of caching:

1. player.js Cache
  - Caches downloaded player.js content
  - TTL: 10 minutes
  - Reduces network requests

2. Signature Cache
  - Caches decrypted signatures
  - TTL: 1 hour
  - Improves performance for repeated videos

Cache cleanup runs every 5 minutes to remove expired entries.

# Limitations

1. JavaScript Execution
  - Full JavaScript execution may be slow
  - Some modern JS features not supported
  - Fallback to regex/pattern matching

2. Network
  - Requires network access for player.js
  - Subject to YouTube API changes
  - Network timeouts possible

3. Memory
  - Cache size grows with unique signatures
  - No maximum cache size limit
  - Regular cleanup required

# Thread Safety

The package is thread-safe:
- All cache operations are protected by mutexes
- Metrics updates are atomic
- HTTP client should be thread-safe

# Dependencies

- github.com/robertkrimen/otto: JavaScript engine
- net/http: HTTP client
- encoding/json: JSON handling
- sync: Concurrency primitives
*/
package cipher
