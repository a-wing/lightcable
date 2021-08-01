package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"lightcable"
)

func main() {
	address := flag.String("p", "0.0.0.0:8080", "set server port")
	help := flag.Bool("h", false, "this help")
	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	server := lightcable.New(lightcable.DefaultConfig)
	go server.Run(context.Background())

	log.Println("===============")
	log.Println("Listen Port", *address)
	log.Fatal(http.ListenAndServe(*address, server))
}
