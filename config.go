package main

import "time"

type MasterConfig struct {
	RedisAddr         string
	RedisMaxIdle      int
	RedisMaxConn      int
	ApiListen         string
	ApiPrefix         string
	PublicKeyLocation string
	Version           string
	CompileDate       string
	ExternalIp        string
	StartTime         time.Time
}

func (m MasterConfig) GetStartTime() string {
	return m.StartTime.Format("2006/01/02 15:04:05")
}

type SlaveConfig struct {
	LogInterval       time.Duration
	ScanInterval      time.Duration
	MaxMessage        int
	RedisAddr         string
	RedisMaxIdle      int
	RedisMaxConn      int
	ApiListen         string
	ApiPrefix         string
	DecodeServiceAddr string
	Version           string
	CompileDate       string
	ExternalIp        string
	ReadBuffer        int
	WriteBuffer       int
	StartTime         time.Time
}

func (s SlaveConfig) GetStartTime() string {
	return s.StartTime.Format("2006/01/02 15:04:05")
}
