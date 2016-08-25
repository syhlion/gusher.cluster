package main

import (
	"net/http"

	"github.com/Sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
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
