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
	api_uri_prefix = os.Getenv("GUSHER_API_URI_PREFIX")
	if api_listen == "" {
		logger.Fatal("empty env GUSHER_API_URI_PREIFX")
	}
	master_uri_prefix = os.Getenv("GUSHER_MASTER_URI_PREFIX")
	if api_listen == "" {
		logger.Fatal("empty env GUSHER_MASTER_URI_PREFIX")
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
