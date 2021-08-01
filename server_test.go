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

func TestServer(t *testing.T) {
	server := NewServer()
	ctx, cancel := context.WithCancel(context.Background())
	sign := make(chan bool)
	server.OnServClose = func() {
		sign <- true
	}
	go server.Run(ctx)

	httpServer := httptest.NewServer(server)

	cable := "/test"
	ws, _, err := websocket.DefaultDialer.DialContext(ctx, makeWsProto(httpServer.URL+cable), nil)
	if err != nil {
		t.Error(err)
	}

	ws2, _, err := websocket.DefaultDialer.DialContext(ctx, makeWsProto(httpServer.URL+cable), nil)
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 10; i++ {
		data := make([]byte, 4096)
		n, err := rand.Read(data)
		if err != nil {
			t.Error(err)
		}
		ws.WriteMessage(websocket.TextMessage, data[:n])

		typ, recv, err := ws2.ReadMessage()
		if err != nil {
			t.Error(err)
		}

		if typ != websocket.TextMessage {
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

func TestServerBroadcast(t *testing.T) {
	server := NewServer()
	ctx, cancel := context.WithCancel(context.Background())
	sign := make(chan bool)
	server.OnServClose = func() {
		sign <- true
	}
	go server.Run(ctx)

	httpServer := httptest.NewServer(server)

	cable := "/test"
	ws, _, err := websocket.DefaultDialer.DialContext(ctx, makeWsProto(httpServer.URL+cable), nil)
	if err != nil {
		t.Error(err)
	}

	ws2, _, err := websocket.DefaultDialer.DialContext(ctx, makeWsProto(httpServer.URL+cable), nil)
	if err != nil {
		t.Error(err)
	}

	ws3, _, err := websocket.DefaultDialer.DialContext(ctx, makeWsProto(httpServer.URL+cable+"2"), nil)
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 10; i++ {
		data := make([]byte, 4096)
		n, err := rand.Read(data)
		if err != nil {
			t.Error(err)
		}
		server.Broadcast(cable, "test", websocket.TextMessage, data[:n])

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
	server := NewServer()
	ctx, cancel := context.WithCancel(context.Background())
	sign := make(chan bool)
	server.OnServClose = func() {
		sign <- true
	}
	go server.Run(ctx)

	httpServer := httptest.NewServer(server)

	for i := 0; i < 4096; i++ {
		data := make([]byte, 4096)
		cable := "/test-" + strconv.Itoa(i)
		ws, _, err := websocket.DefaultDialer.DialContext(ctx, makeWsProto(httpServer.URL+cable), nil)
		if err != nil {
			t.Error(err)
		}

		for i := 0; i < 10; i++ {
			n, err := rand.Read(data)
			if err != nil {
				t.Error(err)
			}
			ws.WriteMessage(websocket.TextMessage, data[:n])
		}
	}
	cancel()
	<-sign
}
