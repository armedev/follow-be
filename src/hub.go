package main

type BroadcastMessage struct {
	message []byte
	client  *Client
}

const (
	followMe   string = "FOLLOWME"
	unFollowMe string = "UNFOLLOWME"
)

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan BroadcastMessage

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// history of events for a leader
	history []*MessageBuilder

	// history channel to get message and process it async
	historyChan chan *MessageBuilder
}

func newHub() *Hub {
	return &Hub{
		broadcast:   make(chan BroadcastMessage),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		clients:     make(map[*Client]bool),
		history:     []*MessageBuilder{},
		historyChan: make(chan *MessageBuilder),
	}
}

// make all the connected clients consumers
func (h *Hub) makeAllClientsConsumers() {
	for client := range h.clients {
		makeConsumer(client)
	}
}

// make all the connected clients consumers except passed client
func (h *Hub) makeAllClientsConsumersExcept(c *Client) {
	for client := range h.clients {
		if client == c {
			makeLeader(c)
		} else {
			makeConsumer(client)
		}
	}
}

// loops over clients and calls the cb on each client and breaks if cb returns false
func (h *Hub) loopOverClients(cb func(*Client) bool) {
	for client := range h.clients {
		loopContinue := cb(client)
		if !loopContinue {
			break
		}
	}
}

func (h *Hub) leaderPresent() bool {
	var isLeaderPresent bool

	cb := func(client *Client) bool {
		if client.leader {
			isLeaderPresent = true
		}
		return !client.leader
	}

	h.loopOverClients(cb)

	return isLeaderPresent
}

func makeLeader(c *Client) {
	c.leader = true
}

func makeConsumer(c *Client) {
	c.leader = false
}

func (h *Hub) handleHistory() {
	for {
		select {
		case msg := <-h.historyChan:
			{
				// TODO: Logic for avoiding repeated messages

				h.history = append(h.history, msg)
			}
		}
	}
}

func (h *Hub) run() {
	for {
		select {

		case client := <-h.register:
			{
				h.clients[client] = true
				if h.leaderPresent() {
					for _, msg := range h.history {
						select {
						case client.send <- msg.format():
						default:
							close(client.send)
							delete(h.clients, client)
						}
					}
				}
			}

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				if client.leader {
					h.history = []*MessageBuilder{}
					messageBuilder := MessageBuilder{data: "UNFOLLOWME"}
					for c := range h.clients {
						select {
						case c.send <- messageBuilder.format():
						default:
							close(c.send)
							delete(h.clients, c)
						}
					}

				}
				close(client.send)
			}

		case received := <-h.broadcast:
			{
				msg := string(received.message)
				if msg == followMe || msg == txtPlaceholder+followMe {
					h.makeAllClientsConsumersExcept(received.client)
					h.history = []*MessageBuilder{}
				} else if msg == unFollowMe || msg == txtPlaceholder+unFollowMe {
					makeConsumer(received.client)
					h.history = []*MessageBuilder{}
				} else if !received.client.leader {
					continue
				}

				messageBuilder := MessageBuilder{data: msg}

				h.historyChan <- &messageBuilder

				for client := range h.clients {
					if client == received.client {
						continue
					}
					select {
					case client.send <- messageBuilder.format():
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
		}
	}
}
