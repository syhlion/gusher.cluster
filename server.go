package main

import (
	"os"
	"strconv"
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
	logger          *Logger
	masterMsgFormat = "\nmaster mode start at \"{{.GetStartTime}}\"\tserver ip:\"{{.ExternalIp}}\"\tversion:\"{{.Version}}\"\tcomplie at \"{{.CompileDate}}\"\n" +
		"api_listen:\"{{.ApiListen}}\"\tapi_preifx:\"{{.ApiPrefix}}\"\n" +
		"redis_server_addr:\"{{.RedisAddr}}\"\n" +
		"redis_server_max_idle:\"{{.RedisMaxIdle}}\"\n" +
		"redis_server_max_conn:\"{{.RedisMaxConn}}\"\n" +
		"public_key_location:\"{{.PublicKeyLocation}}\"\n\n"
	slaveMsgFormat = "\nslave mode start at \"{{.GetStartTime}}\"\tserver ip:\"{{.ExternalIp}}\"\tversion:\"{{.Version}}\"\tcomplie at \"{{.CompileDate}}\"\n" +
		"api_listen:\"{{.ApiListen}}\"\tapi_preifx:\"{{.ApiPrefix}}\"\n" +
		"redis_server_addr:\"{{.RedisAddr}}\"\n" +
		"redis_server_max_idle:\"{{.RedisMaxIdle}}\"\n" +
		"redis_server_max_conn:\"{{.RedisMaxConn}}\"\n" +
		"decode_service_addr:\"{{.DecodeServiceAddr}}\"\n\n"
)

func init() {
	listenChannelPrefix = name + "." + version + "."
	/*logger init*/
	logger = GetLogger()
}
func getSlaveConfig(c *cli.Context) (sc SlaveConfig) {
	sc = SlaveConfig{}
	envInit(c)
	sc.RedisAddr = os.Getenv("GUSHER_REDIS_ADDR")
	if sc.RedisAddr == "" {
		logger.Fatal("empty env GUSHER_REDIS_ADDR")
	}
	var err error
	sc.RedisMaxIdle, err = strconv.Atoi(os.Getenv("GUSHER_REDIS_MAX_IDLE"))
	if err != nil {
		sc.RedisMaxIdle = 80
	}
	sc.RedisMaxConn, err = strconv.Atoi(os.Getenv("GUSHER_REDIS_MAX_CONN"))
	if err != nil {
		sc.RedisMaxConn = 800
	}
	sc.ApiListen = os.Getenv("GUSHER_API_LISTEN")
	if sc.ApiListen == "" {
		logger.Fatal("empty env GUSHER_API_LISTEN")
	}
	sc.ApiPrefix = os.Getenv("GUSHER_API_URI_PREFIX")
	if sc.ApiPrefix == "" {
		logger.Fatal("empty env GUSHER_API_URI_PREIFX")
	}
	sc.DecodeServiceAddr = os.Getenv("GUSHER_DECODE_SERVICE")
	if sc.DecodeServiceAddr == "" {
		logger.Fatal("empty env GUSHER_DECODE_SERVICE")
	}
	sc.StartTime = time.Now()
	sc.CompileDate = compileDate
	sc.Version = version
	sc.ExternalIp, err = GetExternalIP()
	if err != nil {
		logger.Fatal("cant get ip")
	}
	return
}
func getMasterConfig(c *cli.Context) (mc MasterConfig) {
	envInit(c)
	mc = MasterConfig{}
	mc.PublicKeyLocation = os.Getenv("GUSHER_PUBLIC_PEM_FILE")
	if mc.PublicKeyLocation == "" {
		logger.Fatal("empty env GUSHER_PUBLIC_PEM_FILE")
	}
	mc.RedisAddr = os.Getenv("GUSHER_REDIS_ADDR")
	if mc.RedisAddr == "" {
		logger.Fatal("empty env GUSHER_REDIS_ADDR")
	}
	var err error
	mc.RedisMaxIdle, err = strconv.Atoi(os.Getenv("GUSHER_REDIS_MAX_IDLE"))
	if err != nil {
		mc.RedisMaxIdle = 10
	}
	mc.RedisMaxConn, err = strconv.Atoi(os.Getenv("GUSHER_REDIS_MAX_CONN"))
	if err != nil {
		mc.RedisMaxConn = 100
	}
	mc.ApiListen = os.Getenv("GUSHER_MASTER_API_LISTEN")
	if mc.ApiListen == "" {
		logger.Fatal("empty env GUSHER_MASTER_API_LISTEN")
	}
	mc.ApiPrefix = os.Getenv("GUSHER_MASTER_URI_PREFIX")
	if mc.ApiPrefix == "" {
		logger.Fatal("empty env GUSHER_MASTER_URI_PREFIX")
	}
	mc.StartTime = time.Now()
	mc.CompileDate = compileDate
	mc.Version = version
	mc.ExternalIp, err = GetExternalIP()
	if err != nil {
		logger.Fatal("cant get ip")
	}
	return
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

	if c.Bool("debug") {
		logger.Logger.Level = logrus.DebugLevel
	} else {
		logger.Logger.Level = logrus.InfoLevel
	}

}

func main() {
	cli.AppHelpTemplate += "\nWEBSITE:\n\t\thttps://github.com/syhlion/gusher.cluster\n\n"
	gusher := cli.NewApp()
	gusher.Name = name
	gusher.Author = "Scott (syhlion)"
	gusher.Usage = "websocket push server"
	gusher.UsageText = "very simple to use http request push message to websocket and very easy to scale"
	gusher.Version = version
	gusher.Compiled = time.Now()
	gusher.Commands = []cli.Command{
		cmdSlave,
		cmdMaster,
	}
	gusher.Run(os.Args)

}
