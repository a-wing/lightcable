package lightcable

import (
	"context"
)

type topic struct {
	room   string
	server *Server

	// Registered clients.
	clients map[*Client]bool

	// Register requests from the clients.
	register chan *Client

	// Inbound messages from the clients.
	broadcast chan Message

	// Unregister requests from clients.
	unregister chan *Client
}

func NewTopic(room string, server *Server) *topic {
	return &topic{
		room:   room,
		server: server,

		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		broadcast:  make(chan Message),
		unregister: make(chan *Client),
	}
}

func (t *topic) run(ctx context.Context) {
	defer t.server.onRoomClose(t.room)
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
			t.server.onConnClose(client)

			// Last client, need close this room
			if len(t.clients) == 0 {
				t.server.unregister <- client
				t.server.onRoomClose(t.room)
				return
			}
		case message := <-t.broadcast:
			t.server.onMessage(&message)
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
