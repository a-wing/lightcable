package lightcable

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Message represents a message send and received from the Websocket connection.
//
// Code is websocket Opcode
// Name is user custom Name
type Message struct {
	Room string
	Name string
	Code int
	Data []byte
	conn *websocket.Conn
}

// Client is a middleman between the websocket connection and the worker.
type Client struct {
	Name string
	Room string

	Err error

	worker *worker

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan Message
}

// readPump pumps messages from the websocket connection to the worker.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.worker.unregister <- c
		c.conn.Close()
	}()
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		c.Err = err
	}
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		code, data, err := c.conn.ReadMessage()
		if err != nil {
			c.Err = err
			break
		}
		c.worker.broadcast <- Message{
			Name: c.Name,
			Room: c.Room,
			Code: code,
			Data: data,
			conn: c.conn,
		}
	}
}

// writePump pumps messages from the worker to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.Err = err
			}
			if !ok {
				// The worker closed the channel.
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					c.Err = err
				}
				return
			}

			if err := c.conn.WriteMessage(msg.Code, msg.Data); err != nil {
				c.Err = err
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.Err = err
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.Err = err
				return
			}
		case <-ctx.Done():
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
	}
}
