package main

import (
	"flag"
	"log"
	"net"

	"raftkv/server"
	"raftkv/storage"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:6379", "TCP address to listen on")
	flag.Parse()

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	s := server.New(listener, storage.New())
	log.Printf("listening on %s", listener.Addr())
	if err := s.Serve(); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
