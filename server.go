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
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/syhlion/redisocket"
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
	pool := redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", redis_err)
	}, 10)
	rsocket = redisocket.NewApp(pool)

	/*api start*/
	apiListener, err := net.Listen("tcp", public_api_addr)
	if err != nil {
		os.Exit(1)
	}
	r := mux.NewRouter()
	wm := &WsManager{
		users:   make(map[*User]bool),
		RWMutex: &sync.RWMutex{},
		pool:    pool,
	}
	r.HandleFunc("/ws", wm.Connect)
	http.Handle("/", r)
	go http.Serve(apiListener, nil)
	// block and listen syscall
	shutdow_observer := make(chan os.Signal, 1)
	log.Println(name, "Start ! ")
	signal.Notify(shutdow_observer, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	log.Println("Receive signal:", <-shutdow_observer)

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
