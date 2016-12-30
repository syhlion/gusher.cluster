package main

import (
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/syhlion/httplog"
	redisocket "github.com/syhlion/redisocket.v2"
	"github.com/syhlion/requestwork.v2"
	"github.com/urfave/cli"
	"github.com/urfave/negroni"
)

// master server
func master(c *cli.Context) {

	envInit(c)

	b, err := ioutil.ReadFile(public_pem_file)
	if err != nil {
		logger.Warn(err)
	}
	public_pem, rsaKeyErr := jwt.ParseRSAPublicKeyFromPEM(b)
	if rsaKeyErr != nil {
		logger.Warnf("Did not start \"%sdecode\" api", master_uri_prefix)
	}

	/*redis init*/
	rpool := redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", redis_addr)
	}, 10)
	/*Test redis connect*/
	_, err = rpool.Get().Do("PING")
	if err != nil {
		logger.Fatal(err)
	}
	rsender := redisocket.NewSender(rpool)

	/*externl ip*/
	externalIP, err := GetExternalIP()
	if err != nil {
		logger.Fatal(err)
	}

	/*api start*/
	apiListener, err := net.Listen("tcp", master_api_listen)
	if err != nil {
		logger.Fatal(err)
	}
	r := mux.NewRouter()

	sub := r.PathPrefix(master_uri_prefix).Subrouter()
	sub.HandleFunc("/push/{app_key}/{channel}/{event}", PushMessage(rsender)).Methods("POST")
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

	// block and listen syscall
	shutdow_observer := make(chan os.Signal, 1)
	logger.Info(name, " master start ! ")
	logger.Infof("listen redis in \"%s\"", redis_addr)
	logger.Infof("listen web api in \"%s\"", master_api_listen)
	logger.Infof("master uri preifx \"%s\"", master_uri_prefix)
	logger.Infof("localhost ip is \"%s\"", externalIP)
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
	envInit(c)

	/*redis init*/
	rpool := redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", redis_addr)
	}, 10)
	/*Test redis connect*/
	_, err := rpool.Get().Do("PING")
	if err != nil {
		logger.Fatal(err)
	}

	rsHub := redisocket.NewHub(rpool, c.Bool("debug"))
	rsHub.Config.Upgrader.WriteBufferSize = 8192
	rsHub.Config.Upgrader.ReadBufferSize = 8192
	rsHub.Config.MaxMessageSize = 4096
	rsHubErr := make(chan error, 1)
	go func() {
		rsHubErr <- rsHub.Listen(listenChannelPrefix)
	}()

	/*externl ip*/
	externalIP, err := GetExternalIP()
	if err != nil {
		logger.Fatal(err)
	}

	/*request worker*/
	worker := requestwork.New(50)

	/*api start*/
	apiListener, err := net.Listen("tcp", api_listen)
	if err != nil {
		logger.Fatal(err)
	}
	r := mux.NewRouter()

	wm := &WsManager{
		users:   make(map[*User]bool),
		RWMutex: &sync.RWMutex{},
		pool:    rpool,
		Hub:     rsHub,
		worker:  worker,
	}
	/*api end*/

	server := http.NewServeMux()

	sub := r.PathPrefix(api_uri_prefix).Subrouter()
	sub.HandleFunc("/ws/{app_key}", wm.Connect).Methods("GET")
	sub.HandleFunc("/auth", wm.Auth).Methods("POST")
	n := negroni.New()
	n.Use(httplog.NewLogger())
	n.UseHandler(r)
	server.Handle("/", n)
	serverError := make(chan error, 1)
	go func() {
		err := http.Serve(apiListener, server)
		serverError <- err
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
				logger.Infof("connection now: %v", wm.Count())
			case <-closeConnTotal:
				return
			}
		}

	}()

	defer func() {
		closeConnTotal <- 1
		apiListener.Close()
		wm.Close()
		rsHub.Close()
		rpool.Close()
	}()

	// block and listen syscall
	shutdow_observer := make(chan os.Signal, 1)
	logger.Info(name, " slave start ! ")
	logger.Infof("listen redis in \"%s\"", redis_addr)
	logger.Infof("listen web api in \"%s\"", api_listen)
	logger.Infof("api uri preifx \"%s\"", api_uri_prefix)
	logger.Infof("localhost ip is \"%s\"", externalIP)
	logger.Infof("decode service \"%s\"", decode_service)
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
