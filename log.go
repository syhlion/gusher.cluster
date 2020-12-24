package main

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

func GetLogger() *Logger {
	l := logrus.New()
	/*
		e := l.WithFields(logrus.Fields{
			"Version":        version,
			"RuntimeVersion": runtime.Version(),
		})
	*/

	return &Logger{
		l,
	}

}

type Logger struct {
	*logrus.Logger
}

func (l *Logger) GetLogger() *logrus.Logger {
	return l.Logger
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
