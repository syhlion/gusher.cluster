package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

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
)

func start(c *cli.Context) {

	pwd, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	envfile := flag.String("env", pwd+"/.env", ".env file path")
	flag.Parse()
	err = godotenv.Load(*envfile)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	var (
		redis_err       = os.Getenv("REDIS_ADDR")
		public_api_addr = os.Getenv("PUBLIC_API_ADDR")
	)

	/*redis start*/
	rpool = redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", redis_err)
	}, 10)
	rsocket = redisocket.NewApp(rpool)
	rsocketErr := make(chan error, 1)
	go func() {
		err := rsocket.Listen()
		rsocketErr <- err
	}()

	/*api start*/
	apiListener, err := net.Listen("tcp", public_api_addr)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	r := mux.NewRouter()

	//TODO
	worker = requestwork.New(50)
	wm := &WsManager{
		users:   make(map[*User]bool),
		RWMutex: &sync.RWMutex{},
		pool:    rpool,
	}

	r.HandleFunc("/ws/{app_key}", HttpUse(wm.Connect, AuthMiddleware))
	http.Handle("/", handlers.LoggingHandler(os.Stdout, r))
	serverError := make(chan error, 1)
	go func() {
		err := http.Serve(apiListener, nil)
		serverError <- err
	}()
	// block and listen syscall
	shutdow_observer := make(chan os.Signal, 1)
	log.Println(name, "Start ! ")
	signal.Notify(shutdow_observer, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	select {
	case <-shutdow_observer:
		log.Println("Receive signal")
	case err := <-serverError:
		log.Println(err)
	case err := <-rsocketErr:
		log.Println(err)
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
