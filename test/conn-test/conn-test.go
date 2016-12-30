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
	"sync/atomic"
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
		Name:    "run",
		Aliases: []string{"r"},
		Usage:   "test connect to websocket ",
		Action:  start,
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
	ws_auth_api := os.Getenv("GUSHER-CONN-TEST_WS_AUTH_API")
	if ws_auth_api == "" {
		log.Fatal("empty env GUSHER-CONN-TEST_WS_AUTH_API")
	}
	push_api := os.Getenv("GUSHER-CONN-TEST_PUSH_API")
	if push_api == "" {
		log.Fatal("empty env GUSHER-CONN-TEST_PUSH_API")
	}
	jwt := os.Getenv("GUSHER-CONN-TEST_JWT")
	if jwt == "" {
		log.Fatal("empty env GUSHER-CONN-TEST_JWT")
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
	/*auth*/

	wsAuthurl, err := url.Parse(ws_auth_api)
	if err != nil {
		log.Fatal(err)
	}
	work := requestwork.New(5)
	loginUrl := url.Values{}

	loginUrl.Add("jwt", jwt)
	tokenChan := make(chan string, conn_total)
	tokenGroup := sync.WaitGroup{}
	for i := 0; i < conn_total; i++ {
		tokenGroup.Add(1)
		go func() {
			defer tokenGroup.Done()
			req, err := http.NewRequest("POST", wsAuthurl.String(), bytes.NewBufferString(loginUrl.Encode()))
			if err != nil {
				log.Fatal(err)
			}
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Add("Content-Length", strconv.Itoa(len(loginUrl.Encode())))
			ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
			err = work.Execute(ctx, req, func(resp *http.Response, e error) (err error) {
				if e != nil {
					return e
				}
				defer resp.Body.Close()
				b, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return err
				}
				v, err := jsonparser.GetString(b, "token")
				if err != nil {
					return err
				}
				tokenChan <- v
				return
			})
		}()

	}
	tokenGroup.Wait()
	close(tokenChan)
	/**/
	wsurlChan := make(chan *url.URL, conn_total)
	wsurlGroup := sync.WaitGroup{}
	for v := range tokenChan {
		wsurlGroup.Add(1)
		go func(v string) {
			defer wsurlGroup.Done()
			wsurl, err := url.Parse(ws_api + "?token=" + v)
			if err != nil {
				return
			}
			wsurlChan <- wsurl

		}(v)
	}
	wsurlGroup.Wait()
	close(wsurlChan)
	pushurl, err := url.Parse(push_api)
	if err != nil {
		log.Fatal(err)
	}
	wsHeaders := http.Header{
		"Origin":                   {"*"},
		"Sec-WebSocket-Extensions": {"permessage-deflate; client_max_window_bits, x-webkit-deflate-frame"},
	}
	connChan := make(chan *websocket.Conn, conn_total)

	connGroup := sync.WaitGroup{}
	log.Infof("%v connect start!", conn_total)
	for wsurl := range wsurlChan {
		connGroup.Add(1)
		go func(wsurl *url.URL) {
			defer connGroup.Done()
			rawConn, err := net.Dial("tcp", wsurl.Host)
			if err != nil {
				return
			}

			conn, _, err := websocket.NewClient(rawConn, wsurl, wsHeaders, 8192, 8192)
			if err != nil {
				rawConn.Close()
				return
			}
			connChan <- conn
			return
		}(wsurl)
	}
	connGroup.Wait()
	close(connChan)
	connTotal := len(connChan)
	var counter uint64
	for conn := range connChan {
		wg.Add(1)
		listen_wg.Add(1)
		go func(conn *websocket.Conn) {
			subStatus := false
			for {
				_, d, err := conn.ReadMessage()
				if err != nil {
					if c.Bool("debug") {
						log.Error(err)
					}
					if !subStatus {
						listen_wg.Done()
					}
					wg.Done()
					atomic.AddUint64(&counter, 1)
					return
				}

				if string(d) == sub_resp {
					subStatus = true
					listen_wg.Done()
				}
				if c.Bool("debug") {
					log.Println("slave repsonse message", string(d))
				}
				data, _ := jsonparser.GetString(d, "data")
				if data == push_msg {
					wg.Done()
					return
				}
			}
		}(conn)
		err = conn.WriteMessage(websocket.TextMessage, []byte(sub_msg))
		if err != nil {
			conn.Close()
			continue
		}
	}

	listen_wg.Wait()
	log.Infof("%v connect finish", connTotal)
	//push start
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
	var pushStart time.Time
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
		if c.Bool("debug") {
			log.Println("master response", string(b))
		}
		pushStart = time.Now()
		return
	})
	log.Println("Waiting...")
	wg.Wait()
	t := time.Now().Sub(pushStart)
	if connTotal == 0 {
		log.Error("0 client connect, please check slave server!")
	} else if connTotal == int(counter) {
		log.Error("no client read message, please check master server!")
	} else {

		log.Infof("%v client connect, %v error read , receive msg time:%s", connTotal, counter, t)
	}

	return
}

func main() {
	cli.AppHelpTemplate += "\nWEBSITE:\n\t\thttps://github.com/syhlion/gusher.cluster/tree/master/test/conn-test\n\n"
	gusher := cli.NewApp()
	gusher.Usage = "simple connection test for gusher.cluster"
	gusher.Name = name
	gusher.Author = "Scott (syhlion)"
	gusher.Version = version
	gusher.Compiled = time.Now()
	gusher.Commands = []cli.Command{
		cmdStart,
	}
	gusher.Run(os.Args)
}
