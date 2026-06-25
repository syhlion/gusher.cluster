package main

import (
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	_ "net/http/pprof"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	redisocket "github.com/syhlion/redisocket.v2"
	"github.com/urfave/cli"
	"github.com/urfave/negroni"
)

// master server
func master(c *cli.Context) {

	mc := getMasterConfig(c)
	/*logging: 輸出 stdout/file/both ＋ 輪替(env 驅動)*/
	ls, logErr := setupLoggingFromEnv()
	if logErr != nil {
		logger.Fatal(logErr)
	}
	logger = ls.App
	defer ls.Close()

	b, err := ioutil.ReadFile(mc.PublicKeyLocation)
	if err != nil {
		logger.Warn(err)
	}
	public_pem, rsaKeyErr := jwt.ParseRSAPublicKeyFromPEM(b)
	if rsaKeyErr != nil {
		logger.Warnf("Did not start \"%sdecode\" api", mc.ApiPrefix)
	}

	/*NATS:publish + presence 聚合(master 無連線,presence 查詢經 request/reply 匯總各 slave)*/
	nc, err := connectNATS(mc.NatsAddr, "gusher-master")
	if err != nil {
		logger.Fatal("nats connect error: ", err)
	}
	defer nc.Close()
	broker := redisocket.NewNATSBroker(nc)
	presence, err := redisocket.NewMemoryPresence(nc, listenChannelPrefix)
	if err != nil {
		logger.Fatal("nats presence error: ", err)
	}
	rsender := redisocket.NewSenderWithBrokerAndPresence(broker, presence)

	/*api start*/
	apiListener, err := net.Listen("tcp", mc.ApiListen)
	if err != nil {
		logger.Fatal(err)
	}
	r := mux.NewRouter()

	sub := r.PathPrefix(mc.ApiPrefix).Subrouter()
	sub.HandleFunc("/push/socket/{app_key}/{socket_id}", PushToSocket(rsender)).Methods("POST")
	sub.HandleFunc("/push/user/{app_key}/{user_id}", PushToUser(rsender)).Methods("POST")
	sub.HandleFunc("/push/{app_key}/{channel}/{event}", PushMessage(rsender)).Methods("POST")
	sub.HandleFunc("/push_batch/{app_key}", PushBatchMessage(rsender)).Methods("POST")
	sub.HandleFunc("/push/{app_key}", PushMessageByPattern(rsender)).Methods("POST")
	sub.HandleFunc("/reload/channel/user/{app_key}/{user_id}", ReloadUserChannels(rsender)).Methods("POST")
	sub.HandleFunc("/add/channel/user/{app_key}/{user_id}", AddUserChannels(rsender)).Methods("POST")
	sub.HandleFunc("/{app_key}/channels", GetAllChannel(rsender)).Methods("GET")
	sub.HandleFunc("/{app_key}/channels/count", GetAllChannelCount(rsender)).Methods("GET")
	sub.HandleFunc("/{app_key}/online/bychannel/{channel}", GetOnlineByChannel(rsender)).Methods("GET")
	sub.HandleFunc("/{app_key}/online/bychannel/{channel}/count", GetOnlineCountByChannel(rsender)).Methods("GET")
	sub.HandleFunc("/{app_key}/online", GetOnline(rsender)).Methods("GET")
	sub.HandleFunc("/{app_key}/online/count", GetOnlineCount(rsender)).Methods("GET")
	sub.HandleFunc("/ping", Ping()).Methods("GET")
	sub.HandleFunc("/ready", Ready(nc)).Methods("GET")
	if rsaKeyErr == nil {
		sub.HandleFunc("/decode", DecodeJWT(public_pem)).Methods("POST")
	}
	n := negroni.New()
	n.Use(RequestLogger(ls.Slog))
	n.UseHandler(r)
	serverError := make(chan error, 1)
	server := http.Server{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		Handler:      n,
	}
	go func() {
		err := server.Serve(apiListener)
		serverError <- err
	}()
	go func() {
		logger.Error(http.ListenAndServe("127.0.0.1:7799", nil))
	}()

	// block and listen syscall
	shutdow_observer := make(chan os.Signal, 1)
	t := template.Must(template.New("gusher master start msg").Parse(masterMsgFormat))
	t.Execute(os.Stdout, mc)
	signal.Notify(shutdow_observer, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-shutdow_observer:
		logger.Info("Receive signal")
	case err := <-serverError:
		logger.Warn(err)
	}

}
func runtimeStats() (m *runtime.MemStats) {
	m = &runtime.MemStats{}

	//log.Println("# goroutines: ", runtime.NumGoroutine())
	runtime.ReadMemStats(m)
	//log.Println("Memory Acquired: ", m.Sys)
	//log.Println("Memory Used    : ", m.Alloc)
	return m
}

// slave server
func slave(c *cli.Context) {

	sc := getSlaveConfig(c)
	/*logging: 輸出 stdout/file/both ＋ 輪替(env 驅動);引擎與 app 皆 slog、同一目的地*/
	ls, logErr := setupLoggingFromEnv()
	if logErr != nil {
		logger.Fatal(logErr)
	}
	logger = ls.App
	defer ls.Close()
	/*本機 JWT 驗證:載入公鑰(取代 decode service + greq)*/
	pemBytes, err := ioutil.ReadFile(sc.PublicKeyLocation)
	if err != nil {
		logger.Fatal(err)
	}
	publicPem, err := jwt.ParseRSAPublicKeyFromPEM(pemBytes)
	if err != nil {
		logger.Fatal("parse public key error: ", err)
	}

	/*NATS:bus + presence(取代 redis pub/sub + sorted-set presence)*/
	nc, err := connectNATS(sc.NatsAddr, "gusher-slave")
	if err != nil {
		logger.Fatal("nats connect error: ", err)
	}
	broker := redisocket.NewNATSBroker(nc)
	presence, err := redisocket.NewMemoryPresence(nc, listenChannelPrefix)
	if err != nil {
		logger.Fatal("nats presence error: ", err)
	}
	rsHub := redisocket.NewHubWithBrokerAndPresence(broker, presence, ls.Slog, c.Bool("debug"))
	rsHub.Config.MaxMessageSize = int64(sc.MaxMessage)
	rsHub.Config.ScanInterval = sc.ScanInterval
	rsHub.Config.Upgrader.ReadBufferSize = sc.ReadBuffer
	rsHub.Config.Upgrader.WriteBufferSize = sc.WriteBuffer
	rsHubErr := make(chan error, 1)
	go func() {
		rsHubErr <- rsHub.Listen(listenChannelPrefix)
	}()
	/*api start*/
	apiListener, err := net.Listen("tcp", sc.ApiListen)
	if err != nil {
		logger.Fatal(err)
	}
	r := mux.NewRouter()

	/*api end*/

	//server := http.NewServeMux()

	sub := r.PathPrefix(sc.ApiPrefix).Subrouter()
	sub.HandleFunc("/ws/{app_key}", WsConnect(sc, publicPem, rsHub)).Methods("GET")
	sub.HandleFunc("/wtf/{app_key}", WsConnect(sc, publicPem, rsHub)).Methods("GET")
	sub.HandleFunc("/auth", WsAuth(sc, publicPem)).Methods("POST")
	sub.HandleFunc("/ping", Ping()).Methods("GET")
	sub.HandleFunc("/ready", Ready(nc)).Methods("GET")
	n := negroni.New()
	n.Use(RequestLogger(ls.Slog))
	n.UseHandler(r)
	serverError := make(chan error, 1)
	server := http.Server{
		ReadTimeout: 3 * time.Second,
		Handler:     n,
	}
	go func() {
		err := server.Serve(apiListener)
		serverError <- err
	}()
	go func() {
		logger.Error(http.ListenAndServe("127.0.0.1:8899", nil))
	}()

	closeConnTotal := make(chan int, 0)
	//固定30秒log出 現在連線人數
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer func() {
			t.Stop()
		}()

		for {
			select {
			case <-t.C:
				m := runtimeStats()
				logger.WithFields(Fields{
					"memory-acquired": m.Sys,
					"memory-used":     m.Alloc,
					"goroutines":      runtime.NumGoroutine(),
					"users-now":       rsHub.CountOnlineUsers(),
				}).Info("server info")
			case <-closeConnTotal:
				return
			}
		}

	}()
	defer func() {
		closeConnTotal <- 1
		apiListener.Close()
		rsHub.Close()
		nc.Close()
	}()

	// block and listen syscall
	shutdow_observer := make(chan os.Signal, 1)
	t := template.Must(template.New("gusher slave start msg").Parse(slaveMsgFormat))
	t.Execute(os.Stdout, sc)
	signal.Notify(shutdow_observer, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-shutdow_observer:
		logger.Info("receive signal")
	case err := <-serverError:
		logger.Error(err)
	case err := <-rsHubErr:
		logger.Error("hub listen stopped: ", err)
	}
	return

}
