package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// LogConfig represents the complete logging configuration
type LogConfig struct {
	Level      string            `json:"level"`
	Format     string            `json:"format"`
	Output     string            `json:"output"`
	Components map[string]bool   `json:"components"`
	ShowCaller bool              `json:"show_caller"`
	Timestamp  bool              `json:"timestamp"`
	Rotation   *RotationConfig   `json:"rotation,omitempty"`
	Filters    map[string]string `json:"filters,omitempty"`
}

// RotationConfig represents log rotation configuration
type RotationConfig struct {
	MaxSize    string `json:"max_size"`    // e.g., "100MB", "1GB"
	MaxAge     string `json:"max_age"`     // e.g., "7d", "24h"
	MaxBackups int    `json:"max_backups"` // number of backup files
	Compress   bool   `json:"compress"`    // compress old logs
}

// DefaultLogConfig returns default logging configuration
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:  "INFO",
		Format: "text",
		Output: "stdout",
		Components: map[string]bool{
			"app":        true,
			"downloader": false,
			"cipher":     false,
			"innertube":  false,
			"client":     false,
			"format":     false,
			"botguard":   false,
		},
		ShowCaller: false,
		Timestamp:  false,
		Rotation: &RotationConfig{
			MaxSize:    "100MB",
			MaxAge:     "7d",
			MaxBackups: 3,
			Compress:   true,
		},
		Filters: map[string]string{
			"min_level": "INFO",
		},
	}
}

// LoadConfigFromFile loads configuration from a JSON file
func LoadConfigFromFile(filename string) (*LogConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read config file: %v", err)
	}

	var config LogConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config file: %v", err)
	}

	return &config, nil
}

// SaveConfigToFile saves configuration to a JSON file
func (c *LogConfig) SaveConfigToFile(filename string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %v", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write config file: %v", err)
	}

	return nil
}

// ToLoggerConfig converts LogConfig to logger.Config
func (c *LogConfig) ToLoggerConfig() (*Config, error) {
	// Parse level
	level, err := parseLevel(c.Level)
	if err != nil {
		return nil, fmt.Errorf("parse level: %v", err)
	}

	// Parse format
	format, err := parseFormat(c.Format)
	if err != nil {
		return nil, fmt.Errorf("parse format: %v", err)
	}

	// Parse output
	output, err := parseOutput(c.Output)
	if err != nil {
		return nil, fmt.Errorf("parse output: %v", err)
	}

	// Convert component map
	components := make(map[Component]bool)
	for name, enabled := range c.Components {
		components[Component(name)] = enabled
	}

	return &Config{
		Level:      level,
		Format:     format,
		Output:     output,
		Components: components,
		ShowCaller: c.ShowCaller,
		Timestamp:  c.Timestamp,
	}, nil
}

// parseLevel parses level string to Level enum
func parseLevel(levelStr string) (Level, error) {
	switch strings.ToUpper(levelStr) {
	case "TRACE":
		return TRACE, nil
	case "DEBUG":
		return DEBUG, nil
	case "INFO":
		return INFO, nil
	case "WARN", "WARNING":
		return WARN, nil
	case "ERROR":
		return ERROR, nil
	default:
		return INFO, fmt.Errorf("unknown level: %s", levelStr)
	}
}

// parseFormat parses format string to Format enum
func parseFormat(formatStr string) (Format, error) {
	switch strings.ToLower(formatStr) {
	case "text":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	case "color", "colored":
		return FormatColor, nil
	default:
		return FormatText, fmt.Errorf("unknown format: %s", formatStr)
	}
}

// parseOutput parses output string to io.Writer
func parseOutput(outputStr string) (io.Writer, error) {
	switch strings.ToLower(outputStr) {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	case "null", "none":
		return io.Discard, nil
	default:
		// Check if it's a file path
		if strings.HasPrefix(outputStr, "file:") {
			filePath := strings.TrimPrefix(outputStr, "file:")
			// Create directory if it doesn't exist
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return nil, fmt.Errorf("create log directory: %v", err)
			}
			file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return nil, fmt.Errorf("open log file: %v", err)
			}
			return file, nil
		}
		return nil, fmt.Errorf("unknown output: %s", outputStr)
	}
}

// CreateLoggerFromConfig creates a logger from LogConfig
func CreateLoggerFromConfig(config *LogConfig) (*Logger, error) {
	loggerConfig, err := config.ToLoggerConfig()
	if err != nil {
		return nil, fmt.Errorf("convert config: %v", err)
	}

	return New(loggerConfig), nil
}

// EnvironmentConfig loads configuration from environment variables
func EnvironmentConfig() *LogConfig {
	config := DefaultLogConfig()

	// Override with environment variables
	if level := os.Getenv("YTDLP_LOG_LEVEL"); level != "" {
		config.Level = level
	}
	if format := os.Getenv("YTDLP_LOG_FORMAT"); format != "" {
		config.Format = format
	}
	if output := os.Getenv("YTDLP_LOG_OUTPUT"); output != "" {
		config.Output = output
	}
	if caller := os.Getenv("YTDLP_LOG_CALLER"); caller != "" {
		config.ShowCaller = caller == "true" || caller == "1"
	}
	if timestamp := os.Getenv("YTDLP_LOG_TIMESTAMP"); timestamp != "" {
		config.Timestamp = timestamp == "true" || timestamp == "1"
	}

	// Parse component filters from environment
	if components := os.Getenv("YTDLP_LOG_COMPONENTS"); components != "" {
		config.Components = make(map[string]bool)
		for _, comp := range strings.Split(components, ",") {
			comp = strings.TrimSpace(comp)
			if comp != "" {
				config.Components[comp] = true
			}
		}
	}

	return config
}

// ValidateConfig validates the configuration
func (c *LogConfig) ValidateConfig() error {
	// Validate level
	if _, err := parseLevel(c.Level); err != nil {
		return fmt.Errorf("invalid level: %v", err)
	}

	// Validate format
	if _, err := parseFormat(c.Format); err != nil {
		return fmt.Errorf("invalid format: %v", err)
	}

	// Validate output
	if _, err := parseOutput(c.Output); err != nil {
		return fmt.Errorf("invalid output: %v", err)
	}

	// Validate rotation config
	if c.Rotation != nil {
		if err := c.Rotation.Validate(); err != nil {
			return fmt.Errorf("invalid rotation config: %v", err)
		}
	}

	return nil
}

// Validate validates rotation configuration
func (r *RotationConfig) Validate() error {
	// Validate max size
	if r.MaxSize != "" {
		if _, err := parseSize(r.MaxSize); err != nil {
			return fmt.Errorf("invalid max_size: %v", err)
		}
	}

	// Validate max age
	if r.MaxAge != "" {
		if _, err := parseDuration(r.MaxAge); err != nil {
			return fmt.Errorf("invalid max_age: %v", err)
		}
	}

	// Validate max backups
	if r.MaxBackups < 0 {
		return fmt.Errorf("max_backups must be non-negative")
	}

	return nil
}

// parseSize parses size string (e.g., "100MB", "1GB") to bytes
func parseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0, nil
	}

	// Extract number and unit
	var numStr, unit string
	for i, r := range sizeStr {
		if r >= '0' && r <= '9' {
			numStr += string(r)
		} else {
			unit = sizeStr[i:]
			break
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("no number found in size: %s", sizeStr)
	}

	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse number: %v", err)
	}

	// Convert to bytes based on unit
	unit = strings.ToUpper(unit)
	switch unit {
	case "B", "":
		return num, nil
	case "KB":
		return num * 1024, nil
	case "MB":
		return num * 1024 * 1024, nil
	case "GB":
		return num * 1024 * 1024 * 1024, nil
	case "TB":
		return num * 1024 * 1024 * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("unknown unit: %s", unit)
	}
}

// parseDuration parses duration string (e.g., "7d", "24h", "30m") to time.Duration
func parseDuration(durationStr string) (time.Duration, error) {
	durationStr = strings.TrimSpace(durationStr)
	if durationStr == "" {
		return 0, nil
	}

	// Extract number and unit
	var numStr, unit string
	for i, r := range durationStr {
		if r >= '0' && r <= '9' {
			numStr += string(r)
		} else {
			unit = durationStr[i:]
			break
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("no number found in duration: %s", durationStr)
	}

	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse number: %v", err)
	}

	// Convert to duration based on unit
	unit = strings.ToLower(unit)
	switch unit {
	case "s", "sec", "second", "seconds":
		return time.Duration(num) * time.Second, nil
	case "m", "min", "minute", "minutes":
		return time.Duration(num) * time.Minute, nil
	case "h", "hour", "hours":
		return time.Duration(num) * time.Hour, nil
	case "d", "day", "days":
		return time.Duration(num) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown unit: %s", unit)
	}
}
