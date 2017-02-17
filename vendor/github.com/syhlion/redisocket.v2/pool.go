package redisocket

type registerPayload struct {
	event string
	user  User
}
type unregisterPayload struct {
	event string
	user  User
}
type unregisterAllPayload struct {
	user User
}

type eventPayload struct {
	payload *Payload
	event   string
}

type Pool struct {
	subjects    map[string]map[User]bool
	subscribers map[User]map[string]bool
	trigger     chan *eventPayload
	reg         chan *registerPayload
	unreg       chan *unregisterPayload
	unregAll    chan *unregisterAllPayload
	close       chan int
}

func (h *Pool) Stop() {
	h.close <- 1
}

func (h *Pool) Run() {
	for {
		select {
		case <-h.close:
			for u, _ := range h.subscribers {
				u.Close()
			}
			return
		case p := <-h.trigger:
			users, ok := h.subjects[p.event]
			if !ok {
				continue
			}
			for u, _ := range users {
				u.Trigger(p.event, p.payload)
			}

		case p := <-h.reg:
			h.register(p.event, p.user)
		case p := <-h.unreg:
			h.unregister(p.event, p.user)
		case p := <-h.unregAll:
			h.unregisterAll(p.user)
		}

	}
}
func (a *Pool) Trigger(event string, p *Payload) {
	a.trigger <- &eventPayload{p, event}
}
func (a *Pool) UnregisterAll(c User) {
	a.unregAll <- &unregisterAllPayload{c}
}
func (a *Pool) Unregister(event string, c User) {
	a.unreg <- &unregisterPayload{event, c}
}
func (a *Pool) Register(event string, c User) {
	a.reg <- &registerPayload{event, c}
}
func (a *Pool) register(event string, c User) {
	//observer map
	if m, ok := a.subscribers[c]; !ok {
		events := make(map[string]bool)
		events[event] = true
		a.subscribers[c] = events
	} else {
		m[event] = true
	}

	//event map
	if clients, ok := a.subjects[event]; !ok {
		clients := make(map[User]bool)
		clients[c] = true
		a.subjects[event] = clients
	} else {
		clients[c] = true
	}
	return
}
func (h *Pool) unregister(event string, c User) {

	//observer map
	if m, ok := h.subscribers[c]; ok {
		delete(m, event)
		if len(m) == 0 {
			delete(h.subscribers, c)
		}
	}
	//event map
	if m, ok := h.subjects[event]; ok {
		delete(m, c)
		if len(m) == 0 {
			delete(h.subjects, event)
		}
	}

	return
}

func (a *Pool) unregisterAll(c User) {
	m, ok := a.subscribers[c]
	if ok {
		for e, _ := range m {
			a.unregister(e, c)
		}
	}
	delete(a.subscribers, c)
	return
}
