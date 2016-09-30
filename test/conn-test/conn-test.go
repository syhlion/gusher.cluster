package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/syhlion/requestwork.v2"
	"github.com/urfave/cli"
)

var (
	name     string
	version  string
	cmdStart = cli.Command{
		Name:   "start",
		Usage:  "connect ws cli",
		Action: start,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name: "env-file",
			},
		},
	}
)

func start(c *cli.Context) {
	if c.String("env-file") != "" {
		envfile := c.String("env-file")
		//flag.Parse()
		err := godotenv.Load(envfile)
		if err != nil {
			log.Fatal(err)
		}
	}
	ws_api := os.Getenv("GUSHER-CONN-TEST_WS_API")
	if ws_api == "" {
		log.Fatal("empty env GUSHER-CONN-TEST_WS_API")
	}
	push_api := os.Getenv("GUSHER-CONN-TEST_PUSH_API")
	if push_api == "" {
		log.Fatal("empty env GUSHER-CONN-TEST_PUSH_API")
	}
	login_msg := os.Getenv("GUSHER-CONN-TEST_LOGIN_MESSAGE")
	if login_msg == "" {
		log.Fatal("empty env GUSHER-CONN-TEST_LOGIN_MESSAGE")
	}
	sub_msg := os.Getenv("GUSHER-CONN-TEST_SUBSCRIBE_MESSAGE")
	if sub_msg == "" {
		log.Fatal("empty env GUSHER-CONN-TEST_SUBSCRIBE_MESSAGE")
	}
	push_msg := os.Getenv("GUSHER-CONN-TEST_PUSH_MESSAGE")
	if push_msg == "" {
		log.Fatal("empty env GUSHER-CONN-TEST_PUSH_MESSAGE")
	}
	wsurl, err := url.Parse(ws_api)
	if err != nil {
		log.Fatal(err)
	}
	pushurl, err := url.Parse(push_api)
	if err != nil {
		log.Fatal(err)
	}
	wsHeaders := http.Header{
		"Origin":                   {wsurl.String()},
		"Sec-WebSocket-Extensions": {"permessage-deflate; client_max_window_bits, x-webkit-deflate-frame"},
	}
	rawConn, err := net.Dial("tcp", wsurl.Host)
	if err != nil {
		fmt.Println(err)
		return
	}
	conn, _, err := websocket.NewClient(rawConn, wsurl, wsHeaders, 1024, 1024)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = conn.WriteMessage(websocket.TextMessage, []byte(login_msg))
	err = conn.WriteMessage(websocket.TextMessage, []byte(sub_msg))
	time.Sleep(1 * time.Second)
	sucess_chan := make(chan int)
	go func() {
		for {
			_, d, err := conn.ReadMessage()
			if err != nil {
				log.Fatal(err)
				return
			}
			data, _ := jsonparser.GetString(d, "data")
			if data == push_msg {
				sucess_chan <- 1
			}
		}
	}()

	//push start
	work := requestwork.New(5)
	v := url.Values{}

	v.Add("data", push_msg)
	req, err := http.NewRequest("POST", pushurl.String(), bytes.NewBufferString(v.Encode()))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(v.Encode())))

	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	err = work.Execute(ctx, req, func(resp *http.Response, e error) (err error) {
		if e != nil {
			return
		}
		defer resp.Body.Close()
		return
	})
	log.Println("Waiting...")
	<-sucess_chan
	log.Println("Scuess")
	defer func() {
		conn.Close()
	}()

	return
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
