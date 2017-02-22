package redisocket

type eventPayload struct {
	payload *Payload
	event   string
}

type Pool struct {
	users     map[*Client]bool
	broadcast chan *eventPayload
	join      chan *Client
	leave     chan *Client
}

func (h *Pool) Run() {
	for {
		select {
		case p := <-h.broadcast:
			for u, _ := range h.users {
				u.Trigger(p.event, p.payload)
			}

		case u := <-h.join:
			h.users[u] = true

		case u := <-h.leave:
			if _, ok := h.users[u]; ok {
				close(u.send)
				delete(h.users, u)
			}
		}

	}
}
func (a *Pool) Broadcast(event string, p *Payload) {
	a.broadcast <- &eventPayload{p, event}
}
func (a *Pool) Join(c *Client) {
	a.join <- c
}
func (a *Pool) Leave(c *Client) {
	a.leave <- c
}
