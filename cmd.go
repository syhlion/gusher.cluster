package main

import (
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/syhlion/greq"
	"github.com/syhlion/httplog"
	redisocket "github.com/syhlion/redisocket.v2"
	"github.com/syhlion/requestwork.v2"
	"github.com/urfave/cli"
	"github.com/urfave/negroni"
)

// master server
func master(c *cli.Context) {

	mc := getMasterConfig(c)

	b, err := ioutil.ReadFile(mc.PublicKeyLocation)
	if err != nil {
		logger.Warn(err)
	}
	public_pem, rsaKeyErr := jwt.ParseRSAPublicKeyFromPEM(b)
	if rsaKeyErr != nil {
		logger.Warnf("Did not start \"%sdecode\" api", mc.ApiPrefix)
	}

	/*redis init*/
	rpool := redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", mc.RedisAddr)
	}, 10)
	rpool.MaxIdle = mc.RedisMaxIdle
	rpool.MaxActive = mc.RedisMaxConn

	/*Test redis connect*/
	err = RedisTestConn(rpool.Get())
	if err != nil {
		logger.Fatal(err)
	}
	rsender := redisocket.NewSender(rpool)

	/*api start*/
	apiListener, err := net.Listen("tcp", mc.ApiListen)
	if err != nil {
		logger.Fatal(err)
	}
	r := mux.NewRouter()

	sub := r.PathPrefix(mc.ApiPrefix).Subrouter()
	sub.HandleFunc("/push/{app_key}/{channel}/{event}", PushMessage(rsender)).Methods("POST")
	sub.HandleFunc("/push_batch/{app_key}", PushBatchMessage(rsender)).Methods("POST")
	sub.HandleFunc("/channels", GetAllChannel(rsender)).Methods("GET")
	if rsaKeyErr == nil {
		sub.HandleFunc("/decode", DecodeJWT(public_pem)).Methods("POST")
	}
	server := http.NewServeMux()
	n := negroni.New()
	n.Use(httplog.NewLogger())
	n.UseHandler(r)
	server.Handle("/", http.TimeoutHandler(n, 3*time.Second, "Timeout"))
	serverError := make(chan error, 1)
	go func() {
		err := http.Serve(apiListener, server)
		serverError <- err
	}()
	go func() {
		logger.Error(http.ListenAndServe(":7799", nil))
	}()

	// block and listen syscall
	shutdow_observer := make(chan os.Signal, 1)
	t := template.Must(template.New("gusher master start msg").Parse(masterMsgFormat))
	t.Execute(os.Stdout, mc)
	signal.Notify(shutdow_observer, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	select {
	case <-shutdow_observer:
		logger.Info("Receive signal")
	case err := <-serverError:
		logger.Warn(err)
	}

}

//slave server
func slave(c *cli.Context) {

	sc := getSlaveConfig(c)
	/*redis init*/
	rpool := redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", sc.RedisAddr)
	}, 10)

	rpool.MaxIdle = sc.RedisMaxIdle
	rpool.MaxActive = sc.RedisMaxConn
	rpool.Wait = true
	rpool.IdleTimeout = 30 * time.Second
	rpool.TestOnBorrow = func(c redis.Conn, t time.Time) error {
		if time.Since(t) < time.Minute {
			return nil
		}
		_, err := c.Do("PING")
		return err
	}

	/*Test redis connect*/
	err := RedisTestConn(rpool.Get())
	if err != nil {
		logger.Fatal(err)
	}

	rsHub := redisocket.NewHub(rpool, c.Bool("debug"))
	rsHub.Config.Upgrader.WriteBufferSize = 1024
	rsHub.Config.Upgrader.ReadBufferSize = 1024
	rsHub.Config.MaxMessageSize = 1024
	rsHubErr := make(chan error, 1)
	go func() {
		rsHubErr <- rsHub.Listen(listenChannelPrefix)
	}()

	/*request worker*/
	worker := requestwork.New(50)
	client := greq.New(worker, 15*time.Second)
	/*api start*/
	apiListener, err := net.Listen("tcp", sc.ApiListen)
	if err != nil {
		logger.Fatal(err)
	}
	r := mux.NewRouter()

	wm := &WsManager{
		pool:   rpool,
		Hub:    rsHub,
		client: client,
	}
	/*api end*/

	server := http.NewServeMux()

	sub := r.PathPrefix(sc.ApiPrefix).Subrouter()
	sub.HandleFunc("/ws/{app_key}", wm.Connect()).Methods("GET")
	sub.HandleFunc("/auth", wm.Auth(sc)).Methods("POST")
	n := negroni.New()
	n.Use(httplog.NewLogger())
	n.UseHandler(r)
	server.Handle("/", n)
	serverError := make(chan error, 1)
	go func() {
		err := http.Serve(apiListener, server)
		serverError <- err
	}()
	go func() {
		logger.Error(http.ListenAndServe(":8899", nil))
	}()

	closeConnTotal := make(chan int, 0)
	//固定30秒log出 現在連線人數
	go func() {
		t := time.NewTicker(1 * time.Second)
		defer func() {
			t.Stop()
		}()

		for {
			select {
			case <-t.C:
				logger.Infof("users now: %v,channels now: %v", wm.Hub.CountOnlineUsers(), wm.Hub.CountChannels())
			case <-closeConnTotal:
				return
			}
		}

	}()

	defer func() {
		closeConnTotal <- 1
		apiListener.Close()
		rsHub.Close()
		rpool.Close()
	}()

	// block and listen syscall
	shutdow_observer := make(chan os.Signal, 1)
	t := template.Must(template.New("gusher slave start msg").Parse(slaveMsgFormat))
	t.Execute(os.Stdout, sc)
	signal.Notify(shutdow_observer, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	select {
	case <-shutdow_observer:
		logger.Info("receive signal")
	case err := <-serverError:
		logger.Error(err)
	case err := <-rsHubErr:
		logger.Error(err)
	}
	return

}
