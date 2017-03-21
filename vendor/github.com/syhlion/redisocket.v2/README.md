# redisocket.v2

Base on gorilla/websocket & garyburd/redigo

Implement By Observer pattern

## Documention

* [API Reference](https://godoc.org/github.com/syhlion/redisocket.v2)

## Install

`go get github.com/syhlion/redisocket.v2`

## Usaged

``` go
func TestEvent(d []byte) (data []byte, err error) {
	return d, nil
}

func main() {
	pool := redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", ":6379")
	}, 10)
	app := redisocket.NewHub(pool,false)

	err := make(chan error)
	go func() {
		err <- app.Listen()
	}()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {

        client,err:= app.Upgrade(w, r, nil, "Scott", "appKey")
		if err != nil {
			log.Fatal("Client Connect Error")
			return
		}
		err = client.Listen(func(data []byte) (msg *redisocket.ReceiveMsg, err error) {
		    msg = &redisocket.ReceiveMsg{}
			msg.Sub = true
			msg.Event = "Test"
			msg.ResponseMsg = []byte("welcome")
			return msg,nil

		})
		log.Println(err, "http point")
		return
	})

	go func() {
		err <- http.ListenAndServe(":8888", nil)
	}()
	select {
	case e := <-err:
		log.Println(e)
	}
}
```
