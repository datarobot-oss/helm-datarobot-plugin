// Copyright 2025 DataRobot, Inc. and its affiliates.
//
// All rights reserved.
//
// DataRobot, Inc. Confidential.
//
// This is unpublished proprietary source code of DataRobot, Inc.
// and its affiliates.
//
// The copyright notice above does not evidence any actual or intended
// publication of such source code.
package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// LogLevel represents the level of logging.
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	FATAL
)

// Logger represents a logger instance.
type Logger struct {
	level     LogLevel
	output    io.Writer
	prefix    string
	showLevel bool
}

var (
	instance *Logger
	once     sync.Once
)

// init initializes the default logger instance.
func init() {
	once.Do(func() {
		instance = &Logger{
			level:     INFO,      // Default log level
			output:    os.Stdout, // Default output
			showLevel: true,
		}
	})
}

// SetLevel sets the log level.
func SetLevel(level LogLevel) {
	instance.level = level
}

func SetPrefix(prefix string) {
	instance.prefix = prefix
}

func HideLevel() {
	instance.showLevel = false
}

// SetOutput sets the output destination.
func SetOutput(output io.Writer) {
	instance.output = output
}

// log writes the log message to the output.
func log(level LogLevel, format string, args ...interface{}) {
	if level < instance.level {
		return
	}

	levelStr := ""
	switch level {
	case INFO:
		levelStr = "INFO"
	case WARNING:
		levelStr = "WARN"
	case ERROR:
		levelStr = "ERROR"
	case FATAL:
		levelStr = "FATAL"
	case DEBUG:
		levelStr = "DEBU"
	}

	message := fmt.Sprintf(format, args...)

	logMessage := fmt.Sprintf("[%s]%s %s\n", levelStr, instance.prefix, message)
	if !instance.showLevel {
		logMessage = fmt.Sprintf("%s %s\n", instance.prefix, message)
	}

	if instance.output != nil {
		_, _ = fmt.Fprint(instance.output, logMessage)
	}
}

// Info logs an info message.
func Info(format string, args ...interface{}) {
	log(INFO, format, args...)
}

// Warning logs a warning message.
func Warning(format string, args ...interface{}) {
	log(WARNING, format, args...)
}

// Error logs an error message.
func Error(format string, args ...interface{}) {
	log(ERROR, format, args...)
}

// Debug logs a debug message.
func Debug(format string, args ...interface{}) {
	log(DEBUG, format, args...)
}

func Fatal(format string, args ...interface{}) {
	log(FATAL, format, args...)
	os.Exit(1)
}
