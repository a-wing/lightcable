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

	onMessage   func(*Message)
	onConnected func(w http.ResponseWriter, r *http.Request) (room, name string, ok bool)
	onConnReady func(*Client)
	onServClose func()
	onRoomClose func(room string)
	onConnClose func(*Client)
}

func New(cfg *Config) *Server {
	return &Server{
		config: cfg.Worker,
		worker: make(map[string]*worker),

		register:   make(chan *Client, cfg.SignBufferCount),
		broadcast:  make(chan Message, cfg.CastBufferCount),
		unregister: make(chan *Client, cfg.SignBufferCount),

		onMessage: func(*Message) {},
		onConnected: func(w http.ResponseWriter, r *http.Request) (room, name string, ok bool) {
			return r.URL.Path, getUniqueID(), true
		},
		onConnReady: func(*Client) {},
		onServClose: func() {},
		onRoomClose: func(room string) {},
		onConnClose: func(*Client) {},
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
				s.onServClose()
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
	if room, name, ok := s.onConnected(w, r); ok {
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

func (s *Server) OnMessage(fn func(*Message)) {
	s.onMessage = fn
}

func (s *Server) OnConnected(fn func(w http.ResponseWriter, r *http.Request) (room, name string, ok bool)) {
	s.onConnected = fn
}

func (s *Server) OnConnReady(fn func(*Client)) {
	s.onConnReady = fn
}

func (s *Server) OnServClose(fn func()) {
	s.onServClose = fn
}

func (s *Server) OnRoomClose(fn func(room string)) {
	s.onRoomClose = fn
}

func (s *Server) OnConnClose(fn func(*Client)) {
	s.onConnClose = fn
}
