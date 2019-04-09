package Logger

import (
	"fmt"
	"log"
	"os"
)

/* Prefix for all default log messages. */
const defaultLoggerPrefix = "[Injector Log] [%s] - "

/*
A generic Logger that can be hooked into by the end user if required.

Example usage: Logger.SetDebugLogger(log.Println)
*/
type Logger struct {
	/* Logging debug messages with this - can be user provided. */
	debugLog func(...interface{})
	/* Logging error messages with this - can be user provided. */
	errorLog func(...interface{})
}

/* Create a new Logger with user-provided logging functions. */
func New(debugLog func(...interface{}), errorLog func(...interface{})) Logger {
	return Logger{debugLog: debugLog, errorLog: errorLog}
}

/* Create a new Logger which defaults logging to stdout / stderr. */
func NewStdLogger() *Logger {
	return &Logger{debugLog: getDefaultDebugLogger(), errorLog: getDefaultErrorLogger()}
}

/* Write a debug log. */
func (l *Logger) Debug(msg string) {
	l.debugLog(msg)
}

/* Write an error log. */
func (l *Logger) Error(msg string) {
	l.errorLog(msg)
}

/* Default to stdout. */
func getDefaultDebugLogger() func(...interface{}) {
	return log.New(os.Stdout, fmt.Sprintf(defaultLoggerPrefix, "DEBUG"), 0).Println
}

/* Default to stderr. */
func getDefaultErrorLogger() func(...interface{}) {
	return log.New(os.Stdout, fmt.Sprintf(defaultLoggerPrefix, "ERROR"), 0).Println
}
