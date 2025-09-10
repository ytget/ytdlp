// Package logger provides structured logging functionality for the ytdlp project.
//
// Features:
//   - Multiple log levels (TRACE, DEBUG, INFO, WARN, ERROR)
//   - Component-based filtering
//   - Multiple output formats (text, JSON, color)
//   - Thread-safe operations
//   - Configurable output and formatting
//
// Usage:
//
//	// Get a component logger
//	log := logger.WithComponent(logger.ComponentDownloader)
//
//	// Log messages with different levels
//	log.Info("Starting download", map[string]interface{}{
//		"url": "https://example.com/video.mp4",
//		"size": 1024,
//	})
//
//	// Configure global logger
//	config := logger.DefaultConfig()
//	config.Level = logger.DEBUG
//	config.Format = logger.FormatJSON
//	logger.SetGlobalLogger(logger.New(config))
//
// Components:
//   - ComponentApp: Main application logs
//   - ComponentDownloader: Download process logs
//   - ComponentCipher: Signature decryption logs
//   - ComponentInnerTube: YouTube API logs
//   - ComponentClient: HTTP client logs
//   - ComponentFormat: Format selection logs
//   - ComponentBotGuard: BotGuard verification logs
package logger
