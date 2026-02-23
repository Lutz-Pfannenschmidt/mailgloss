package logger

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
)

var Log *log.Logger

// Init initializes the logger
func Init() error {
	// Get log file path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	logDir := filepath.Join(homeDir, ".config", "mailgloss")
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return err
	}

	logPath := filepath.Join(logDir, "mailgloss.log")

	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}

	// Create logger
	Log = log.NewWithOptions(logFile, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
		TimeFormat:      "2006-01-02 15:04:05",
		Prefix:          "mailgloss",
	})

	Log.SetLevel(log.InfoLevel)

	return nil
}

// Debug logs a debug message
func Debug(msg string, keyvals ...interface{}) {
	if Log != nil {
		Log.Debug(msg, keyvals...)
	}
}

// Info logs an info message
func Info(msg string, keyvals ...interface{}) {
	if Log != nil {
		Log.Info(msg, keyvals...)
	}
}

// Warn logs a warning message
func Warn(msg string, keyvals ...interface{}) {
	if Log != nil {
		Log.Warn(msg, keyvals...)
	}
}

// Error logs an error message
func Error(msg string, keyvals ...interface{}) {
	if Log != nil {
		Log.Error(msg, keyvals...)
	}
}

// Fatal logs a fatal message and exits
func Fatal(msg string, keyvals ...interface{}) {
	if Log != nil {
		Log.Fatal(msg, keyvals...)
	}
}
