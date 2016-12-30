package httplog

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/urfave/negroni"
)

// Logger is a middleware handler that logs the request as it goes in and the response as it goes out.
type Logger struct {
	// Logger inherits from log.Logger used to log messages with the Logger middleware
	*log.Logger
}

// NewLogger returns a new Logger instance
func NewLogger() *Logger {
	return &Logger{log.New(os.Stdout, "[http log] ", 0)}
}

func (l *Logger) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()

	next(rw, r)

	res := rw.(negroni.ResponseWriter)
	r.ParseForm()
	l.Printf("%s - [%v] \"%s %s\" %d %v %s \"%s\"", r.RemoteAddr, time.Now(), r.Method, r.URL.Path, res.Status(), time.Since(start), r.UserAgent(), r.Form.Encode())
}
