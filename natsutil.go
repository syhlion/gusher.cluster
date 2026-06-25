package main

import (
	"net/http"
	"os"
	"time"

	nats "github.com/nats-io/nats.go"
)

// connectNATS dials NATS with production-grade options: reconnect forever with
// backoff, a buffer to absorb publishes while disconnected, and lifecycle logs.
// Core subscriptions (bus + presence responder) are automatically re-established
// by nats.go on reconnect, so no extra resubscribe code is needed.
//
//	GUSHER_NATS_CREDS  path to a .creds file → enable user-credential auth
//	TLS                use a tls:// addr (or NATS server config) for TLS
func connectNATS(addr, name string) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.Name(name),
		nats.MaxReconnects(-1), // reconnect forever
		nats.ReconnectWait(2 * time.Second),
		nats.ReconnectBufSize(8 * 1024 * 1024),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			logger.WithError(err).Warn("nats disconnected")
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Warnf("nats reconnected to %s", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			logger.Error("nats connection closed")
		}),
	}
	if creds := os.Getenv("GUSHER_NATS_CREDS"); creds != "" {
		opts = append(opts, nats.UserCredentials(creds))
	}
	return nats.Connect(addr, opts...)
}

// Ready returns 200 only while NATS is connected — use it as a k8s readiness
// probe (vs /ping which is a plain liveness check).
func Ready(nc *nats.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if nc.IsConnected() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ready"))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("nats not connected"))
	}
}
