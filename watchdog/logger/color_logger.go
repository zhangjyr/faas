package logger

import (
	"fmt"

	"github.com/mgutz/ansi"
)

// ColorLogger - A Logger that logs to stdout in color
type ColorLogger struct {
	Verbose     bool
	Level       int
	Prefix      string
	Color       bool
}

// Trace - Log a very verbose trace message
func (logger *ColorLogger) Trace(format string, args ...interface{}) {
	if !logger.Verbose {
		return
	}
	logger.log("lightgrey", format, args...)
}

// Debug - Log a debug message
func (logger *ColorLogger) Debug(format string, args ...interface{}) {
	if logger.Level > LOG_LEVEL_ALL  {
		return
	}
	logger.log("", format, args...)
}

// Info - Log a general message
func (logger *ColorLogger) Info(format string, args ...interface{}) {
	if logger.Level > LOG_LEVEL_INFO  {
		return
	}
	logger.log("green", format, args...)
}

// Warn - Log a warning
func (logger *ColorLogger) Warn(format string, args ...interface{}) {
	if logger.Level > LOG_LEVEL_WARN  {
		return
	}
	logger.log("yellow", format, args...)
}

// Error - Log a error
func (logger *ColorLogger) Error(format string, args ...interface{}) {
	if logger.Level > LOG_LEVEL_NONE  {
		return
	}
	logger.log("red", format, args...)
}


func (logger *ColorLogger) log(color, format string, args ...interface{}) {
	if logger.Color && color != "" {
		format = ansi.Color(format, color)
	}
	fmt.Printf(fmt.Sprintf("%s%s\n", logger.Prefix, format), args...)
}