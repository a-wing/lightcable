package lightcable

import (
	"context"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
)

type readyState int8

const (
	readyStateOpening readyState = iota
	readyStateRunning
	readyStateClosing
	readyStateClosed
)

type Server struct {
	config *Config
	worker map[string]*worker

	// Register requests from the clients.
	register chan *Client

	// Inbound messages from the clients.
	broadcast chan Message

	// Unregister requests from clients.
	unregister chan *Client

	readyState

	OnMessage   func(*Message)
	OnConnected func(w http.ResponseWriter, r *http.Request) (room, name string, ok bool)
	OnConnReady func(*Client)
	OnServClose func()
	OnRoomClose func(room string)
	OnConnClose func(*Client)
}

func New(cfg *Config) *Server {
	return &Server{
		config: cfg.Worker,
		worker: make(map[string]*worker),

		register:   make(chan *Client, cfg.SignBufferCount),
		broadcast:  make(chan Message, cfg.CastBufferCount),
		unregister: make(chan *Client, cfg.SignBufferCount),

		OnMessage: func(*Message) {},
		OnConnected: func(w http.ResponseWriter, r *http.Request) (room, name string, ok bool) {
			return r.URL.Path, getUniqueID(), true
		},
		OnConnReady: func(*Client) {},
		OnServClose: func() {},
		OnRoomClose: func(room string) {},
		OnConnClose: func(*Client) {},
	}
}

func (s *Server) Run(ctx context.Context) {
	s.readyState = readyStateRunning
	for {
		select {
		// unregister must first
		// close and open concurrency
		case c := <-s.unregister:
			delete(s.worker, c.Room)

			// Last room, server onClose
			if len(s.worker) == 0 && s.readyState == readyStateClosing {
				s.OnServClose()
				s.readyState = readyStateClosed
				return
			}
		case c := <-s.register:
			c.worker = s.worker[c.Room]
			if c.worker == nil {
				c.worker = newWorker(c.Room, s)
				go c.worker.run(ctx)
				s.worker[c.Room] = c.worker
			}
			c.worker.register <- c
		case m := <-s.broadcast:
			if worker, ok := s.worker[m.Room]; ok {
				worker.broadcast <- m
			}
		case <-ctx.Done():
			s.readyState = readyStateClosing
		}
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if room, name, ok := s.OnConnected(w, r); ok {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, err.Error())
		}

		// The server lack of resources: close the connection
		select {
		case s.register <- &Client{
			Room: room,
			Name: name,
			conn: conn,
			send: make(chan Message, 256),
		}:
		default:
			conn.WriteMessage(websocket.CloseMessage, []byte{})
		}
	}
}

// https://www.rfc-editor.org/rfc/rfc6455.html#section-11.8
// code is websocket Opcode
func (s *Server) Broadcast(room, name string, code int, data []byte) {
	s.broadcast <- Message{
		Name: name,
		Room: room,
		Code: code,
		Data: data,
	}
}
