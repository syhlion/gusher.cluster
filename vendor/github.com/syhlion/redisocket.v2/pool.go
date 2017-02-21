package redisocket

type eventPayload struct {
	payload *Payload
	event   string
}

type Pool struct {
	users   map[User]bool
	trigger chan *eventPayload
	join    chan User
	leave   chan User
}

func (h *Pool) Run() {
	for {
		select {
		case p := <-h.trigger:
			for u, _ := range h.users {
				u.Trigger(p.event, p.payload)
			}

		case u := <-h.join:
			h.users[u] = true

		case u := <-h.leave:
			if _, ok := h.users[u]; ok {
				u.Close()
				delete(h.users, u)
			}
		}

	}
}
func (a *Pool) Trigger(event string, p *Payload) {
	a.trigger <- &eventPayload{p, event}
}
func (a *Pool) Join(c User) {
	a.join <- c
}
func (a *Pool) Leave(c User) {
	a.leave <- c
}
