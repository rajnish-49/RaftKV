package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"raftkv/command"
	"raftkv/protocol"
	"raftkv/storage"
)

type Server struct {
	listener net.Listener
	store    *storage.Store
}

//A net.Listener is a socket waiting for clients to connect
//new creates a server using the listener and a shared store.
func New(listener net.Listener, store *storage.Store) *Server {
	return &Server{listener: listener, store: store}
}

// Serve accepts clients and handles each connection in its own goroutine.
// Closing the listener stops Serve with a nil error; existing connections
// continue until their handlers return. Serve closes the listener on return.
func (s *Server) Serve() error {
	defer s.listener.Close()

	for {
		conn, err := s.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("accept connection: %w", err)
		}

		go func(conn net.Conn) {
			if err := s.handleConn(conn); err != nil {
				log.Printf("connection %s: %v", conn.RemoteAddr(), err)
			}
		}(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) error {
	defer conn.Close()

	for {
		payload, err := protocol.ReadFrame(conn) // tcp has no message boundaries , returns 1 complete message
		if errors.Is(err, io.EOF) {              // client closed connection
			return nil
		}
		if err != nil { // anything else
			return err
		}

		var response protocol.Response
		req, err := protocol.DecodeRequest(payload) // turn into structured request
		if err != nil {
			response = protocol.Response{OK: false, Error: err.Error()}
		} else {
			response = command.Execute(s.store, req)
		}

		encoded, err := protocol.EncodeResponse(response) // turns the Response struct into bytes
		if err != nil {
			encoded, err = protocol.EncodeResponse(protocol.Response{ // small error msg in case the response is too big
				OK: false, Error: "response too large",
			})
			if err != nil { //in case the tiny error cant be encoded
				return err
			}
		}
		if err := protocol.WriteFrame(conn, encoded); err != nil {
			return err
		}
	}
}
