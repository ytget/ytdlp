# Logging Policy

This document describes the logging policy and configuration for the ytdlp project.

## Overview

The project uses a structured logging system with the following features:

- **Multiple log levels**: TRACE, DEBUG, INFO, WARN, ERROR
- **Component-based filtering**: Filter logs by component (app, downloader, cipher, etc.)
- **Multiple output formats**: Text, JSON, and colored text
- **Log rotation**: Automatic rotation based on size and age
- **Thread-safe operations**: Safe for concurrent use
- **Configurable output**: File, stdout, stderr, or custom writer

## Log Levels

### TRACE
- Maximum verbosity
- HTTP request/response headers
- Detailed internal operations
- Use only for debugging

### DEBUG
- Detailed debugging information
- Internal state changes
- Performance metrics
- Useful for development

### INFO
- General information about operations
- Important state changes
- User-relevant information
- Default level for production

### WARN
- Warning conditions
- Non-critical errors
- Deprecated functionality
- Recoverable errors

### ERROR
- Error conditions
- Critical failures
- Unrecoverable errors
- Always logged

## Components

### app
Main application logs, high-level operations

### downloader
Download process logs, chunk operations, progress

### cipher
Signature decryption logs, player.js operations

### innertube
YouTube API logs, InnerTube requests

### client
HTTP client logs, connection details

### format
Format selection logs, quality detection

### botguard
BotGuard verification logs, token operations

## Configuration

### Configuration File

Create `config/logging.json`:

```json
{
  "level": "INFO",
  "format": "text",
  "output": "stdout",
  "components": {
    "app": true,
    "downloader": true,
    "cipher": false,
    "innertube": false,
    "client": false,
    "format": false,
    "botguard": false
  },
  "show_caller": false,
  "timestamp": true,
  "rotation": {
    "max_size": "100MB",
    "max_age": "7d",
    "max_backups": 3,
    "compress": true
  }
}
```

### Environment Variables

Override configuration with environment variables:

```bash
export YTDLP_LOG_LEVEL=DEBUG
export YTDLP_LOG_FORMAT=json
export YTDLP_LOG_OUTPUT=file:logs/ytdlp.log
export YTDLP_LOG_CALLER=true
export YTDLP_LOG_TIMESTAMP=true
export YTDLP_LOG_COMPONENTS=app,downloader
```

### Output Formats

#### Text Format (default)
```
2025-01-27 12:00:00 [INFO] [app] Starting download url=https://example.com
```

#### JSON Format
```json
{
  "timestamp": "2025-01-27T12:00:00Z",
  "level": "INFO",
  "component": "app",
  "message": "Starting download",
  "fields": {
    "url": "https://example.com"
  }
}
```

#### Color Format
Colored output for terminal with syntax highlighting.

### Output Destinations

- `stdout`: Standard output (default)
- `stderr`: Standard error
- `file:path/to/file.log`: File output
- `null`: Discard all logs

### Log Rotation

Configure automatic log rotation:

```json
{
  "rotation": {
    "max_size": "100MB",    // Rotate when file exceeds 100MB
    "max_age": "7d",        // Rotate files older than 7 days
    "max_backups": 3,       // Keep 3 backup files
    "compress": true        // Compress old log files
  }
}
```

## Usage Examples

### Basic Usage

```go
import "github.com/ytget/ytdlp/internal/logger"

// Get component logger
log := logger.WithComponent(logger.ComponentApp)

// Log messages
log.Info("Starting operation")
log.Debug("Debug information", map[string]interface{}{
    "key": "value",
})
log.Error("Operation failed", map[string]interface{}{
    "error": err,
})
```

### Custom Logger

```go
// Create custom logger
config := logger.DefaultConfig()
config.Level = logger.DEBUG
config.Format = logger.FormatJSON
config.Output = os.Stdout

logger := logger.New(config)
compLogger := logger.WithComponent(logger.ComponentDownloader)
```

### Global Logger

```go
// Set global logger
logger.SetGlobalLogger(customLogger)

// Use global logger
log := logger.WithComponent(logger.ComponentApp)
log.Info("Using global logger")
```

## Best Practices

### 1. Use Appropriate Log Levels

- **ERROR**: Only for actual errors that prevent operation
- **WARN**: For recoverable issues or deprecated usage
- **INFO**: For important state changes and user information
- **DEBUG**: For detailed debugging information
- **TRACE**: For maximum verbosity (HTTP headers, etc.)

### 2. Include Context

Always include relevant context in log messages:

```go
// Good
log.Info("Download started", map[string]interface{}{
    "url": url,
    "size": totalSize,
    "format": format,
})

// Bad
log.Info("Download started")
```

### 3. Use Structured Fields

Prefer structured fields over string formatting:

```go
// Good
log.Error("Download failed", map[string]interface{}{
    "error": err,
    "url": url,
    "attempt": attempt,
})

// Bad
log.Error(fmt.Sprintf("Download failed: %v (url: %s, attempt: %d)", err, url, attempt))
```

### 4. Avoid Sensitive Information

Never log sensitive information like passwords, tokens, or personal data:

```go
// Good
log.Debug("Request sent", map[string]interface{}{
    "url": url,
    "method": "POST",
})

// Bad
log.Debug("Request sent", map[string]interface{}{
    "url": url,
    "headers": headers, // May contain sensitive data
})
```

### 5. Use Component Filtering

Enable only necessary components for your use case:

```json
{
  "components": {
    "app": true,        // Always enable for main operations
    "downloader": true, // Enable for download debugging
    "cipher": false,    // Disable unless debugging cipher issues
    "innertube": false, // Disable unless debugging API issues
    "client": false,    // Disable unless debugging HTTP issues
    "format": false,    // Disable unless debugging format selection
    "botguard": false   // Disable unless debugging BotGuard
  }
}
```

## Troubleshooting

### Too Many Logs

If you're getting too many logs:

1. Increase log level to WARN or ERROR
2. Disable unnecessary components
3. Use TRACE level only for debugging

### Missing Logs

If logs are missing:

1. Check log level configuration
2. Verify component is enabled
3. Check output destination
4. Ensure logger is properly initialized

### Performance Issues

If logging affects performance:

1. Use INFO level or higher
2. Disable caller information (`show_caller: false`)
3. Use file output instead of stdout
4. Enable log rotation

### Log Rotation Issues

If log rotation isn't working:

1. Check file permissions
2. Verify rotation configuration
3. Ensure sufficient disk space
4. Check for file locks

## Migration from Old Logging

The old logging system used `log.Printf` and `fmt.Print*` functions. To migrate:

1. Replace `log.Printf` with component logger
2. Replace `fmt.Print*` with appropriate log level
3. Add structured fields for context
4. Remove debug prints from production code

### Before

```go
log.Printf("Downloader: Starting download to %s", outputPath)
log.Printf("Downloader: Request headers:")
for k, v := range req.Header {
    log.Printf("  %s: %s", k, v)
}
```

### After

```go
downloaderLogger.Info("Starting download", map[string]interface{}{
    "output_path": outputPath,
})
downloaderLogger.Trace("Request headers", map[string]interface{}{
    "headers": req.Header,
})
```

## Configuration Examples

### Minimal (Default)

```json
{
  "level": "INFO",
  "format": "text",
  "output": "stdout",
  "components": {
    "app": true,
    "downloader": false,
    "cipher": false,
    "innertube": false,
    "client": false,
    "format": false,
    "botguard": false
  },
  "show_caller": false,
  "timestamp": false
}
```

### Development

```json
{
  "level": "DEBUG",
  "format": "color",
  "output": "stdout",
  "components": {
    "app": true,
    "downloader": true,
    "cipher": false,
    "innertube": false,
    "client": false,
    "format": false,
    "botguard": false
  },
  "show_caller": true,
  "timestamp": true
}
```

### Production

```json
{
  "level": "INFO",
  "format": "json",
  "output": "file:logs/ytdlp.log",
  "components": {
    "app": true,
    "downloader": false,
    "cipher": false,
    "innertube": false,
    "client": false,
    "format": false,
    "botguard": false
  },
  "show_caller": false,
  "timestamp": true,
  "rotation": {
    "max_size": "100MB",
    "max_age": "7d",
    "max_backups": 3,
    "compress": true
  }
}
```

### Debugging

```json
{
  "level": "TRACE",
  "format": "text",
  "output": "stdout",
  "components": {
    "app": true,
    "downloader": true,
    "cipher": true,
    "innertube": true,
    "client": true,
    "format": true,
    "botguard": true
  },
  "show_caller": true,
  "timestamp": true
}
```
