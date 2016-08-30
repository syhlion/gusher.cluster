package main

import "sync"

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
	return
}
func (s *SlaveInfos) Info() map[string]ServerInfo {
	return s.servers
}
