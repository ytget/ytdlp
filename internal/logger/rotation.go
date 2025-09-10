package logger

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RotatingWriter implements log rotation functionality
type RotatingWriter struct {
	filename   string
	maxSize    int64
	maxAge     time.Duration
	maxBackups int
	compress   bool
	file       *os.File
	size       int64
	mu         sync.Mutex
	lastRotate time.Time
}

// NewRotatingWriter creates a new rotating writer
func NewRotatingWriter(filename string, maxSize int64, maxAge time.Duration, maxBackups int, compress bool) (*RotatingWriter, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return nil, fmt.Errorf("create log directory: %v", err)
	}

	// Open or create the log file
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %v", err)
	}

	// Get current file size
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("stat log file: %v", err)
	}

	rw := &RotatingWriter{
		filename:   filename,
		maxSize:    maxSize,
		maxAge:     maxAge,
		maxBackups: maxBackups,
		compress:   compress,
		file:       file,
		size:       stat.Size(),
		lastRotate: time.Now(),
	}

	return rw, nil
}

// Write implements io.Writer interface
func (rw *RotatingWriter) Write(p []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// Check if rotation is needed
	if rw.needsRotation() {
		if err := rw.rotate(); err != nil {
			return 0, fmt.Errorf("rotate log file: %v", err)
		}
	}

	// Write to current file
	n, err = rw.file.Write(p)
	if err != nil {
		return n, err
	}

	rw.size += int64(n)
	return n, nil
}

// Close closes the rotating writer
func (rw *RotatingWriter) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file != nil {
		return rw.file.Close()
	}
	return nil
}

// needsRotation checks if log rotation is needed
func (rw *RotatingWriter) needsRotation() bool {
	// Check size limit
	if rw.maxSize > 0 && rw.size >= rw.maxSize {
		return true
	}

	// Check age limit
	if rw.maxAge > 0 && time.Since(rw.lastRotate) >= rw.maxAge {
		return true
	}

	return false
}

// rotate performs log rotation
func (rw *RotatingWriter) rotate() error {
	// Close current file
	if err := rw.file.Close(); err != nil {
		return fmt.Errorf("close current file: %v", err)
	}

	// Generate rotated filename with timestamp
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	rotatedFilename := fmt.Sprintf("%s.%s", rw.filename, timestamp)

	// Rename current file to rotated filename
	if err := os.Rename(rw.filename, rotatedFilename); err != nil {
		return fmt.Errorf("rename log file: %v", err)
	}

	// Compress if enabled
	if rw.compress {
		if err := rw.compressFile(rotatedFilename); err != nil {
			// Log error but don't fail rotation
			fmt.Fprintf(os.Stderr, "Failed to compress log file %s: %v\n", rotatedFilename, err)
		}
	}

	// Clean up old backups
	if err := rw.cleanupOldBackups(); err != nil {
		// Log error but don't fail rotation
		fmt.Fprintf(os.Stderr, "Failed to cleanup old backups: %v\n", err)
	}

	// Create new log file
	file, err := os.OpenFile(rw.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("create new log file: %v", err)
	}

	rw.file = file
	rw.size = 0
	rw.lastRotate = time.Now()

	return nil
}

// compressFile compresses a log file
func (rw *RotatingWriter) compressFile(filename string) error {
	// Open source file
	srcFile, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open source file: %v", err)
	}
	defer srcFile.Close()

	// Create compressed file
	compressedFilename := filename + ".gz"
	dstFile, err := os.Create(compressedFilename)
	if err != nil {
		return fmt.Errorf("create compressed file: %v", err)
	}
	defer dstFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()

	// Copy and compress
	if _, err := io.Copy(gzWriter, srcFile); err != nil {
		return fmt.Errorf("compress file: %v", err)
	}

	// Close gzip writer to flush
	if err := gzWriter.Close(); err != nil {
		return fmt.Errorf("close gzip writer: %v", err)
	}

	// Close destination file
	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("close compressed file: %v", err)
	}

	// Remove original file
	if err := os.Remove(filename); err != nil {
		return fmt.Errorf("remove original file: %v", err)
	}

	return nil
}

// cleanupOldBackups removes old backup files
func (rw *RotatingWriter) cleanupOldBackups() error {
	// Get directory and base filename
	dir := filepath.Dir(rw.filename)
	base := filepath.Base(rw.filename)

	// Read directory
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read log directory: %v", err)
	}

	// Find backup files
	var backupFiles []backupFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Check if it's a backup of our log file
		if strings.HasPrefix(name, base+".") {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			backupFiles = append(backupFiles, backupFile{
				name:    filepath.Join(dir, name),
				modTime: info.ModTime(),
			})
		}
	}

	// Sort by modification time (oldest first)
	sort.Slice(backupFiles, func(i, j int) bool {
		return backupFiles[i].modTime.Before(backupFiles[j].modTime)
	})

	// Remove excess backups
	if len(backupFiles) > rw.maxBackups {
		toRemove := backupFiles[:len(backupFiles)-rw.maxBackups]
		for _, backup := range toRemove {
			if err := os.Remove(backup.name); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to remove old backup %s: %v\n", backup.name, err)
			}
		}
	}

	return nil
}

// backupFile represents a backup file with its modification time
type backupFile struct {
	name    string
	modTime time.Time
}

// CreateRotatingWriterFromConfig creates a rotating writer from LogConfig
func CreateRotatingWriterFromConfig(config *LogConfig) (io.Writer, error) {
	if config.Rotation == nil {
		// No rotation configured, use regular output
		return parseOutput(config.Output)
	}

	// Parse max size
	var maxSize int64
	if config.Rotation.MaxSize != "" {
		size, err := parseSize(config.Rotation.MaxSize)
		if err != nil {
			return nil, fmt.Errorf("parse max size: %v", err)
		}
		maxSize = size
	}

	// Parse max age
	var maxAge time.Duration
	if config.Rotation.MaxAge != "" {
		age, err := parseDuration(config.Rotation.MaxAge)
		if err != nil {
			return nil, fmt.Errorf("parse max age: %v", err)
		}
		maxAge = age
	}

	// Extract filename from output
	filename := config.Output
	if strings.HasPrefix(filename, "file:") {
		filename = strings.TrimPrefix(filename, "file:")
	} else {
		// Default to current directory
		filename = "ytdlp.log"
	}

	return NewRotatingWriter(
		filename,
		maxSize,
		maxAge,
		config.Rotation.MaxBackups,
		config.Rotation.Compress,
	)
}

// CreateLoggerWithRotation creates a logger with rotation support
func CreateLoggerWithRotation(config *LogConfig) (*Logger, error) {
	// Validate config
	if err := config.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("validate config: %v", err)
	}

	// Create rotating writer if rotation is configured
	var output io.Writer
	var err error

	if config.Rotation != nil && config.Output != "stdout" && config.Output != "stderr" {
		output, err = CreateRotatingWriterFromConfig(config)
		if err != nil {
			return nil, fmt.Errorf("create rotating writer: %v", err)
		}
	} else {
		output, err = parseOutput(config.Output)
		if err != nil {
			return nil, fmt.Errorf("parse output: %v", err)
		}
	}

	// Create logger config
	loggerConfig, err := config.ToLoggerConfig()
	if err != nil {
		return nil, fmt.Errorf("convert config: %v", err)
	}

	// Override output
	loggerConfig.Output = output

	return New(loggerConfig), nil
}
