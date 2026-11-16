package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"raftkv/protocol"
)

func main() {
	if err := run(); err != nil { 
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// run does all the work and returns an error, so main can exit non-zero.
// os.Exit skips defers, so conn.Close() has to live inside run, not main.
func run() error {
	addr := flag.String("addr", "127.0.0.1:6379", "fastkv server address")
	flag.Parse()

	req, err := parseArgs(flag.Args())
	if err != nil {
		return err
	}

	conn, err := net.DialTimeout("tcp", *addr, 3*time.Second) // like Dial but with a time limit 
	if err != nil {
		return err
	}
	defer conn.Close()
	// One deadline covers the write and the read, so a silent server can't hang us.
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if err := protocol.WriteFrame(conn, payload); err != nil {
		return err
	}

	reply, err := protocol.ReadFrame(conn)
	if err != nil {
		return err
	}
	var resp protocol.Response
	if err := json.Unmarshal(reply, &resp); err != nil {
		return fmt.Errorf("invalid reply from server: %w", err)
	}
	return printResponse(req.Op, resp)
}

// parseArgs turns terminal arguments like ["set", "name", "Alice"] into a Request.
func parseArgs(args []string) (protocol.Request, error) {
	if len(args) == 0 {
		return protocol.Request{}, errors.New("usage: client [-addr host:port] SET key value | GET key | DELETE key")
	}

	op := strings.ToUpper(args[0]) // the server only accepts uppercase ops
	switch op {
	case protocol.OpSet:
		if len(args) != 3 {
			return protocol.Request{}, errors.New("SET needs a key and a value")
		}
		return protocol.Request{Op: op, Key: args[1], Value: args[2]}, nil
	case protocol.OpGet, protocol.OpDelete:
		if len(args) != 2 {
			return protocol.Request{}, fmt.Errorf("%s needs exactly one key", op)
		}
		return protocol.Request{Op: op, Key: args[1]}, nil
	default:
		return protocol.Request{}, fmt.Errorf("unknown command %q", args[0])
	}
}

// printresponse shows a readable result instead of raw JSON.
func printResponse(op string, resp protocol.Response) error {
	if !resp.OK {
		return errors.New(resp.Error)
	}
	found := resp.Found != nil && *resp.Found // Found is a *bool

	switch op {
	case protocol.OpSet:
		fmt.Println("OK")
	case protocol.OpGet:
		if found {
			fmt.Println(resp.Value)
		} else {
			fmt.Println("(not found)")
		}
	case protocol.OpDelete:
		if found {
			fmt.Println("deleted")
		} else {
			fmt.Println("(not found)")
		}
	}
	return nil
}