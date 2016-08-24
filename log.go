package main

import (
	"net/http"

	"github.com/Sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
}

func (l *Logger) RequestWarn(r *http.Request, a interface{}) {
	l.WithFields(logrus.Fields{
		"Method":        r.Method,
		"RequestUri":    r.RequestURI,
		"RemoteAddr":    r.RemoteAddr,
		"ContentLength": r.ContentLength,
		"UserAgent":     r.UserAgent(),
	}).Warn(a)
}
func (l *Logger) RequestInfo(r *http.Request, a interface{}) {
	l.WithFields(logrus.Fields{
		"Method":        r.Method,
		"RequestUri":    r.RequestURI,
		"RemoteAddr":    r.RemoteAddr,
		"ContentLength": r.ContentLength,
		"UserAgent":     r.UserAgent(),
	}).Info(a)
}
func (l *Logger) RequestDebug(r *http.Request, a interface{}) {
	l.WithFields(logrus.Fields{
		"Method":        r.Method,
		"RequestUri":    r.RequestURI,
		"RemoteAddr":    r.RemoteAddr,
		"ContentLength": r.ContentLength,
		"UserAgent":     r.UserAgent(),
	}).Debug(a)
}
