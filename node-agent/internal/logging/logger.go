package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	globalLogger *slog.Logger
	loggerOnce   sync.Once
)

type RotatingFileWriter struct {
	mu         sync.Mutex
	filename   string
	maxSize    int64 // bytes before rotation (e.g., 10MB)
	curSize    int64
	file       *os.File
}

func NewRotatingFileWriter(filename string, maxSizeMB int64) (*RotatingFileWriter, error) {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	w := &RotatingFileWriter{
		filename: filename,
		maxSize:  maxSizeMB * 1024 * 1024,
	}

	if err := w.openFile(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *RotatingFileWriter) openFile() error {
	info, err := os.Stat(w.filename)
	if err == nil {
		w.curSize = info.Size()
	} else {
		w.curSize = 0
	}

	f, err := os.OpenFile(w.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	w.file = f
	return nil
}

func (w *RotatingFileWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	writeLen := int64(len(p))
	if w.curSize+writeLen > w.maxSize && w.maxSize > 0 {
		w.file.Close()
		backupName := fmt.Sprintf("%s.%s", w.filename, time.Now().Format("20060102-150405"))
		_ = os.Rename(w.filename, backupName)
		_ = w.openFile()
	}

	n, err = w.file.Write(p)
	w.curSize += int64(n)
	return n, err
}

func (w *RotatingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// InitLogger initializes the global structured JSON logger.
func InitLogger(logDir string, levelStr string, toStdout bool) (*slog.Logger, error) {
	var level slog.Level
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var writers []io.Writer
	if toStdout {
		writers = append(writers, os.Stdout)
	}

	if logDir != "" {
		logFile := filepath.Join(logDir, "node-agent.log")
		rotator, err := NewRotatingFileWriter(logFile, 10) // 10MB rotation
		if err == nil {
			writers = append(writers, rotator)
		}
	}

	var mw io.Writer
	if len(writers) == 1 {
		mw = writers[0]
	} else if len(writers) > 1 {
		mw = io.MultiWriter(writers...)
	} else {
		mw = os.Stdout
	}

	handlerOpts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Redact sensitive keys if present
			key := strings.ToLower(a.Key)
			if key == "token" || key == "enrollment_token" || key == "secret" || key == "credentials" {
				return slog.String(a.Key, "[REDACTED]")
			}
			return a
		},
	}

	logger := slog.New(slog.NewJSONHandler(mw, handlerOpts))
	slog.SetDefault(logger)
	globalLogger = logger

	return logger, nil
}

// GetLogger returns the configured logger instance.
func GetLogger() *slog.Logger {
	if globalLogger == nil {
		loggerOnce.Do(func() {
			globalLogger = slog.Default()
		})
	}
	return globalLogger
}
