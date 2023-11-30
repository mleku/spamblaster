package logger

import (
	"fmt"
	"os"
)

type T struct {
	name string
}

// NewLogger creates a new T with info and error level writers, interfaces
// with syslog, if available, or falls back to stderr if not.
func NewLogger(name string) (l *T) {
	l = &T{name: name}
	return
}

func (l *T) Trace(format string, a ...interface{}) {
	format = fmt.Sprintf("[%s] trace: %s", l.name, format)
	_, _ = fmt.Fprintf(os.Stderr, format, a...)
}

func (l *T) Info(format string, a ...interface{}) {
	format = fmt.Sprintf("[%s] info: %s", l.name, format)
	_, _ = fmt.Fprintf(os.Stderr, format, a...)
}

func (l *T) Warn(format string, a ...interface{}) {
	format = fmt.Sprintf("[%s] warn: %s", l.name, format)
	_, _ = fmt.Fprintf(os.Stderr, format, a...)
}

func (l *T) Err(format string, a ...interface{}) {
	format = fmt.Sprintf("[%s] error: %s", l.name, format)
	_, _ = fmt.Fprintf(os.Stderr, format, a...)
}

func (l *T) Fatal(format string, a ...interface{}) {
	format = fmt.Sprintf("[%s] fatal: %s", l.name, format)
	_, _ = fmt.Fprintf(os.Stderr, format, a...)
}
