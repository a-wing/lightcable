package lightcable

import (
	"context"
	"math/rand"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func makeWsProto(s string) string {
	return "ws" + strings.TrimPrefix(s, "http")
}

func TestNoWebSocket(t *testing.T) {
	server := New(DefaultConfig)
	ctx, cancel := context.WithCancel(context.Background())
	sign := make(chan bool)
	server.OnServClose(func() {
		sign <- true
	})
	go server.Run(ctx)
	httpServer := httptest.NewServer(server)
	client := httpServer.Client()
	if res, err := client.Get(httpServer.URL + "/test"); err != nil {
		t.Errorf("Should websocket upgrader error: %+v, %s\n", res, err)
	}
	cancel()
	<-sign
}

func makeConns(t testing.TB, server *Server, rooms ...string) []*websocket.Conn {
	httpServer := httptest.NewServer(server)
	conns := make([]*websocket.Conn, len(rooms))
	var err error
	for i, room := range rooms {
		if conns[i], _, err = websocket.DefaultDialer.Dial(makeWsProto(httpServer.URL+room), nil); err != nil {
			t.Error(err)
		}
	}
	return conns
}

func TestServer(t *testing.T) {
	server := New(DefaultConfig)
	conns := makeConns(t, server, "/test", "/test")
	ws, ws2 := conns[0], conns[1]

	ctx, cancel := context.WithCancel(context.Background())
	sign := make(chan bool)
	server.OnServClose(func() {
		sign <- true
	})
	join := make(chan string)
	server.OnConnReady(func(c *Client) {
		join <- c.Name
	})
	go server.Run(ctx)

	// Need wait for connection ready
	<-join
	<-join

	for i := 0; i < 10; i++ {
		data := make([]byte, 4096)
		n, err := rand.Read(data)
		if err != nil {
			t.Error(err)
		}
		if err := ws.WriteMessage(websocket.TextMessage, data[:n]); err != nil {
			t.Error(err)
		}

		code, recv, err := ws2.ReadMessage()
		if err != nil {
			t.Error(err)
		}

		if code != websocket.TextMessage {
			t.Error("Type should TextMessage")
		}

		if string(recv) != string(data[:n]) {
			t.Error("Data should Equal")
		}
	}

	if err := ws.SetReadDeadline(time.Now().Add(time.Millisecond)); err != nil {
		t.Error(err)
	}

	if _, _, err := ws.ReadMessage(); err == nil {
		t.Error("Should have error")
	}

	cancel()
	<-sign
}

func TestServerCallback(t *testing.T) {
	server := New(DefaultConfig)
	conns := makeConns(t, server, "/test", "/test-2")
	ws := conns[0]

	signServ := make(chan bool, 4)
	signRoom := make(chan bool, 4)
	signConn := make(chan bool, 4)
	signMsg := make(chan bool)

	server.OnRoomReady(func(room string) { signRoom <- true })
	server.OnConnReady(func(c *Client) { signConn <- true })
	server.OnMessage(func(m *Message) { signMsg <- true })
	server.OnConnClose(func(c *Client) { signConn <- true })
	server.OnRoomClose(func(room string) { signRoom <- true })
	server.OnServClose(func() { signServ <- true })

	ctx, cancel := context.WithCancel(context.Background())
	go server.Run(ctx)
	data := make([]byte, 4096)
	n, err := rand.Read(data)
	if err != nil {
		t.Error(err)
	}
	if err := ws.WriteMessage(websocket.TextMessage, data[:n]); err != nil {
		t.Error(err)
	}
	if err := ws.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
		t.Error(err)
	}

	<-signRoom
	<-signConn
	<-signMsg
	<-signConn
	<-signRoom

	cancel()
	<-signServ
}

func TestServerLocal(t *testing.T) {
	config := DefaultConfig
	config.Worker.Local = true
	server := New(DefaultConfig)
	conns := makeConns(t, server, "/test")
	ws := conns[0]

	ctx, cancel := context.WithCancel(context.Background())
	sign := make(chan bool)
	server.OnServClose(func() {
		sign <- true
	})
	join := make(chan string)
	server.OnConnReady(func(c *Client) {
		join <- c.Name
	})
	go server.Run(ctx)

	// Need wait for connection ready
	<-join

	data := []byte("xxx")
	ws.WriteMessage(websocket.BinaryMessage, data)
	code, data2, err := ws.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if code != websocket.BinaryMessage {
		t.Error("websocket data type:", code)
	}
	if string(data) != string(data2) {
		t.Errorf("ReadMessage is: %s", data2)
	}

	cancel()
	<-sign
}

func TestServerBroadcast(t *testing.T) {
	server := New(DefaultConfig)
	conns := makeConns(t, server, "/test", "/test", "/test-2")
	ws, ws2, ws3 := conns[0], conns[1], conns[2]

	ctx, cancel := context.WithCancel(context.Background())
	sign := make(chan bool)
	server.OnServClose(func() {
		sign <- true
	})
	join := make(chan string)
	server.OnConnReady(func(c *Client) {
		join <- c.Name
	})
	go server.Run(ctx)

	// Need wait for connection ready
	<-join
	<-join
	<-join

	for i := 0; i < 10; i++ {
		data := make([]byte, 4096)
		n, err := rand.Read(data)
		if err != nil {
			t.Error(err)
		}
		server.Broadcast("/test", "test", websocket.TextMessage, data[:n])

		// ws
		if code, recv, err := ws.ReadMessage(); err == nil {
			if code != websocket.TextMessage {
				t.Error("Type should TextMessage")
			}

			if string(recv) != string(data[:n]) {
				t.Error("Data should Equal")
			}
		} else {
			t.Error(err)
		}

		// ws2
		if code, recv, err := ws2.ReadMessage(); err == nil {
			if code != websocket.TextMessage {
				t.Error("Type should TextMessage")
			}

			if string(recv) != string(data[:n]) {
				t.Error("Data should Equal")
			}
		} else {
			t.Error(err)
		}
	}

	if err := ws3.SetReadDeadline(time.Now().Add(time.Millisecond)); err != nil {
		t.Error(err)
	}

	if _, _, err := ws3.ReadMessage(); err == nil {
		t.Error("Should have error")
	}

	cancel()
	<-sign
}

func TestServerVeryMuchRoom(t *testing.T) {
	server := New(DefaultConfig)
	ctx, cancel := context.WithCancel(context.Background())
	sign := make(chan bool)
	server.OnServClose(func() {
		sign <- true
	})
	join := make(chan string)
	server.OnConnReady(func(c *Client) {
		join <- c.Room
	})
	go server.Run(ctx)

	httpServer := httptest.NewServer(server)

	for i := 0; i < 128; i++ {
		data := make([]byte, 4096)
		cable := "/test-" + strconv.Itoa(i)
		ws, _, err := websocket.DefaultDialer.DialContext(ctx, makeWsProto(httpServer.URL+cable), nil)
		if err != nil {
			t.Error(err)
		}

		if room := <-join; cable != room {
			t.Errorf("Not Join successfully: %s, %s", cable, room)
		}

		for i := 0; i < 10; i++ {
			n, err := rand.Read(data)
			if err != nil {
				t.Error(err)
			}
			if err := ws.WriteMessage(websocket.TextMessage, data[:n]); err != nil {
				t.Error(err)
			}
		}
	}
	cancel()
	<-sign
}
