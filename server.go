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
	topic map[string]*topic

	// Register requests from the clients.
	register chan *Client

	// Inbound messages from the clients.
	broadcast chan []byte

	// Unregister requests from clients.
	unregister chan *Client

	readyState

	OnMessage func(*Message)
	OnConnected func(w http.ResponseWriter, r *http.Request) (room, name string, ok bool)
	OnServClose func()
	OnRoomClose func(room string)
	OnConnClose func(*Client)
}

type Config struct {
	OnMessage func(*Message)
	// Safe Close
	OnServClose func()
	OnRoomClose func(room string)
	OnConnClose func(*Client)
}

func NewServer() *Server {
	return &Server{
		topic: make(map[string]*topic),

		register:   make(chan *Client),
		broadcast:  make(chan []byte),
		unregister: make(chan *Client),

		OnMessage:   func(*Message) {},
		OnConnected: func(w http.ResponseWriter, r *http.Request) (room, name string, ok bool) {
			return r.URL.Path, getUniqueID(), true
		},
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
			delete(s.topic, c.Room)

			// Last room, server onClose
			if len(s.topic) == 0 && s.readyState == readyStateClosing {
				s.OnServClose()
				s.readyState = readyStateClosed
				return
			}
		case c := <-s.register:
			c.topic = s.topic[c.Room]
			if c.topic == nil {
				c.topic = NewTopic(c.Room, s)
				go c.topic.run(ctx)
				s.topic[c.Room] = c.topic
			}
			c.topic.register <- c

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
