# YouTube Cipher Package

This package implements YouTube signature decryption with caching, metrics, and error handling.

## Features

- Multiple decryption methods with fallback
- Signature and player.js caching
- Performance metrics
- Structured error handling
- Thread-safe operations

## Usage

```go
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
```

## Error Handling

The package provides structured errors with codes and helper functions:

```go
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
```

## Metrics

The package collects performance metrics:

- Total requests
- Cache hit/miss ratio
- Average decryption time
- Total decryption time

Metrics are logged periodically and can be used for monitoring.

## Caching

Two levels of caching are implemented:

1. player.js Cache
   - TTL: 10 minutes
   - Reduces network requests

2. Signature Cache
   - TTL: 1 hour
   - Improves performance

Cache cleanup runs every 5 minutes.

## Architecture

The package uses a multi-layered approach:

1. Cache Layer
   - Signature caching
   - player.js caching
   - Periodic cleanup

2. Decryption Layer
   - Regex-based parser
   - Minimal JS environment
   - Full otto execution
   - Pattern-based fallback

3. Error Handling
   - Structured errors
   - JSON serialization
   - Helper functions

4. Metrics
   - Request tracking
   - Cache statistics
   - Timing measurements

## Thread Safety

The package is thread-safe:
- Cache operations use mutexes
- Metrics updates are atomic
- HTTP client should be thread-safe

## Dependencies

- github.com/robertkrimen/otto
- net/http
- encoding/json
- sync

## Limitations

1. JavaScript Execution
   - Full JS execution may be slow
   - Limited modern JS support
   - Fallback mechanisms used

2. Network
   - Requires network access
   - Subject to API changes
   - Network timeouts possible

3. Memory
   - Cache size grows with usage
   - No size limits
   - Regular cleanup needed





