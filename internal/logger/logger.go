package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Level represents the logging level
type Level int

const (
	TRACE Level = iota
	DEBUG
	INFO
	WARN
	ERROR
)

var levelNames = map[Level]string{
	TRACE: "TRACE",
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
}

// Component represents the logging component
type Component string

const (
	ComponentApp        Component = "app"
	ComponentDownloader Component = "downloader"
	ComponentCipher     Component = "cipher"
	ComponentInnerTube  Component = "innertube"
	ComponentClient     Component = "client"
	ComponentFormat     Component = "format"
	ComponentBotGuard   Component = "botguard"
)

// Format represents the log output format
type Format int

const (
	FormatText Format = iota
	FormatJSON
	FormatColor
)

// Config holds logger configuration
type Config struct {
	Level      Level
	Format     Format
	Output     io.Writer
	Components map[Component]bool
	ShowCaller bool
	Timestamp  bool
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:  INFO,
		Format: FormatText,
		Output: os.Stdout,
		Components: map[Component]bool{
			ComponentApp:        true,
			ComponentDownloader: false,
			ComponentCipher:     false,
			ComponentInnerTube:  false,
			ComponentClient:     false,
			ComponentFormat:     false,
			ComponentBotGuard:   false,
		},
		ShowCaller: false,
		Timestamp:  false,
	}
}

// Entry represents a single log entry
type Entry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     Level                  `json:"level"`
	Component Component              `json:"component"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
}

// Logger provides structured logging functionality
type Logger struct {
	config *Config
	mu     sync.RWMutex
}

// New creates a new logger instance
func New(config *Config) *Logger {
	if config == nil {
		config = DefaultConfig()
	}
	return &Logger{
		config: config,
	}
}

// WithComponent creates a new logger instance for a specific component
func (l *Logger) WithComponent(component Component) *ComponentLogger {
	return &ComponentLogger{
		logger:    l,
		component: component,
	}
}

// SetLevel changes the logging level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Level = level
}

// SetFormat changes the log format
func (l *Logger) SetFormat(format Format) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Format = format
}

// SetOutput changes the log output
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Output = w
}

// EnableComponent enables logging for a specific component
func (l *Logger) EnableComponent(component Component) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Components[component] = true
}

// DisableComponent disables logging for a specific component
func (l *Logger) DisableComponent(component Component) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Components[component] = false
}

// log writes a log entry
func (l *Logger) log(level Level, component Component, message string, fields map[string]interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Check if level is enabled
	if level < l.config.Level {
		return
	}

	// Check if component is enabled
	if !l.config.Components[component] {
		return
	}

	entry := Entry{
		Timestamp: time.Now(),
		Level:     level,
		Component: component,
		Message:   message,
		Fields:    fields,
	}

	// Add caller information if enabled
	if l.config.ShowCaller {
		// Note: Caller information disabled to avoid runtime dependency
		// entry.Caller = "caller_info_disabled"
	}

	l.writeEntry(entry)
}

// writeEntry writes the log entry to output
func (l *Logger) writeEntry(entry Entry) {
	var output string

	switch l.config.Format {
	case FormatJSON:
		output = l.formatJSON(entry)
	case FormatColor:
		output = l.formatColor(entry)
	default:
		output = l.formatText(entry)
	}

	fmt.Fprintln(l.config.Output, output)
}

// formatText formats entry as plain text
func (l *Logger) formatText(entry Entry) string {
	var parts []string

	if l.config.Timestamp {
		parts = append(parts, entry.Timestamp.Format("2006-01-02 15:04:05"))
	}

	parts = append(parts, fmt.Sprintf("[%s]", levelNames[entry.Level]))
	parts = append(parts, fmt.Sprintf("[%s]", entry.Component))
	parts = append(parts, entry.Message)

	if entry.Caller != "" {
		parts = append(parts, fmt.Sprintf("(%s)", entry.Caller))
	}

	if len(entry.Fields) > 0 {
		var fieldParts []string
		for k, v := range entry.Fields {
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", k, v))
		}
		parts = append(parts, strings.Join(fieldParts, " "))
	}

	return strings.Join(parts, " ")
}

// formatJSON formats entry as JSON
func (l *Logger) formatJSON(entry Entry) string {
	data, _ := json.Marshal(entry)
	return string(data)
}

// formatColor formats entry with colors
func (l *Logger) formatColor(entry Entry) string {
	var parts []string

	if l.config.Timestamp {
		parts = append(parts, "\033[90m"+entry.Timestamp.Format("2006-01-02 15:04:05")+"\033[0m")
	}

	// Color by level
	levelColor := l.getLevelColor(entry.Level)
	parts = append(parts, fmt.Sprintf("%s[%s]\033[0m", levelColor, levelNames[entry.Level]))
	parts = append(parts, fmt.Sprintf("\033[36m[%s]\033[0m", entry.Component))
	parts = append(parts, entry.Message)

	if entry.Caller != "" {
		parts = append(parts, fmt.Sprintf("\033[90m(%s)\033[0m", entry.Caller))
	}

	if len(entry.Fields) > 0 {
		var fieldParts []string
		for k, v := range entry.Fields {
			fieldParts = append(fieldParts, fmt.Sprintf("\033[33m%s\033[0m=\033[32m%v\033[0m", k, v))
		}
		parts = append(parts, strings.Join(fieldParts, " "))
	}

	return strings.Join(parts, " ")
}

// getLevelColor returns color code for log level
func (l *Logger) getLevelColor(level Level) string {
	switch level {
	case TRACE:
		return "\033[37m" // White
	case DEBUG:
		return "\033[94m" // Blue
	case INFO:
		return "\033[92m" // Green
	case WARN:
		return "\033[93m" // Yellow
	case ERROR:
		return "\033[91m" // Red
	default:
		return "\033[0m" // Reset
	}
}

// ComponentLogger provides component-specific logging
type ComponentLogger struct {
	logger    *Logger
	component Component
}

// Trace logs a trace message
func (cl *ComponentLogger) Trace(message string, fields ...map[string]interface{}) {
	cl.log(TRACE, message, fields...)
}

// Debug logs a debug message
func (cl *ComponentLogger) Debug(message string, fields ...map[string]interface{}) {
	cl.log(DEBUG, message, fields...)
}

// Info logs an info message
func (cl *ComponentLogger) Info(message string, fields ...map[string]interface{}) {
	cl.log(INFO, message, fields...)
}

// Warn logs a warning message
func (cl *ComponentLogger) Warn(message string, fields ...map[string]interface{}) {
	cl.log(WARN, message, fields...)
}

// Error logs an error message
func (cl *ComponentLogger) Error(message string, fields ...map[string]interface{}) {
	cl.log(ERROR, message, fields...)
}

// log writes a log entry for the component
func (cl *ComponentLogger) log(level Level, message string, fields ...map[string]interface{}) {
	var mergedFields map[string]interface{}
	if len(fields) > 0 {
		mergedFields = fields[0]
	}
	cl.logger.log(level, cl.component, message, mergedFields)
}

// Global logger instance
var globalLogger = New(DefaultConfig())

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	return globalLogger
}

// WithComponent returns a component logger from global logger
func WithComponent(component Component) *ComponentLogger {
	return globalLogger.WithComponent(component)
}
