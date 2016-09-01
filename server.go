package main

import (
	"flag"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/codegangsta/cli"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/syhlion/redisocket.v2"
	"github.com/syhlion/requestwork.v2"
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
	}
	cmdMaster = cli.Command{
		Name:   "master",
		Usage:  "start gusher.master server",
		Action: master,
	}
	rpool                      *redis.Pool
	worker                     *requestwork.Worker
	rsocket                    redisocket.App
	logger                     *Logger
	client                     *rpc.Client
	loglevel                   string
	externalIP                 string
	api_listen                 string
	master_api_listen          string
	master_addr                string
	redis_addr                 string
	remote_listen              string
	return_serverinfo_interval string
	wm                         *WsManager
	slaveInfos                 *SlaveInfos
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
	varInit()

	r_interval, err := strconv.Atoi(return_serverinfo_interval)
	if err != nil {
		logger.Fatal(err)
	}

	rsocket = redisocket.NewApp(rpool)
	rsocketErr := make(chan error, 1)
	go func() {
		err := rsocket.Listen()
		rsocketErr <- err
	}()

	/*api start*/
	apiListener, err := net.Listen("tcp", api_listen)
	if err != nil {
		logger.Fatal(err)
	}
	r := mux.NewRouter()

	worker = requestwork.New(50)
	wm = &WsManager{
		users:   make(map[*User]bool),
		RWMutex: &sync.RWMutex{},
		pool:    rpool,
	}
	/*api end*/

	/*remote rpc*/
	client, err = rpc.Dial("tcp", "127.0.0.1:1234")
	if err != nil {
		logger.Fatal("Cant remote rpc %s", err)
	}

	/*remote process*/
	remoteErr := make(chan error, 1)
	go func() {
		t := time.NewTicker(time.Duration(r_interval) * time.Second)
		for {
			select {
			case <-t.C:
				var b bool
				info := getInfo()
				err = client.Call("HealthTrack.Put", &info, &b)
				if err != nil {
					remoteErr <- err
					return
				}

			}
		}
	}()

	/*remote preocess end*/

	r.HandleFunc("/ws/{app_key}", HttpUse(wm.Connect, AuthMiddleware)).Methods("GET")
	http.Handle("/", handlers.CombinedLoggingHandler(os.Stdout, r))
	serverError := make(chan error, 1)
	go func() {
		err := http.Serve(apiListener, nil)
		serverError <- err
	}()

	// block and listen syscall
	shutdow_observer := make(chan os.Signal, 1)
	logger.Info(loglevel, " mode")
	logger.Info(name, " slave start ! ")
	logger.Infof("listen redis in %s", redis_addr)
	logger.Infof("listen TCP  in %s", api_listen)
	logger.Infof("locahost IP is  %s", externalIP)
	signal.Notify(shutdow_observer, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	select {
	case <-shutdow_observer:
		logger.Info("receive signal")
	case err := <-serverError:
		logger.Fatal(err)
	case err := <-rsocketErr:
		logger.Fatal(err)
	case err := <-remoteErr:
		logger.Fatal(err)
	}

}

// master server
func master(c *cli.Context) {

	varInit()

	slaveInfos = &SlaveInfos{
		servers: make(map[string]ServerInfo),
		lock:    &sync.Mutex{},
	}
	/*remote start*/
	healthTrack := &HealthTrack{
		s: slaveInfos,
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
	go func() {
		rpc.Accept(in)
	}()

	/*api start*/
	apiListener, err := net.Listen("tcp", master_api_listen)
	if err != nil {
		logger.Fatal(err)
	}
	r := mux.NewRouter()

	r.HandleFunc("/api/system/slaveinfos", SystemInfo).Methods("GET")
	r.HandleFunc("/api/check/{app_key}", CheckAppKey).Methods("GET")
	r.HandleFunc("/api/push/{app_key}/{channel}/{event}", PushMessage).Methods("POST")
	http.Handle("/", handlers.CombinedLoggingHandler(os.Stdout, r))
	serverError := make(chan error, 1)
	go func() {
		err := http.Serve(apiListener, nil)
		serverError <- err
	}()

	/*Test redis connect*/
	_, err = rpool.Get().Do("PING")
	if err != nil {
		logger.Fatal(err)
	}

	// block and listen syscall
	shutdow_observer := make(chan os.Signal, 1)
	logger.Info(loglevel, " mode")
	logger.Info(name, " master start ! ")
	logger.Infof("listen redis in %s", redis_addr)
	logger.Infof("listen TCP  in %s", master_api_listen)
	logger.Infof("locahost IP is  %s", externalIP)
	signal.Notify(shutdow_observer, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	select {
	case <-shutdow_observer:
		logger.Info("Receive signal")
	case err := <-serverError:
		logger.Warn(err)
	}

}

func varInit() {
	/*env init*/
	pwd, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		logger.Fatal(err)
	}
	envfile := flag.String("env", pwd+"/.env", ".env file path")
	flag.Parse()
	err = godotenv.Load(*envfile)
	if err != nil {
		logger.Fatal(err)
	}

	loglevel = os.Getenv("LOGLEVEL")
	redis_addr = os.Getenv("REDIS_ADDR")
	master_api_listen = os.Getenv("MASTER_API_LISTEN")
	remote_listen = os.Getenv("REMOTE_LISTEN")
	redis_addr = os.Getenv("REDIS_ADDR")
	api_listen = os.Getenv("API_LISTEN")
	return_serverinfo_interval = os.Getenv("RETURN_SERVERINFO_INTERVAL")

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

	/*redis init*/
	rpool = redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", redis_addr)
	}, 10)

	/*externl ip*/
	externalIP, err = getExternalIP()
	if err != nil {
		logger.Fatal(err)
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
