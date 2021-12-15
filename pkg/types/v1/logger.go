package v1

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
)

// Logger is the interface we want for our logger, so we can plug different ones easily
type Logger interface {
	Info(...interface{})
	Warn(...interface{})
	Debug(...interface{})
	Error(...interface{})
	Fatal(...interface{})
	Panic(...interface{})
	Trace(...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Debugf(string, ...interface{})
	Errorf(string, ...interface{})
	Fatalf(string, ...interface{})
	Panicf(string, ...interface{})
	Tracef(string, ...interface{})
}

type LoggerOptions func(l Logger) error

func NewLogger() Logger {
	return log.New()
}

// NewNullLogger will return a logger that discards all logs, used mainly for testing
func NewNullLogger() Logger {
	logger := log.New()
	logger.SetOutput(ioutil.Discard)
	return logger
}