package main

import (
	"log"
	"net/http"

	"github.com/garyburd/redigo/redis"
	"github.com/syhlion/redisocket.v2"
)

func TestEvent(event string, d []byte) (data []byte, err error) {
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

		client, err := app.Upgrade(w, r, nil)
		if err != nil {
			log.Fatal("Client Connect Error")
			return
		}
		err = client.On("Test", TestEvent)
		if err != nil {
			return
		}
		err = client.Listen(func(data []byte) (err error) {
			app.Publish("Test", []byte("Hello"))
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
