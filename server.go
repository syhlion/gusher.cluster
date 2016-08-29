package main

import (
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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
	cmdStart    = cli.Command{
		Name:   "start",
		Usage:  "Start gusher.cluster Server",
		Action: start,
	}
	rpool   *redis.Pool
	worker  *requestwork.Worker
	rsocket redisocket.App
	logger  *Logger
)

func init() {
	logger = &Logger{logrus.New()}
	logger.Level = logrus.DebugLevel
	//logger.Formatter = &logrus.TextFormatter{}
}

func start(c *cli.Context) {

	pwd, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		logger.Warn(err)
		os.Exit(1)
	}
	envfile := flag.String("env", pwd+"/.env", ".env file path")
	flag.Parse()
	err = godotenv.Load(*envfile)
	if err != nil {
		logger.Warn(err)
		os.Exit(1)
	}
	var (
		redis_addr      = os.Getenv("REDIS_ADDR")
		public_api_addr = os.Getenv("PUBLIC_API_ADDR")
	)

	/*redis start*/
	rpool = redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", redis_addr)
	}, 10)
	rsocket = redisocket.NewApp(rpool)
	rsocketErr := make(chan error, 1)
	go func() {
		err := rsocket.Listen()
		rsocketErr <- err
	}()

	ip, err := externalIP()
	if err != nil {
		logger.Warn(err)
		os.Exit(1)
	}

	/*api start*/
	apiListener, err := net.Listen("tcp", public_api_addr)
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}
	r := mux.NewRouter()

	worker = requestwork.New(50)
	wm := &WsManager{
		users:   make(map[*User]bool),
		RWMutex: &sync.RWMutex{},
		pool:    rpool,
	}

	r.HandleFunc("/ws/{app_key}", HttpUse(wm.Connect, AuthMiddleware)).Methods("GET")
	http.Handle("/", handlers.CombinedLoggingHandler(os.Stdout, r))
	serverError := make(chan error, 1)
	go func() {
		err := http.Serve(apiListener, nil)
		serverError <- err
	}()
	// block and listen syscall
	shutdow_observer := make(chan os.Signal, 1)
	logger.Info(name, "Start ! ")
	logger.Infof("Listen redis in %s", redis_addr)
	logger.Infof("Listen TCP  in %s", public_api_addr)
	logger.Infof("Locahost IP is  %s", ip)
	signal.Notify(shutdow_observer, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	select {
	case <-shutdow_observer:
		logger.Info("Receive signal")
	case err := <-serverError:
		logger.Warn(err)
	case err := <-rsocketErr:
		logger.Warn(err)
	}

}

func main() {
	gusher := cli.NewApp()
	gusher.Name = name
	gusher.Version = version
	gusher.Commands = []cli.Command{
		cmdStart,
	}
	gusher.Compiled = time.Now()
	gusher.Run(os.Args)

}
