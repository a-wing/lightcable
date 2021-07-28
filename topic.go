package lightcable

import (
	"context"
)

type topic struct {
	name   string
	server *Server

	// Registered clients.
	clients map[*Client]bool

	// Register requests from the clients.
	register chan *Client

	// Inbound messages from the clients.
	broadcast chan message

	// Unregister requests from clients.
	unregister chan *Client
}

func NewTopic(name string, server *Server) *topic {
	return &topic{
		name:   name,
		server: server,

		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		broadcast:  make(chan message),
		unregister: make(chan *Client),
	}
}

func (t *topic) run(ctx context.Context) {
	for {
		select {
		case client := <-t.register:
			t.clients[client] = true

			go client.readPump()
			go client.writePump(ctx)

		case client := <-t.unregister:
			if _, ok := t.clients[client]; ok {
				delete(t.clients, client)
				close(client.send)
			}
			if len(t.clients) == 0 {
				t.server.unregister <- client
				return
			}
		case message := <-t.broadcast:
			for client := range t.clients {
				if message.conn != client.conn {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(t.clients, client)
					}
				}
			}
		case <-ctx.Done():
			// safe Close
		}
	}
}
