package lightcable

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
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
	// TODO:
	// onConnClose(*Client, error)
	// OnMessage(*message)
	//onConnect func(*Client)
	// hook onConnect
	// hook disconnected
	// hook ommessage
	// func send message
	onMessage func(*Message)

	onServClose func()
	onRoomClose func(room string)
	onConnClose func(*Client)
}

type Config struct {
	OnMessage func(*Message)
	// Safe Close
	OnServClose func()
	OnRoomClose func(room string)
	OnConnClose func(*Client)
}

// TODO: maybe need
// func (s *Server) GetStats() {}

func NewServer() *Server {
	return &Server{
		topic: make(map[string]*topic),

		register:   make(chan *Client),
		broadcast:  make(chan []byte),
		unregister: make(chan *Client),

		onMessage:   func(*Message) {},
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
			delete(s.topic, c.Room)

			// Last room, server onClose
			if len(s.topic) == 0 && s.readyState == readyStateClosing {
				s.onServClose()
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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, err.Error())
	}
	token := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", &conn)))
	s.JoinCable(r.URL.Path, token, conn)
}

func (s *Server) JoinCable(room, name string, conn *websocket.Conn) error {
	select {
	case s.register <- &Client{
		Room: room,
		Name: name,
		conn: conn,
		send: make(chan Message, 256),
	}:
		return nil
	default:
		return errors.New("join failure")
	}
}

// All room broadcast
func (s *Server) AllBroadcast(typ int, data []byte) {
	for _, topic := range s.topic {
		topic.broadcast <- Message{
			//Name: ,
			//Room: ,
			Code: typ,
			Data: data,
			//conn: ,
		}
	}
}

// room broadcast
func (s *Server) Broadcast(room string, typ int, data []byte) {
	if topic, ok := s.topic[room]; ok {
		topic.broadcast <- Message{
			//Name: ,
			//Room: ,
			Code: typ,
			Data: data,
			//conn: ,
		}
	}
}
