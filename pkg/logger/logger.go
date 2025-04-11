package logger

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger is a wrapper around logrus.Logger
type Logger struct {
	*logrus.Logger
}

// New creates a new logger
func New() *Logger {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
	})
	log.SetLevel(logrus.InfoLevel)

	return &Logger{Logger: log}
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level string) {
	switch level {
	case "debug":
		l.Logger.SetLevel(logrus.DebugLevel)
	case "info":
		l.Logger.SetLevel(logrus.InfoLevel)
	case "warn":
		l.Logger.SetLevel(logrus.WarnLevel)
	case "error":
		l.Logger.SetLevel(logrus.ErrorLevel)
	default:
		l.Logger.SetLevel(logrus.InfoLevel)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(args ...interface{}) {
	l.Logger.Debug(args...)
}

// Info logs an info message
func (l *Logger) Info(args ...interface{}) {
	l.Logger.Info(args...)
}

// Warn logs a warning message
func (l *Logger) Warn(args ...interface{}) {
	l.Logger.Warn(args...)
}

// Error logs an error message
func (l *Logger) Error(args ...interface{}) {
	l.Logger.Error(args...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(args ...interface{}) {
	l.Logger.Fatal(args...)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debugf(format, args...)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Infof(format, args...)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logger.Warnf(format, args...)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Errorf(format, args...)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Fatalf(format, args...)
}
