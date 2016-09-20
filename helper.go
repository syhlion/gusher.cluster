package main

import (
	"errors"
	"net"
	"runtime"
	"time"
)

func getInfo(ip string, c int) (s ServerInfo) {
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

func getExternalIP() (string, error) {
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
