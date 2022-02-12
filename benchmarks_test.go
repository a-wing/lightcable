package lightcable

import (
	"context"
	"math/rand"
	"testing"

	"github.com/gorilla/websocket"
)

func BenchmarkBroadcast(b *testing.B) {
	server := New(DefaultConfig)
	conns := makeConns(b, server, "/test", "/test")
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

	data := make([]byte, 4096)
	n, err := rand.Read(data)
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < b.N; i++ {
		if err := ws.WriteMessage(websocket.TextMessage, data[:n]); err != nil {
			b.Error(err)
		}

		if code, recv, err := ws2.ReadMessage(); err == nil {
			if code != websocket.TextMessage {
				b.Error("Type should TextMessage")
			}

			if string(recv) != string(data[:n]) {
				b.Error("Data should Equal")
			}
		} else {
			b.Error(err)
		}
	}

	cancel()
	<-sign
}
