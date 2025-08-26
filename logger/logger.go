package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// Level represents the severity level of log messages
type Level int

const (
	// DEBUG level for detailed debugging information
	DEBUG Level = iota
	// INFO level for general information
	INFO
	// WARN level for warnings that don't stop execution
	WARN
	// ERROR level for errors that should be addressed
	ERROR
	// FATAL level for critical errors that cause program termination
	FATAL
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging functionality
type Logger struct {
	level      Level
	useColors  bool
	showCaller bool
}

// Config holds logger configuration
type Config struct {
	Level      Level
	UseColors  bool
	ShowCaller bool
}

// New creates a new logger with the given configuration
func New(config Config) *Logger {
	return &Logger{
		level:      config.Level,
		useColors:  config.UseColors,
		showCaller: config.ShowCaller,
	}
}

// NewDefault creates a logger with default configuration
func NewDefault() *Logger {
	level := INFO
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		switch strings.ToUpper(levelStr) {
		case "DEBUG":
			level = DEBUG
		case "INFO":
			level = INFO
		case "WARN":
			level = WARN
		case "ERROR":
			level = ERROR
		case "FATAL":
			level = FATAL
		}
	}

	return &Logger{
		level:      level,
		useColors:  os.Getenv("LOG_COLORS") != "false",
		showCaller: os.Getenv("LOG_CALLER") == "true",
	}
}

// log writes a log message with the specified level
func (l *Logger) log(level Level, message string, args ...interface{}) {
	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	var caller string
	if l.showCaller {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			parts := strings.Split(file, "/")
			if len(parts) > 2 {
				file = strings.Join(parts[len(parts)-2:], "/")
			}
			caller = fmt.Sprintf("%s:%d", file, line)
		}
	}

	var output string
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}

	if l.useColors {
		color := l.getColor(level)
		if caller != "" {
			output = fmt.Sprintf("%s %s[%s]%s %s %s", timestamp, color, level.String(), "\033[0m", caller, message)
		} else {
			output = fmt.Sprintf("%s %s[%s]%s %s", timestamp, color, level.String(), "\033[0m", message)
		}
	} else {
		if caller != "" {
			output = fmt.Sprintf("%s [%s] %s %s", timestamp, level.String(), caller, message)
		} else {
			output = fmt.Sprintf("%s [%s] %s", timestamp, level.String(), message)
		}
	}

	if level == FATAL {
		log.Fatal(output)
	} else {
		log.Println(output)
	}
}

// getColor returns ANSI color code for the given log level
func (l *Logger) getColor(level Level) string {
	switch level {
	case DEBUG:
		return "\033[36m" // Cyan
	case INFO:
		return "\033[32m" // Green
	case WARN:
		return "\033[33m" // Yellow
	case ERROR:
		return "\033[31m" // Red
	case FATAL:
		return "\033[35m" // Magenta
	default:
		return "\033[0m" // Reset
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string, args ...interface{}) {
	l.log(DEBUG, message, args...)
}

// Info logs an info message
func (l *Logger) Info(message string, args ...interface{}) {
	l.log(INFO, message, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, args ...interface{}) {
	l.log(WARN, message, args...)
}

// Error logs an error message
func (l *Logger) Error(message string, args ...interface{}) {
	l.log(ERROR, message, args...)
}

// Fatal logs a fatal message and exits the program
func (l *Logger) Fatal(message string, args ...interface{}) {
	l.log(FATAL, message, args...)
}

// Global logger instance
var defaultLogger = NewDefault()

// Debug logs a debug message using the default logger
func Debug(message string, args ...interface{}) {
	defaultLogger.log(DEBUG, message, args...)
}

// Info logs an info message using the default logger
func Info(message string, args ...interface{}) {
	defaultLogger.log(INFO, message, args...)
}

// Warn logs a warning message using the default logger
func Warn(message string, args ...interface{}) {
	defaultLogger.log(WARN, message, args...)
}

// Error logs an error message using the default logger
func Error(message string, args ...interface{}) {
	defaultLogger.log(ERROR, message, args...)
}

// Fatal logs a fatal message and exits the program using the default logger
func Fatal(message string, args ...interface{}) {
	defaultLogger.log(FATAL, message, args...)
}
