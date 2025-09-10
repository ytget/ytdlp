package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogger_Levels(t *testing.T) {
	var buf bytes.Buffer
	config := DefaultConfig()
	config.Output = &buf
	config.Level = INFO

	logger := New(config)
	compLogger := logger.WithComponent(ComponentApp)

	// Test that DEBUG messages are filtered out
	compLogger.Debug("This should not appear")
	compLogger.Info("This should appear")
	compLogger.Warn("This should appear")
	compLogger.Error("This should appear")

	output := buf.String()
	if strings.Contains(output, "This should not appear") {
		t.Error("DEBUG message should be filtered out")
	}
	if !strings.Contains(output, "This should appear") {
		t.Error("INFO/WARN/ERROR messages should appear")
	}
}

func TestLogger_Components(t *testing.T) {
	var buf bytes.Buffer
	config := DefaultConfig()
	config.Output = &buf
	config.Components[ComponentDownloader] = false

	logger := New(config)
	appLogger := logger.WithComponent(ComponentApp)
	downloaderLogger := logger.WithComponent(ComponentDownloader)

	appLogger.Info("App message")
	downloaderLogger.Info("Downloader message")

	output := buf.String()
	if !strings.Contains(output, "App message") {
		t.Error("App message should appear")
	}
	if strings.Contains(output, "Downloader message") {
		t.Error("Downloader message should be filtered out")
	}
}

func TestLogger_Formats(t *testing.T) {
	var buf bytes.Buffer
	config := DefaultConfig()
	config.Output = &buf
	config.Format = FormatJSON

	logger := New(config)
	compLogger := logger.WithComponent(ComponentApp)

	compLogger.Info("Test message", map[string]interface{}{
		"key": "value",
	})

	output := buf.String()
	t.Logf("JSON output: %s", output)
	if !strings.Contains(output, `"level"`) {
		t.Error("JSON format should contain level field")
	}
	if !strings.Contains(output, `"component":"app"`) {
		t.Error("JSON format should contain component field")
	}
	if !strings.Contains(output, `"message":"Test message"`) {
		t.Error("JSON format should contain message field")
	}
}

func TestLogger_Fields(t *testing.T) {
	var buf bytes.Buffer
	config := DefaultConfig()
	config.Output = &buf

	logger := New(config)
	compLogger := logger.WithComponent(ComponentApp)

	compLogger.Info("Test message", map[string]interface{}{
		"url":   "https://example.com",
		"count": 42,
	})

	output := buf.String()
	if !strings.Contains(output, "url=https://example.com") {
		t.Error("Fields should be included in output")
	}
	if !strings.Contains(output, "count=42") {
		t.Error("Fields should be included in output")
	}
}

func TestLogger_Timestamp(t *testing.T) {
	var buf bytes.Buffer
	config := DefaultConfig()
	config.Output = &buf
	config.Timestamp = true

	logger := New(config)
	compLogger := logger.WithComponent(ComponentApp)

	compLogger.Info("Test message")

	output := buf.String()
	// Check for timestamp format (YYYY-MM-DD HH:MM:SS)
	if !strings.Contains(output, "2025-") {
		t.Error("Timestamp should be included in output")
	}
}

func TestLogger_Caller(t *testing.T) {
	var buf bytes.Buffer
	config := DefaultConfig()
	config.Output = &buf
	config.ShowCaller = true

	logger := New(config)
	compLogger := logger.WithComponent(ComponentApp)

	compLogger.Info("Test message")

	output := buf.String()
	if !strings.Contains(output, "logger_test.go:") {
		t.Error("Caller information should be included in output")
	}
}

func TestGlobalLogger(t *testing.T) {
	var buf bytes.Buffer
	config := DefaultConfig()
	config.Output = &buf

	SetGlobalLogger(New(config))
	compLogger := WithComponent(ComponentApp)

	compLogger.Info("Global logger test")

	output := buf.String()
	if !strings.Contains(output, "Global logger test") {
		t.Error("Global logger should work")
	}
}

func TestLogger_Concurrency(t *testing.T) {
	var buf bytes.Buffer
	config := DefaultConfig()
	config.Output = &buf

	logger := New(config)
	compLogger := logger.WithComponent(ComponentApp)

	// Test concurrent logging
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			compLogger.Info("Concurrent message", map[string]interface{}{
				"goroutine": i,
			})
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 10 {
		t.Errorf("Expected 10 log lines, got %d", len(lines))
	}
}

func TestLogger_LevelNames(t *testing.T) {
	expected := map[Level]string{
		TRACE: "TRACE",
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
	}

	for level, expectedName := range expected {
		if levelNames[level] != expectedName {
			t.Errorf("Level %d should have name %s, got %s", level, expectedName, levelNames[level])
		}
	}
}

func TestLogger_ComponentConstants(t *testing.T) {
	expected := map[Component]string{
		ComponentApp:        "app",
		ComponentDownloader: "downloader",
		ComponentCipher:     "cipher",
		ComponentInnerTube:  "innertube",
		ComponentClient:     "client",
		ComponentFormat:     "format",
		ComponentBotGuard:   "botguard",
	}

	for component, expectedValue := range expected {
		if string(component) != expectedValue {
			t.Errorf("Component %s should have value %s, got %s", component, expectedValue, string(component))
		}
	}
}
