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

	"github.com/Sirupsen/logrus"
	jwt "github.com/dgrijalva/jwt-go"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/syhlion/redisocket.v2"
	"github.com/syhlion/requestwork.v2"
	"github.com/urfave/cli"
)

var env *string
var (
	version     string
	compileDate string
	name        string
	cmdSlave    = cli.Command{
		Name:   "slave",
		Usage:  "start gusher.slave server",
		Action: slave,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name: "env-file",
			},
		},
	}
	cmdMaster = cli.Command{
		Name:   "master",
		Usage:  "start gusher.master server",
		Action: master,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name: "env-file",
			},
		},
	}
	logger            *Logger
	loglevel          string
	externalIP        string
	api_listen        string
	master_api_listen string
	//master_remote_addr         string
	redis_addr string
	//remote_listen              string
	public_pem_file            string
	decode_service             string
	return_serverinfo_interval string
)

func init() {
	/*logger init*/
	logger = &Logger{logrus.New()}
	//logger.Level = logrus.DebugLevel
	switch loglevel {
	case "DEV":
		logger.Level = logrus.DebugLevel
		break
	case "PRODUCTION":
		logger.Level = logrus.InfoLevel
		break
	default:
		logger.Level = logrus.DebugLevel
		break
	}

}

//slave server
func slave(c *cli.Context) {
	varInit(c)

	/*redis init*/
	rpool := redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", redis_addr)
	}, 10)
	/*Test redis connect*/
	_, err := rpool.Get().Do("PING")
	if err != nil {
		logger.Fatal(err)
	}

	rsHub := redisocket.NewHub(rpool)
	rsHub.Config.Upgrader.WriteBufferSize = 8192
	rsHub.Config.Upgrader.ReadBufferSize = 8192
	rsHubErr := make(chan error, 1)
	go func() {
		rsHubErr <- rsHub.Listen()
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

	r.HandleFunc("/ws/{app_key}", wm.Connect).Methods("GET")
	http.Handle("/", handlers.CombinedLoggingHandler(os.Stdout, r))
	serverError := make(chan error, 1)
	go func() {
		err := http.Serve(apiListener, nil)
		serverError <- err
	}()

	defer func() {
		apiListener.Close()
		wm.Close()
		rsHub.Close()
		rpool.Close()
	}()

	// block and listen syscall
	shutdow_observer := make(chan os.Signal, 1)
	logger.Info(loglevel, " mode")
	logger.Info(name, " slave start ! ")
	logger.Infof("listen redis in %s", redis_addr)
	logger.Infof("listen web api  in %s", api_listen)
	logger.Infof("localhost ip is  %s", externalIP)
	logger.Infof("decode service  %s", decode_service)
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

// master server
func master(c *cli.Context) {

	varInit(c)

	b, err := ioutil.ReadFile(public_pem_file)
	if err != nil {
		logger.Fatal(err)
	}
	public_pem, err := jwt.ParseRSAPublicKeyFromPEM(b)
	if err != nil {
		logger.Fatal(err)
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

	r.HandleFunc("/api/push/{app_key}/{channel}/{event}", PushMessage(rpool)).Methods("POST")
	r.HandleFunc("/api/decode", DecodeJWT(public_pem)).Methods("POST")
	http.Handle("/", handlers.CombinedLoggingHandler(os.Stdout, r))
	serverError := make(chan error, 1)
	go func() {
		err := http.Serve(apiListener, nil)
		serverError <- err
	}()

	// block and listen syscall
	shutdow_observer := make(chan os.Signal, 1)
	logger.Info(loglevel, " mode")
	logger.Info(name, " master start ! ")
	logger.Infof("listen redis in %s", redis_addr)
	logger.Infof("listen web api in %s", master_api_listen)
	//	logger.Infof("listen master in  %s", remote_listen)
	logger.Infof("localhost ip is  %s", externalIP)
	signal.Notify(shutdow_observer, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	select {
	case <-shutdow_observer:
		logger.Info("Receive signal")
	case err := <-serverError:
		logger.Warn(err)
	}

}

func varInit(c *cli.Context) {
	/*env init*/
	if c.String("env-file") != "" {
		envfile := c.String("env-file")
		//flag.Parse()
		err := godotenv.Load(envfile)
		if err != nil {
			logger.Fatal(err)
		}
	}
	public_pem_file = os.Getenv("GUSHER_PUBLIC_PEM_FILE")
	if public_pem_file == "" {
		logger.Fatal("empty env GUSHER_PUBLIC_PEM_FILE")
	}
	decode_service = os.Getenv("GUSHER_DECODE_SERVICE")
	if decode_service == "" {
		logger.Fatal("empty env GUSHER_DECODE_SERVICE")
	}

	loglevel = os.Getenv("GUSHER_LOGLEVEL")
	if loglevel == "" {
		logger.Fatal("empty env GUSHER_LOGLEVEL")
	}
	redis_addr = os.Getenv("GUSHER_REDIS_ADDR")
	if redis_addr == "" {
		logger.Fatal("empty env GUSHER_REDIS_ADDR")
	}
	master_api_listen = os.Getenv("GUSHER_MASTER_API_LISTEN")
	if master_api_listen == "" {
		logger.Fatal("empty env GUSHER_MASTER_API_LISTEN")
	}
	redis_addr = os.Getenv("GUSHER_REDIS_ADDR")
	if redis_addr == "" {
		logger.Fatal("empty env GUSHER_REDIS_ADDR")
	}
	api_listen = os.Getenv("GUSHER_API_LISTEN")
	if api_listen == "" {
		logger.Fatal("empty env GUSHER_API_LISTEN")
	}

	/*log init*/
	switch loglevel {
	case "DEV":
		logger.Level = logrus.DebugLevel
		break
	case "PRODUCTION":
		logger.Level = logrus.InfoLevel
		break
	default:
		logger.Level = logrus.DebugLevel
		break
	}

}

func main() {
	gusher := cli.NewApp()
	gusher.Name = name
	gusher.Version = version
	gusher.Commands = []cli.Command{
		cmdSlave,
		cmdMaster,
	}
	gusher.Compiled = time.Now()
	gusher.Run(os.Args)

}
