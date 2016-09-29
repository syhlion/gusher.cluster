package main

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"strconv"
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
	logger                     *Logger
	loglevel                   string
	externalIP                 string
	api_listen                 string
	master_api_listen          string
	master_addr                string
	redis_addr                 string
	remote_listen              string
	public_pem_file            string
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

	r_interval, err := strconv.Atoi(return_serverinfo_interval)
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

	rsHub := redisocket.NewHub(rpool)
	rsHub.Config.Upgrader.WriteBufferSize = 8192
	rsHub.Config.Upgrader.ReadBufferSize = 8192
	rsHubErr := make(chan error, 1)
	go func() {
		rsHubErr <- rsHub.Listen()
	}()

	/*externl ip*/
	externalIP, err := getExternalIP()
	if err != nil {
		logger.Fatal(err)
	}

	/*remote rpc*/
	client, err := rpc.Dial("tcp", master_addr)
	if err != nil {
		logger.Fatal("Cant remote rpc %s", err)
	}

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
		rpc:     client,
	}
	/*api end*/

	r.HandleFunc("/ws/{app_key}", wm.Connect).Methods("GET")
	http.Handle("/", handlers.CombinedLoggingHandler(os.Stdout, r))
	serverError := make(chan error, 1)
	go func() {
		err := http.Serve(apiListener, nil)
		serverError <- err
	}()

	/*remote process*/
	remoteErr := make(chan error, 1)
	go func() {
		t := time.NewTicker(time.Duration(r_interval) * time.Second)
		for {
			select {
			case <-t.C:
				var b bool
				info := getInfo(externalIP, wm.Count())

				err = client.Call("HealthTrack.Put", &info, &b)
				if err != nil {
					remoteErr <- err
					return
				}

			}
		}
	}()
	defer func() {
		apiListener.Close()
		client.Close()
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
	logger.Infof("master ip  %s", master_addr)
	signal.Notify(shutdow_observer, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	select {
	case <-shutdow_observer:
		logger.Info("receive signal")
	case err := <-serverError:
		logger.Error(err)
	case err := <-rsHubErr:
		logger.Error(err)
	case err := <-remoteErr:
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
	slaveInfos := &SlaveInfos{
		servers: make(map[string]ServerInfo),
		lock:    &sync.Mutex{},
	}
	/*remote start*/
	healthTrack := &HealthTrack{
		s: slaveInfos,
	}
	jrdecoder := &JWT_RSA_Decoder{
		key: public_pem,
	}
	addr, err := net.ResolveTCPAddr("tcp", remote_listen)
	if err != nil {
		logger.Fatal(err)
	}
	in, err := net.ListenTCP("tcp", addr)
	if err != nil {
		logger.Fatal(err)
	}
	rpc.Register(healthTrack)
	rpc.Register(jrdecoder)
	go func() {
		rpc.Accept(in)
	}()

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
	externalIP, err := getExternalIP()
	if err != nil {
		logger.Fatal(err)
	}

	/*api start*/
	apiListener, err := net.Listen("tcp", master_api_listen)
	if err != nil {
		logger.Fatal(err)
	}
	r := mux.NewRouter()

	r.HandleFunc("/api/system/slaveinfos", SystemInfo(slaveInfos)).Methods("GET")
	r.HandleFunc("/api/push/{app_key}/{channel}/{event}", PushMessage(rpool)).Methods("POST")
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
	logger.Infof("listen master in  %s", remote_listen)
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
	master_addr = os.Getenv("GUSHER_MASTER_ADDR")
	if master_addr == "" {
		logger.Fatal("empty env GUSHER_MASTER_ADDR")
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
	remote_listen = os.Getenv("GUSHER_REMOTE_LISTEN")
	if remote_listen == "" {
		logger.Fatal("empty env GUSHER_REMOTE_LISTEN")
	}
	redis_addr = os.Getenv("GUSHER_REDIS_ADDR")
	if redis_addr == "" {
		logger.Fatal("empty env GUSHER_REDIS_ADDR")
	}
	api_listen = os.Getenv("GUSHER_API_LISTEN")
	if api_listen == "" {
		logger.Fatal("empty env GUSHER_API_LISTEN")
	}
	return_serverinfo_interval = os.Getenv("GUSHER_RETURN_SERVERINFO_INTERVAL")
	if return_serverinfo_interval == "" {
		logger.Fatal("empty env GUSHER_RETURN_SERVERINFO_INTERVAL")
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
