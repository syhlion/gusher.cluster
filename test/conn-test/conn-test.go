package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
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
	wg        sync.WaitGroup
	listen_wg sync.WaitGroup
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
	sub_resp := os.Getenv("GUSHER-CONN-TEST_SUBSCRIBE_RESPONSE")
	if sub_resp == "" {
		log.Fatal("empty env GUSHER-CONN-TEST_SUBSCRIBE_RESPONSE")
	}
	push_msg := os.Getenv("GUSHER-CONN-TEST_PUSH_MESSAGE")
	if push_msg == "" {
		log.Fatal("empty env GUSHER-CONN-TEST_PUSH_MESSAGE")
	}
	connections := os.Getenv("GUSHER-CONN-TEST_CONNECTIONS")
	if connections == "" {
		log.Fatal("empty env GUSHER-CONN-TEST_CONNECTIONS")
	}
	conn_total, err := strconv.Atoi(connections)
	if err != nil {
		log.Fatal(err)
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
	conns := make([]*websocket.Conn, 0)
	for i := 0; i < conn_total; i++ {
		wg.Add(1)
		rawConn, err := net.Dial("tcp", wsurl.Host)
		if err != nil {
			log.Fatal(err)
			wg.Done()
			continue
		}

		conn, _, err := websocket.NewClient(rawConn, wsurl, wsHeaders, 1024, 1024)
		if err != nil {
			rawConn.Close()
			wg.Done()
			log.Fatal(err)
			continue
		}
		err = conn.WriteMessage(websocket.TextMessage, []byte(login_msg))
		if err != nil {
			rawConn.Close()
			conn.Close()
			wg.Done()
			log.Warn(err)
			continue
		}
		conns = append(conns, conn)
	}
	for i, conn := range conns {
		listen_wg.Add(1)
		go func(i int, conn *websocket.Conn) {
			for {
				_, d, err := conn.ReadMessage()
				if err != nil {
					log.Fatal(err)
					listen_wg.Done()
					wg.Done()
					return
				}

				if string(d) == sub_resp {
					listen_wg.Done()
				}
				log.Println(i, " slave repsonse message", string(d))
				data, _ := jsonparser.GetString(d, "data")
				if data == push_msg {
					wg.Done()
					return
				}
			}
		}(i, conn)
		err = conn.WriteMessage(websocket.TextMessage, []byte(sub_msg))
		if err != nil {
			conn.Close()
			continue
		}
	}

	listen_wg.Wait()
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
	var firstTime time.Time
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	err = work.Execute(ctx, req, func(resp *http.Response, e error) (err error) {
		if e != nil {
			return
		}
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		log.Println("master response", string(b))
		firstTime = time.Now()
		return
	})
	log.Println("Waiting...")
	wg.Wait()
	log.Println("Sucess")
	t := time.Now().Sub(firstTime)
	log.Printf("Total Use time:%s", t)

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
