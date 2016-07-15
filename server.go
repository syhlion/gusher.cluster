package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/codegangsta/cli"
	"github.com/garyburd/redigo/redis"
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
		redis_err = os.Getenv("REDIS_ADDR")
	)
	pool := redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", redis_err)
	}, 10)
	rsocket = redisocket.NewApp(pool)
}

func main() {
	gusher := cli.NewApp()
	gusher.Name = name
	gusher.Version = version
	gusher.Run(os.Args)

}
