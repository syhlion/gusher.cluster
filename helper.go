package main

import (
	"crypto/rsa"
	"errors"
	"net"
	"runtime"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

func Decode(key *rsa.PublicKey, data string) (auth Auth, err error) {

	_, err = jwt.ParseWithClaims(data, &auth, func(token *jwt.Token) (interface{}, error) {
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

func GetInfo(ip string, c int) (s ServerInfo) {
	m := new(runtime.MemStats)
	runtime.ReadMemStats(m)
	s = ServerInfo{

		Ip:             ip,
		LocalListen:    api_listen,
		Version:        version,
		RunTimeVersion: runtime.Version(),
		NumCpu:         runtime.NumCPU(),
		MemAllcoated:   m.Alloc,
		Goroutines:     runtime.NumGoroutine(),
		UpdateTime:     time.Now().Unix(),
		SendInterval:   return_serverinfo_interval,
		Connections:    c,
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
