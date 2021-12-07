package httplog

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"
)

// Logger is a middleware handler that logs the request as it goes in and the response as it goes out.
type Logger struct {
	// Logger inherits from log.Logger used to log messages with the Logger middleware
	*log.Logger
	jsonOut bool
}

// NewLogger returns a new Logger instance
func NewLogger(jsonOut bool) *Logger {
	return &Logger{log.New(os.Stdout, "[http log] ", 0), jsonOut}
}

// ServerHTTP negroni middleware interface
func (l *Logger) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()

	err := r.ParseForm()
	if err != nil {
		l.Println("ParseForm error", err)
	}

	param := r.Form.Encode()
	next(rw, r)

	res := rw.(negroni.ResponseWriter)

	if l.jsonOut {
		logrus.SetFormatter(&logrus.JSONFormatter{})
		logrus.WithFields(map[string]interface{}{
			"Addr":      r.RemoteAddr,
			"Now":       time.Now(),
			"Method":    r.Method,
			"URLPath":   r.URL.Path,
			"ResStatus": res.Status(),
			"StartAt":   time.Since(start),
			"Agent":     r.UserAgent(),
			"Param":     param,
		}).Info("[http log]")
	} else {
		l.Printf("%s - [%v] \"%s %s\" %d %v %s \"%s\"", r.RemoteAddr, time.Now(), r.Method, r.URL.Path, res.Status(), time.Since(start), r.UserAgent(), param)
	}
}
