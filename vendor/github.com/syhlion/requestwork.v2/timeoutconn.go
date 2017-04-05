package requestwork

import (
	"net"
	"time"
)

//Dial dial func
func Dial(network, addr string) (conn net.Conn, err error) {
	dial := net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return dial.Dial(network, addr)
}
