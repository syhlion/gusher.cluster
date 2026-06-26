package main

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/joho/godotenv"
	"github.com/urfave/cli"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

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
		"api_listen:\"{{.ApiListen}}\"\n" +
		"nats_addr:\"{{.NatsAddr}}\"\n" +
		"public_key_location:\"{{.PublicKeyLocation}}\"\n\n"
	slaveMsgFormat = "\nslave mode start at \"{{.GetStartTime}}\"\tserver ip:\"{{.ExternalIp}}\"\tversion:\"{{.Version}}\"\tcomplie at \"{{.CompileDate}}\"\n" +
		"api_listen:\"{{.ApiListen}}\"\n" +
		"read_buffer:\"{{.ReadBuffer}}\"\twrite_buffer:\"{{.WriteBuffer}}\"\tmax_message_size:\"{{.MaxMessage}}\"\tscan_interval:\"{{.ScanInterval}}\"\tlog_sys_interval:\"{{.LogInterval}}\"\n" +
		"nats_addr:\"{{.NatsAddr}}\"\n" +
		"public_key_location:\"{{.PublicKeyLocation}}\"\n\n"
)

func init() {
	listenChannelPrefix = name + "."
	/*logger init*/
	logger = GetLogger()
}
func getSlaveConfig(c *cli.Context) (sc SlaveConfig) {
	sc = SlaveConfig{}
	envInit(c)

	var err error
	sc.ApiListen = os.Getenv("GUSHER_API_LISTEN")
	if sc.ApiListen == "" {
		logger.Fatal("empty env GUSHER_API_LISTEN")
	}
	logInterval, err := strconv.Atoi(os.Getenv("GUSHER_LOG_SYS_INTERVAL"))
	if err != nil {
		logInterval = 30
	}
	sc.LogInterval = time.Duration(logInterval) * time.Second
	scanInterval, err := strconv.Atoi(os.Getenv("GUSHER_SCAN_INTERVAL"))
	if err != nil {
		scanInterval = 30
	}
	sc.ScanInterval = time.Duration(scanInterval) * time.Second
	sc.MaxMessage, err = strconv.Atoi(os.Getenv("GUSHER_RECEIVE_MAX_MESSAGE_SIZE"))
	if err != nil {
		sc.MaxMessage = 512
	}
	sc.ReadBuffer, err = strconv.Atoi(os.Getenv("GUSHER_READ_BUFFER"))
	if err != nil {
		sc.ReadBuffer = 1024
	}
	sc.WriteBuffer, err = strconv.Atoi(os.Getenv("GUSHER_WRITE_BUFFER"))
	if err != nil {
		sc.WriteBuffer = 8192
	}
	sc.NatsAddr = os.Getenv("GUSHER_NATS_ADDR")
	if sc.NatsAddr == "" {
		logger.Fatal("empty env GUSHER_NATS_ADDR")
	}
	// 本機 JWT 驗證(取代 decode service):slave 需公鑰
	sc.PublicKeyLocation = os.Getenv("GUSHER_PUBLIC_PEM_FILE")
	if sc.PublicKeyLocation == "" {
		logger.Fatal("empty env GUSHER_PUBLIC_PEM_FILE")
	}
	// log 格式/輸出由 setupLoggingFromEnv 處理(GUSHER_LOG_*)
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
	mc.NatsAddr = os.Getenv("GUSHER_NATS_ADDR")
	if mc.NatsAddr == "" {
		logger.Fatal("empty env GUSHER_NATS_ADDR")
	}
	mc.PublicKeyLocation = os.Getenv("GUSHER_PUBLIC_PEM_FILE")
	if mc.PublicKeyLocation == "" {
		logger.Fatal("empty env GUSHER_PUBLIC_PEM_FILE")
	}
	var err error
	mc.ApiListen = os.Getenv("GUSHER_MASTER_API_LISTEN")
	if mc.ApiListen == "" {
		logger.Fatal("empty env GUSHER_MASTER_API_LISTEN")
	}
	// log 格式/輸出由 setupLoggingFromEnv 處理(GUSHER_LOG_*)

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
		logLevel.Set(slog.LevelDebug)
	} else {
		logLevel.Set(slog.LevelInfo)
	}

}

func main() {
	cli.AppHelpTemplate += "\nWEBSITE:\n\t\thttps://github.com/syhlion/gusher.cluster\n\n"
	gusher := cli.NewApp()
	gusher.Name = name
	gusher.Author = "Scott (syhlion)"
	gusher.Usage = "very simple to use http request push message to websocket and very easy to scale"
	gusher.UsageText = "gusher.cluster [slave|master] [-e envfile] [-d]"
	gusher.Version = version
	gusher.Compiled = time.Now()
	gusher.Commands = []cli.Command{
		cmdSlave,
		cmdMaster,
	}
	gusher.Run(os.Args)

}
