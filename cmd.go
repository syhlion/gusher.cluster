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

	jwt "github.com/golang-jwt/jwt"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
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
		c, err := redis.Dial("tcp", mc.RedisAddr)
		if err != nil {
			return nil, err
		}
		_, err = c.Do("SELECT", mc.RedisDb)
		if err != nil {
			c.Close()
			return nil, err
		}
		return c, nil
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
	if rsaKeyErr == nil {
		sub.HandleFunc("/decode", DecodeJWT(public_pem)).Methods("POST")
	}
	n := negroni.New()
	n.Use(httplog.NewLogger(true))
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
func runtimeStats() (m *runtime.MemStats) {
	m = &runtime.MemStats{}

	//log.Println("# goroutines: ", runtime.NumGoroutine())
	runtime.ReadMemStats(m)
	//log.Println("Memory Acquired: ", m.Sys)
	//log.Println("Memory Used    : ", m.Alloc)
	return m
}

//slave server
func slave(c *cli.Context) {

	sc := getSlaveConfig(c)
	/*redis init*/
	rpool := redis.NewPool(func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", sc.RedisAddr)
		if err != nil {
			return nil, err
		}
		_, err = c.Do("SELECT", sc.RedisDb)
		if err != nil {
			c.Close()
			return nil, err
		}
		return c, nil
	}, 10)

	rpool.MaxIdle = sc.RedisMaxIdle
	rpool.MaxActive = sc.RedisMaxConn
	rpool.Wait = true
	rpool.IdleTimeout = 240 * time.Second
	rpool.TestOnBorrow = func(c redis.Conn, t time.Time) error {
		if time.Since(t) < time.Minute {
			return nil
		}
		_, err := c.Do("PING")
		return err
	}

	jobRpool := redis.NewPool(func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", sc.RedisJobAddr)
		if err != nil {
			return nil, err
		}
		_, err = c.Do("SELECT", sc.RedisJobDb)
		if err != nil {
			c.Close()
			return nil, err
		}
		return c, nil
	}, 10)

	jobRpool.MaxIdle = sc.RedisJobMaxIdle
	jobRpool.MaxActive = sc.RedisJobMaxConn
	jobRpool.Wait = true
	jobRpool.IdleTimeout = 240 * time.Second
	jobRpool.TestOnBorrow = func(c redis.Conn, t time.Time) error {
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
	err = RedisTestConn(jobRpool.Get())
	if err != nil {
		logger.Fatal(err)
	}

	rsHub := redisocket.NewHub(rpool, logger.GetLogger(), c.Bool("debug"))
	rsHub.Config.MaxMessageSize = int64(sc.MaxMessage)
	rsHub.Config.ScanInterval = sc.ScanInterval
	rsHub.Config.Upgrader.ReadBufferSize = sc.ReadBuffer
	rsHub.Config.Upgrader.WriteBufferSize = sc.WriteBuffer
	rsHubErr := make(chan error, 1)
	go func() {
		rsHubErr <- rsHub.Listen(listenChannelPrefix)
	}()

	/*request worker*/
	worker := requestwork.New(50)
	client := greq.New(worker, 15*time.Second, c.Bool("debug"))
	/*api start*/
	apiListener, err := net.Listen("tcp", sc.ApiListen)
	if err != nil {
		logger.Fatal(err)
	}
	r := mux.NewRouter()

	/*api end*/

	//server := http.NewServeMux()

	sub := r.PathPrefix(sc.ApiPrefix).Subrouter()
	sub.HandleFunc("/ws/{app_key}", WsConnect(sc, rpool, jobRpool, rsHub, client)).Methods("GET")
	sub.HandleFunc("/wtf/{app_key}", WtfConnect(sc, rpool, jobRpool, rsHub, client)).Methods("GET")
	sub.HandleFunc("/auth", WsAuth(sc, rpool, client)).Methods("POST")
	sub.HandleFunc("/ping", Ping()).Methods("GET")
	n := negroni.New()
	n.Use(httplog.NewLogger(true))
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
		logger.Error(http.ListenAndServe(":8899", nil))
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
				logger.WithFields(logrus.Fields{
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
		logger.Error("redis sub connection diconnect ", err)
	}
	return

}
