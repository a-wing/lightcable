package lightcable

import (
	"context"
)

type worker struct {
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

func newWorker(room string, server *Server) *worker {
	return &worker{
		room:   room,
		server: server,

		clients:    make(map[*Client]bool),
		register:   make(chan *Client, server.config.SignBufferCount),
		broadcast:  make(chan Message, server.config.CastBufferCount),
		unregister: make(chan *Client, server.config.SignBufferCount),
	}
}

func (w *worker) run(ctx context.Context) {
	defer w.server.OnRoomClose(w.room)
	for {
		select {
		case client := <-w.register:
			w.clients[client] = true

			go client.readPump()
			go client.writePump(ctx)

			w.server.OnConnReady(client)
		case client := <-w.unregister:
			if _, ok := w.clients[client]; ok {
				delete(w.clients, client)
				close(client.send)
			}
			w.server.OnConnClose(client)

			// Last client, need close this room
			if len(w.clients) == 0 {
				w.server.unregister <- client
				w.server.OnRoomClose(w.room)
				return
			}
		case message := <-w.broadcast:
			w.server.OnMessage(&message)
			for client := range w.clients {
				if message.conn != client.conn {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(w.clients, client)
					}
				}
			}
		case <-ctx.Done():
			// safe Close
		}
	}
}
