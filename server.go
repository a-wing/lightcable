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

// Server is lightcable core Server. use callback notification message
// broadcast message, A Server auto create and manage multiple goroutines
// every room create worker
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
	onRoomReady func(room string)
	onConnReady func(*Client)
	onConnClose func(*Client, error)
	onRoomClose func(room string)
	onServClose func()
}

// New creates a new Server.
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
		onRoomReady: func(room string) {},
		onConnReady: func(*Client) {},
		onConnClose: func(*Client, error) {},
		onRoomClose: func(room string) {},
		onServClose: func() {},
	}
}

// Run need use 'go server.Run(context.Background())' run daemon
// in order to concurrency. server instance only a run
func (s *Server) Run(ctx context.Context) {
	s.readyState = readyStateRunning
	defer func() {
		for {
			// Last room, server onClose
			if len(s.worker) == 0 && s.readyState == readyStateClosing {
				s.onServClose()
				s.readyState = readyStateClosed
				return
			}

			select {
			case c := <-s.unregister:
				delete(s.worker, c.Room)
			}
		}
	}()
	for {
		select {
		// unregister must first
		// close and open concurrency
		case c := <-s.unregister:
			delete(s.worker, c.Room)
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
			return
		}
	}
}

// ServeHTTP Interface 'http.Handler'.
// creates new websocket connection
// Maybe Create new Worker. worker == room
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

// Broadcast will room all websocket connection send message
// https://www.rfc-editor.org/rfc/rfc6455.html#section-11.8
// code is websocket Opcode
// name is custom name, this will be callback OnMessage
func (s *Server) Broadcast(room, name string, code int, data []byte) {
	s.broadcast <- Message{
		Name: name,
		Room: room,
		Code: code,
		Data: data,
	}
}

// OnMessage will all Websocket Conn Recv Message will callback this function
// This have Block worker. Block this room
func (s *Server) OnMessage(fn func(*Message)) {
	s.onMessage = fn
}

// OnConnected auth this websocket connection callback
// ok: true Allows connection; false Reject connection
// Maybe Concurrent. unique ID need self use sync.Mutex
func (s *Server) OnConnected(fn func(w http.ResponseWriter, r *http.Request) (room, name string, ok bool)) {
	s.onConnected = fn
}

// OnRoomReady Create a new room successfully
func (s *Server) OnRoomReady(fn func(room string)) {
	s.onRoomReady = fn
}

// OnConnReady websocket connection successfully and join room
// this will block worker
func (s *Server) OnConnReady(fn func(*Client)) {
	s.onConnReady = fn
}

// OnConnClose will Client error or websocket close or server close
// if context server closed err == nil
func (s *Server) OnConnClose(fn func(*Client, error)) {
	s.onConnClose = fn
}

// OnRoomClose worker all websocket connection closed, worker close
func (s *Server) OnRoomClose(fn func(room string)) {
	s.onRoomClose = fn
}

// OnServClose server safely shutdown done callback
func (s *Server) OnServClose(fn func()) {
	s.onServClose = fn
}
