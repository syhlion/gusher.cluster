package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	redisocket "github.com/syhlion/redisocket.v2"
)

// startEmbeddedNATS spins up an in-process NATS server on a random port.
func startEmbeddedNATS(t *testing.T) *natsserver.Server {
	t.Helper()
	ns, err := natsserver.NewServer(&natsserver.Options{Host: "127.0.0.1", Port: -1})
	if err != nil {
		t.Fatalf("new nats server: %v", err)
	}
	go ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		t.Fatal("nats server not ready")
	}
	return ns
}

// buildE2EJWT signs a gusher JWT (RS256) with the test private key.
func buildE2EJWT(t *testing.T, userID, appKey string, channels []string) string {
	t.Helper()
	priv, err := makePrivateKey()
	if err != nil {
		t.Fatalf("private key: %v", err)
	}
	claims := jwt.MapClaims{
		"gusher": map[string]any{
			"user_id":  userID,
			"channels": channels,
			"app_key":  appKey,
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(priv)
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	return tok
}

// TestE2E_AuthSubscribePush drives the full path with no Redis and no external
// process: embedded NATS, a real slave Hub (NATS broker + presence) behind the
// /auth and /ws HTTP handlers, and a master Sender behind /push. A real
// websocket client authenticates, subscribes, and must receive a message the
// master publishes over NATS.
func TestE2E_AuthSubscribePush(t *testing.T) {
	engineLog := slog.New(slog.NewTextHandler(io.Discard, nil))

	pub, err := makePublicKey()
	if err != nil {
		t.Fatalf("public key: %v", err)
	}

	ns := startEmbeddedNATS(t)
	defer ns.Shutdown()
	natsURL := ns.ClientURL()

	// --- slave side: Hub on NATS broker + presence ---
	ncSlave, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatalf("slave nats connect: %v", err)
	}
	defer ncSlave.Close()
	brokerS := redisocket.NewNATSBroker(ncSlave)
	presenceS, err := redisocket.NewMemoryPresence(ncSlave, listenChannelPrefix)
	if err != nil {
		t.Fatalf("slave presence: %v", err)
	}
	hub := redisocket.NewHubWithBrokerAndPresence(brokerS, presenceS, engineLog, false)
	hubErr := make(chan error, 1)
	go func() { hubErr <- hub.Listen(listenChannelPrefix) }()
	defer hub.Close()

	// --- master side: Sender on its own NATS connection ---
	ncMaster, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatalf("master nats connect: %v", err)
	}
	defer ncMaster.Close()
	brokerM := redisocket.NewNATSBroker(ncMaster)
	presenceM, err := redisocket.NewMemoryPresence(ncMaster, listenChannelPrefix)
	if err != nil {
		t.Fatalf("master presence: %v", err)
	}
	sender := redisocket.NewSenderWithBrokerAndPresence(brokerM, presenceM)

	// --- HTTP servers wired with the real handlers ---
	sc := SlaveConfig{}
	slaveRouter := mux.NewRouter()
	slaveRouter.HandleFunc("/auth", WsAuth(sc, pub)).Methods("POST")
	slaveRouter.HandleFunc("/ws/{app_key}", WsConnect(sc, pub, hub)).Methods("GET")
	slaveSrv := httptest.NewServer(slaveRouter)
	defer slaveSrv.Close()

	masterRouter := mux.NewRouter()
	masterRouter.HandleFunc("/push/{app_key}/{channel}/{event}", PushMessage(sender)).Methods("POST")
	masterSrv := httptest.NewServer(masterRouter)
	defer masterSrv.Close()

	const appKey, channel, event = "TEST", "AA", "myevent"
	token := buildE2EJWT(t, "user-1", appKey, []string{channel})

	// 1) /auth — local JWT verify, echoes the token back
	authResp, err := http.PostForm(slaveSrv.URL+"/auth", url.Values{"jwt": {token}})
	if err != nil {
		t.Fatalf("auth post: %v", err)
	}
	if authResp.StatusCode != http.StatusOK {
		t.Fatalf("auth status = %d, want 200", authResp.StatusCode)
	}
	authResp.Body.Close()

	// 2) open the websocket
	wsURL := "ws" + strings.TrimPrefix(slaveSrv.URL, "http") + "/ws/" + appKey + "?token=" + token
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer c.Close()

	// 3) subscribe to channel AA
	c.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := c.WriteMessage(websocket.TextMessage, []byte(`{"event":"gusher.subscribe","data":{"channel":"AA"}}`)); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}
	c.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, reply, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read subscribe reply: %v", err)
	}
	if !strings.Contains(string(reply), "subscribe_succeeded") {
		t.Fatalf("subscribe reply = %s, want subscribe_succeeded", reply)
	}

	// 4) master pushes to channel AA; client must receive it
	pushResp, err := http.PostForm(
		masterSrv.URL+"/push/"+appKey+"/"+channel+"/"+event,
		url.Values{"data": {"hello-e2e"}},
	)
	if err != nil {
		t.Fatalf("push post: %v", err)
	}
	if pushResp.StatusCode != http.StatusOK {
		t.Fatalf("push status = %d, want 200", pushResp.StatusCode)
	}
	pushResp.Body.Close()

	c.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, got, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read pushed message: %v", err)
	}
	if !strings.Contains(string(got), event) || !strings.Contains(string(got), "hello-e2e") {
		t.Fatalf("pushed message = %s, want event %q + payload hello-e2e", got, event)
	}
}

// TestE2E_RejectBadToken ensures /ws refuses a token not signed by our key.
func TestE2E_RejectBadToken(t *testing.T) {

	pub, err := makePublicKey()
	if err != nil {
		t.Fatalf("public key: %v", err)
	}

	sc := SlaveConfig{}
	router := mux.NewRouter()
	router.HandleFunc("/auth", WsAuth(sc, pub)).Methods("POST")
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.PostForm(srv.URL+"/auth", url.Values{"jwt": {"not-a-jwt"}})
	if err != nil {
		t.Fatalf("auth post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("auth status = %d, want 401 for bad token", resp.StatusCode)
	}
}
