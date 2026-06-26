// Command demo is a tiny backend for the gusher.cluster example: it serves the
// single-page UI, signs a demo JWT (so the browser never holds the private key),
// and proxies publishes to the master (so the page stays same-origin / CORS-free).
// Pure stdlib — no dependencies.
package main

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
)

const (
	appKey  = "TEST"
	channel = "demo"
)

var privKey *rsa.PrivateKey

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	masterURL := env("MASTER_URL", "http://localhost:7777")
	staticDir := env("STATIC_DIR", "static")
	keyPath := env("PRIVATE_PEM", "/keys/private.pem")
	addr := env("LISTEN", ":8080")

	pk, err := loadPriv(keyPath)
	if err != nil {
		log.Fatalf("load private key: %v", err)
	}
	privKey = pk

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServer(http.Dir(staticDir)))

	// /token signs a demo JWT for app=TEST, channel=demo and tells the page where
	// the slave WebSocket lives (same host, port 8888).
	mux.HandleFunc("GET /token", func(w http.ResponseWriter, r *http.Request) {
		tok, err := signJWT()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"token": tok, "app": appKey, "channel": channel, "wsPort": "8888"})
	})

	// /publish proxies a message to the master so the browser stays same-origin.
	mux.HandleFunc("POST /publish", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		body, _ := json.Marshal(map[string]any{"event": "message", "data": in.Message})
		resp, err := http.Post(masterURL+"/v1/apps/"+appKey+"/channels/"+channel+"/messages",
			"application/json", bytes.NewReader(body))
		if err != nil {
			http.Error(w, "publish failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})

	log.Printf("gusher example demo listening on %s (master=%s)", addr, masterURL)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// signJWT builds an RS256 gusher JWT by hand (header.payload.signature) so the
// demo needs no JWT library.
func signJWT() (string, error) {
	header := b64(`{"alg":"RS256","typ":"JWT"}`)
	claims, _ := json.Marshal(map[string]any{
		"gusher": map[string]any{
			"app_key":  appKey,
			"user_id":  "demo-user",
			"channels": []string{channel},
		},
	})
	signingInput := header + "." + b64(string(claims))
	sum := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func b64(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }

func loadPriv(path string) (*rsa.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	blk, _ := pem.Decode(raw)
	if blk == nil {
		return nil, errors.New("no PEM block in key file")
	}
	if k, err := x509.ParsePKCS1PrivateKey(blk.Bytes); err == nil {
		return k, nil
	}
	if k, err := x509.ParsePKCS8PrivateKey(blk.Bytes); err == nil {
		if rk, ok := k.(*rsa.PrivateKey); ok {
			return rk, nil
		}
	}
	return nil, errors.New("unsupported private key (need RSA PKCS1/PKCS8)")
}
