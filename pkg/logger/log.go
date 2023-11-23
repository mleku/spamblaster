package logger

import (
	"fmt"
	"os"
)

type logger struct {
	name string
}

// NewLogger creates a new logger with info and error level writers, interfaces
// with syslog, if available, or falls back to stderr if not.
func NewLogger(name string) (l *logger) {
	l = &logger{name: name}
	return
}

func (l *logger) Trace(format string, a ...interface{}) {
	format = fmt.Sprintf("[%s] trace: %s", l.name, format)
	_, _ = fmt.Fprintf(os.Stderr, format, a...)
}

func (l *logger) Info(format string, a ...interface{}) {
	format = fmt.Sprintf("[%s] info: %s", l.name, format)
	_, _ = fmt.Fprintf(os.Stderr, format, a...)
}

func (l *logger) Warn(format string, a ...interface{}) {
	format = fmt.Sprintf("[%s] warn: %s", l.name, format)
	_, _ = fmt.Fprintf(os.Stderr, format, a...)
}

func (l *logger) Err(format string, a ...interface{}) {
	format = fmt.Sprintf("[%s] error: %s", l.name, format)
	_, _ = fmt.Fprintf(os.Stderr, format, a...)
}

func (l *logger) Fatal(format string, a ...interface{}) {
	format = fmt.Sprintf("[%s] fatal: %s", l.name, format)
	_, _ = fmt.Fprintf(os.Stderr, format, a...)
}
