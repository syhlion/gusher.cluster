package main

import (
	"crypto/rsa"
	"errors"
	"net"

	jwt "github.com/golang-jwt/jwt"
	"github.com/gomodule/redigo/redis"
)

func InArray(c string, s []string) (b bool) {
	for _, v := range s {
		if c == v {
			return true
		}
	}
	return false
}

func RedisTestConn(conn redis.Conn) (err error) {
	_, err = conn.Do("PING")
	conn.Close()
	return
}

func JsonCheck(data string) (j interface{}) {
	err := json.Unmarshal([]byte(data), &j)
	if err != nil {
		j = data
	}
	return
}

func Decode(key *rsa.PublicKey, data string) (jp JwtPack, err error) {

	_, err = jwt.ParseWithClaims(data, &jp, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("token parse error")
		}
		return key, nil
	})
	if err != nil {
		logger.Debugf("data: %s , err: %v", data, err)
		return
	}
	return
}

func GetExternalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}
