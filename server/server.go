package server

import (
	"errors"
	"io"
	"net"

	"raftkv/command"
	"raftkv/protocol"
	"raftkv/storage"
)

type Server struct {
	listener net.Listener
	store    *storage.Store
}

func (s *Server) handleConn(conn net.Conn) error {
	defer conn.Close()

	for {
		payload, err := protocol.ReadFrame(conn)  // tcp has no message boundaries , returns 1 complete message 
		if errors.Is(err, io.EOF) { // client closed connection
			return nil
		}
		if err != nil { // anything else 
			return err
		}

		var response protocol.Response
		req, err := protocol.DecodeRequest(payload)  // turn into structured request
		if err != nil {
			response = protocol.Response{OK: false, Error: err.Error()}
		} else {
			response = command.Execute(s.store, req)
		}

		encoded, err := protocol.EncodeResponse(response) // turns the Response struct into bytes
		if err != nil {
			encoded, err = protocol.EncodeResponse(protocol.Response{// small error msg in case the response is too big 
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
