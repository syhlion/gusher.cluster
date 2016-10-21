package main

import (
	"net/http"
	"runtime"

	"github.com/Sirupsen/logrus"
)

func GetLogger() *Logger {
	l := logrus.New()
	e := l.WithFields(logrus.Fields{
		"Version":        version,
		"RuntimeVersion": runtime.Version(),
	})
	return &Logger{
		e,
	}

}

type Logger struct {
	*logrus.Entry
}

func (l *Logger) GetRequestEntry(r *http.Request) *logrus.Entry {
	return l.WithFields(logrus.Fields{
		"Method":        r.Method,
		"RequestUri":    r.RequestURI,
		"RemoteAddr":    r.RemoteAddr,
		"ContentLength": r.ContentLength,
		"UserAgent":     r.UserAgent(),
	})
}
