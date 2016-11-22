package main

import (
	"os"
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/joho/godotenv"
	"github.com/urfave/cli"
)

var env *string
var (
	version             string
	compileDate         string
	name                string
	listenChannelPrefix string
	cmdSlave            = cli.Command{
		Name:    "slave",
		Usage:   "start gusher.slave server",
		Aliases: []string{"sl"},
		Action:  slave,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "env-file,e",
				Usage: "import env file",
			},
			cli.BoolFlag{
				Name:  "debug,d",
				Usage: "open debug mode",
			},
		},
	}
	cmdMaster = cli.Command{
		Name:    "master",
		Usage:   "start gusher.master server",
		Action:  master,
		Aliases: []string{"ma"},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "env-file,e",
				Usage: "import env file",
			},
			cli.BoolFlag{
				Name:  "debug,d",
				Usage: "open debug mode",
			},
		},
	}
	logger            *Logger
	loglevel          string
	externalIP        string
	api_listen        string
	api_uri_prefix    string
	master_uri_prefix string
	master_api_listen string
	//master_remote_addr         string
	redis_addr string
	//remote_listen              string
	public_pem_file            string
	decode_service             string
	return_serverinfo_interval string
)

func init() {
	listenChannelPrefix = name + "." + version + "."
	/*logger init*/
	logger = GetLogger()
}

func envInit(c *cli.Context) {
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
	api_uri_prefix = os.Getenv("GUSHER_API_URI_PREFIX")
	if api_listen == "" {
		logger.Fatal("empty env GUSHER_API_URI_PREIFX")
	}
	master_uri_prefix = os.Getenv("GUSHER_MASTER_URI_PREFIX")
	if api_listen == "" {
		logger.Fatal("empty env GUSHER_MASTER_URI_PREFIX")
	}

	if c.Bool("debug") {
		logger.Logger.Level = logrus.DebugLevel
	} else {
		logger.Logger.Level = logrus.InfoLevel
	}

}

func main() {

	gusher := cli.NewApp()
	gusher.Name = name
	gusher.Author = "Scott (syhlion)"
	gusher.Usage = "websocket push server"
	gusher.UsageText = "very simple to use http request push message to websocket and very easy to scale"
	gusher.Version = version
	gusher.Commands = []cli.Command{
		cmdSlave,
		cmdMaster,
	}
	gusher.Compiled = time.Now()
	gusher.Run(os.Args)

}
