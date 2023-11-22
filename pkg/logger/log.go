package logger

import (
	"fmt"
	"io"
	"log/syslog"
	"os"
)

type logger struct {
	infoLog, errLog           io.WriteCloser
	infoIsSyslog, errIsSyslog bool
}

// NewLogger creates a new logger with info and error level writers, interfaces
// with syslog, if available, or falls back to stderr if not.
func NewLogger() (l *logger) {
	var err error
	l = &logger{infoIsSyslog: true, errIsSyslog: true}
	l.infoLog, err = syslog.Dial("unix", "/dev/log",
		syslog.LOG_INFO, "spamblaster")
	if err != nil {
		l.infoLog = os.Stderr
		l.infoIsSyslog = false
		_, _ = fmt.Fprintf(l.infoLog,
			"unable to open syslog, maybe not running on a unix")
	}
	l.errLog, err = syslog.Dial("unix", "/dev/log",
		syslog.LOG_ERR, "spamblaster")
	if err != nil {
		l.errLog = os.Stderr
		l.errIsSyslog = false
		_, _ = fmt.Fprintf(l.errLog,
			"unable to open syslog, maybe not running on a unix")
	}
	return
}

func (l *logger) Close() (err error) {
	if l.infoIsSyslog {
		err = l.infoLog.Close()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr,
				"error closing info logger: %s\n", err)
		}
	}
	if l.errIsSyslog {
		err = l.errLog.Close()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr,
				"error closing error logger: %s\n", err)
		}
	}
	return
}

func (l *logger) Info(message string) {
	_, err := l.infoLog.Write([]byte(message))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr,
			"error writing log: %s\n", err)
	}
}

func (l *logger) Err(message string) {
	_, err := l.errLog.Write([]byte(message))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr,
			"error writing log: %s\n", err)
	}
}
