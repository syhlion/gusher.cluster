package main

import (
	"crypto/rsa"
	"errors"
	"sync"

	jwt "github.com/dgrijalva/jwt-go"
)

type JWT_RSA_Decoder struct {
	key *rsa.PublicKey
}

func (j *JWT_RSA_Decoder) Decode(data []byte, auth *Auth) (err error) {
	_, err = jwt.ParseWithClaims(string(data), auth, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("token parse error")
		}
		return j.key, nil
	})
	if err != nil {
		logger.Debugf("data: %s , err: %v", data, err)
		return
	}
	return
}

type HealthTrack struct {
	s *SlaveInfos
}

func (h *HealthTrack) Put(info *ServerInfo, ok *bool) error {
	i := *info
	h.s.Update(i)
	return nil
}

type SlaveInfos struct {
	servers map[string]ServerInfo
	lock    *sync.Mutex
}

func (s *SlaveInfos) Update(si ServerInfo) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.servers[si.Ip+"@"+si.LocalListen] = si
	logger.Debugf("server info update %s", s.servers)
	return
}
func (s *SlaveInfos) Info() map[string]ServerInfo {
	return s.servers
}
