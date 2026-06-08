package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelOff
)

func ParseLevel(s string) Level {
	switch s {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	case "off":
		return LevelOff
	default:
		return LevelInfo
	}
}

type Logger struct {
	level  Level
	writer io.Writer
	mu     sync.Mutex
	file   *os.File
}

var defaultLogger = &Logger{level: LevelOff}

func Init(level string, logFile string) {
	l := ParseLevel(level)
	if l == LevelOff && logFile == "" {
		return
	}

	logger := &Logger{level: l}

	if logFile != "" {
		os.MkdirAll(filepath.Dir(logFile), 0755)
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			logger.file = f
			logger.writer = f
		}
	}

	if logger.writer == nil {
		logger.writer = io.Discard
	}

	defaultLogger = logger
}

func Close() {
	if defaultLogger.file != nil {
		defaultLogger.file.Close()
	}
}

func write(level string, format string, args ...interface{}) {
	if defaultLogger.writer == nil {
		return
	}
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()

	ts := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(defaultLogger.writer, "%s [%s] %s\n", ts, level, msg)
}

func Debug(format string, args ...interface{}) {
	if defaultLogger.level <= LevelDebug {
		write("DEBUG", format, args...)
	}
}

func Info(format string, args ...interface{}) {
	if defaultLogger.level <= LevelInfo {
		write("INFO", format, args...)
	}
}

func Warn(format string, args ...interface{}) {
	if defaultLogger.level <= LevelWarn {
		write("WARN", format, args...)
	}
}

func Error(format string, args ...interface{}) {
	if defaultLogger.level <= LevelError {
		write("ERROR", format, args...)
	}
}
