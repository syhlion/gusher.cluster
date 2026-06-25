package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"
)

// Fields is a structured-field map (replaces logrus.Fields).
type Fields map[string]any

// logLevel is the shared, mutable level for every handler we build. The `-d`
// debug flag flips it to Debug at startup (see envInit). Defaults to Info.
var logLevel = new(slog.LevelVar)

// Logger is a thin slog wrapper that keeps a small, chainable, logrus-like API
// so existing call sites work while logrus is removed. The output destination
// (stdout / file / both) and rotation are decided in logsetup.go.
type Logger struct{ l *slog.Logger }

// GetLogger returns a default stdout JSON logger — used before the env-driven
// setup runs (e.g. config-loading fatals).
func GetLogger() *Logger {
	return &Logger{l: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))}
}

func newLogger(l *slog.Logger) *Logger { return &Logger{l: l} }

func (l *Logger) Debug(args ...any)              { l.l.Debug(argsMsg(args)) }
func (l *Logger) Debugf(format string, a ...any) { l.l.Debug(fmt.Sprintf(format, a...)) }
func (l *Logger) Info(args ...any)               { l.l.Info(argsMsg(args)) }
func (l *Logger) Warn(args ...any)               { l.l.Warn(argsMsg(args)) }
func (l *Logger) Error(args ...any)              { l.l.Error(argsMsg(args)) }
func (l *Logger) Warnf(format string, a ...any)  { l.l.Warn(fmt.Sprintf(format, a...)) }
func (l *Logger) Fatal(args ...any)              { l.l.Error(argsMsg(args)); os.Exit(1) }

func (l *Logger) WithError(err error) *Entry       { return &Entry{l: l.l, attrs: []any{"error", err}} }
func (l *Logger) WithField(k string, v any) *Entry { return &Entry{l: l.l, attrs: []any{k, v}} }
func (l *Logger) WithFields(f Fields) *Entry       { return &Entry{l: l.l, attrs: fieldArgs(f)} }
func (l *Logger) GetRequestEntry(r *http.Request) *Entry {
	return &Entry{l: l.l, attrs: []any{
		"method", r.Method, "uri", r.RequestURI, "remote", r.RemoteAddr,
		"length", r.ContentLength, "ua", r.UserAgent(),
	}}
}

// Entry accumulates attributes for a single log line.
type Entry struct {
	l     *slog.Logger
	attrs []any
}

func (e *Entry) WithError(err error) *Entry       { e.attrs = append(e.attrs, "error", err); return e }
func (e *Entry) WithField(k string, v any) *Entry { e.attrs = append(e.attrs, k, v); return e }
func (e *Entry) Info(args ...any)                 { e.l.Info(argsMsg(args), e.attrs...) }
func (e *Entry) Warn(args ...any)                 { e.l.Warn(argsMsg(args), e.attrs...) }
func (e *Entry) Warnf(format string, a ...any)    { e.l.Warn(fmt.Sprintf(format, a...), e.attrs...) }
func (e *Entry) Error(args ...any)                { e.l.Error(argsMsg(args), e.attrs...) }

func argsMsg(args []any) string {
	if len(args) == 1 {
		if s, ok := args[0].(string); ok {
			return s
		}
	}
	return fmt.Sprint(args...)
}

func fieldArgs(f Fields) []any {
	a := make([]any, 0, len(f)*2)
	for k, v := range f {
		a = append(a, k, v)
	}
	return a
}

// RequestLogger is a stdlib middleware that logs each HTTP request via slog
// (replaces the negroni-based middleware). Wrap a handler: RequestLogger(l)(h).
func RequestLogger(l *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}
			l.Info("http",
				"method", r.Method, "path", r.URL.Path, "status", status,
				"remote", r.RemoteAddr, "dur", time.Since(start).String())
		})
	}
}

// statusRecorder captures the response status while delegating everything to
// the underlying ResponseWriter. It preserves http.Hijacker so the /ws
// websocket upgrade still works through the logging middleware.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

func (s *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := s.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("ResponseWriter does not support Hijack")
	}
	return h.Hijack()
}

func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
