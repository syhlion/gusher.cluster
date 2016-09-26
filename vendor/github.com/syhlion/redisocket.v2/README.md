# redisocket.v2

Base on gorilla/websocket & garyburd/redigo

Implement By Observer pattern

## Documention

* [API Reference](https://godoc.org/github.com/syhlion/redisocket.v2)
* [Simple Example](https://github.com/syhlion/redisocket.v2/blob/master/example/main.go)

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
	app := redisocket.NewHub(pool)

	err := make(chan error)
	go func() {
		err <- app.Listen()
	}()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {

		sub, err := app.NewClient(w, r)
		if err != nil {
			log.Fatal("Client Connect Error")
			return
		}
		err = sub.Subscribe("Test", TestEvent)
		if err != nil {
			return
		}
		err = sub.Listen(func(data []byte) (err error) {
			app.Notify("Test", []byte("Hello"))
			return

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
