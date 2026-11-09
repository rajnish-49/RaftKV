package server

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"raftkv/protocol"
	"raftkv/storage"
)

func startTestConnection(t *testing.T, store *storage.Store) (net.Conn, <-chan error) {
	t.Helper()
	client, peer := net.Pipe()
	if err := client.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		client.Close()
		peer.Close()
		t.Fatal(err)
	}
	s := &Server{store: store}
	done := make(chan error, 1)
	go func() {
		done <- s.handleConn(peer)
		close(done)
	}()
	t.Cleanup(func() {
		client.Close()
		peer.Close()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("connection handler did not exit after closing the connection")
		}
	})
	return client, done
}

func exchange(t *testing.T, conn net.Conn, payload string) protocol.Response {
	t.Helper()
	if err := protocol.WriteFrame(conn, []byte(payload)); err != nil {
		t.Fatal(err)
	}
	encoded, err := protocol.ReadFrame(conn)
	if err != nil {
		t.Fatal(err)
	}
	var response protocol.Response
	if err := json.Unmarshal(encoded, &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func TestHandleConn(t *testing.T) {
	conn, done := startTestConnection(t, storage.New())
	found, missing := true, false
	steps := []struct {
		payload   string
		want      protocol.Response
		wantError bool
	}{
		{payload: `{"op":"SET","key":"name","value":"Alice"}`, want: protocol.Response{OK: true}},
		{payload: `{"op":"GET","key":"name"}`, want: protocol.Response{OK: true, Found: &found, Value: "Alice"}},
		{payload: `{"op":`, wantError: true},
		{payload: `{"op":"UNKNOWN","key":"name"}`, wantError: true},
		// Valid commands still work after invalid requests on the same connection.
		{payload: `{"op":"GET","key":"name"}`, want: protocol.Response{OK: true, Found: &found, Value: "Alice"}},
		{payload: `{"op":"DELETE","key":"name"}`, want: protocol.Response{OK: true, Found: &found}},
		{payload: `{"op":"GET","key":"name"}`, want: protocol.Response{OK: true, Found: &missing}},
	}
	for _, step := range steps {
		got := exchange(t, conn, step.payload)
		if step.wantError {
			if got.OK || got.Error == "" || got.Found != nil || got.Value != "" {
				t.Fatalf("request %s: expected error response, got %#v", step.payload, got)
			}
		} else if !reflect.DeepEqual(got, step.want) {
			t.Fatalf("request %s: got %#v, want %#v", step.payload, got, step.want)
		}
	}
	conn.Close()
	if err := <-done; err != nil {
		t.Fatalf("normal disconnect returned an error: %v", err)
	}
}

func TestHandleConnSharesStore(t *testing.T) {
	store := storage.New()
	first, _ := startTestConnection(t, store)
	second, _ := startTestConnection(t, store)
	if response := exchange(t, first, `{"op":"SET","key":"shared","value":"Alice"}`); !response.OK {
		t.Fatalf("SET failed: %#v", response)
	}
	response := exchange(t, second, `{"op":"GET","key":"shared"}`)
	if !response.OK || response.Found == nil || !*response.Found || response.Value != "Alice" {
		t.Fatalf("second connection cannot read shared store: %#v", response)
	}
}

func TestHandleConnRejectsTruncatedFrame(t *testing.T) {
	conn, done := startTestConnection(t, storage.New())
	if _, err := conn.Write([]byte{0, 0}); err != nil {
		t.Fatal(err)
	}
	conn.Close()
	if err := <-done; !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("truncated frame: got %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestHandleConnContinuesAfterOversizedResponse(t *testing.T) {
	store := storage.New()
	store.Set("name", "Alice")
	// Seed directly to exercise the fallback even though command.Execute
	// prevents new oversized values from being stored through SET.
	store.Set("large", strings.Repeat("<", 200000))
	tests := []struct {
		name    string
		payload string
	}{
		{name: "oversized GET response", payload: `{"op":"GET","key":"large"}`},
		// The request fits in a frame, but the decode error echoes the unknown
		// operation and grows beyond the limit when JSON escapes each '<'.
		{name: "oversized decode error", payload: `{"op":"` + strings.Repeat("<", 200000) + `","key":"name"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, done := startTestConnection(t, store)
			response := exchange(t, conn, tt.payload)
			want := protocol.Response{OK: false, Error: "response too large"}
			if !reflect.DeepEqual(response, want) {
				t.Fatalf("fallback response = %#v, want %#v", response, want)
			}

			response = exchange(t, conn, `{"op":"GET","key":"name"}`)
			if !response.OK || response.Found == nil || !*response.Found || response.Value != "Alice" || response.Error != "" {
				t.Fatalf("valid request after oversized response failed: %#v", response)
			}
			conn.Close()
			if err := <-done; err != nil {
				t.Fatalf("normal disconnect returned an error: %v", err)
			}
		})
	}
}

func TestServeMultipleTCPClients(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := New(listener, storage.New())
	done := make(chan error, 1)
	go func() {
		done <- s.Serve()
	}()
	t.Cleanup(func() {
		listener.Close()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Serve after listener close: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Error("Serve did not stop after listener close")
		}
	})

	dial := func() net.Conn {
		t.Helper()
		conn, err := net.DialTimeout("tcp", listener.Addr().String(), 10*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Close() })
		if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
			t.Fatal(err)
		}
		return conn
	}

	// An idle first client must not block the accept loop from serving another.
	first := dial()
	second := dial()
	response := exchange(t, second, `{"op":"SET","key":"shared","value":"Alice"}`)
	if !response.OK || response.Error != "" {
		t.Fatalf("SET from second client failed: %#v", response)
	}
	response = exchange(t, first, `{"op":"GET","key":"shared"}`)
	if !response.OK || response.Found == nil || !*response.Found || response.Value != "Alice" {
		t.Fatalf("first client cannot read shared store: %#v", response)
	}
	response = exchange(t, first, `{"op":"DELETE","key":"shared"}`)
	if !response.OK || response.Found == nil || !*response.Found {
		t.Fatalf("DELETE from first client failed: %#v", response)
	}
	response = exchange(t, second, `{"op":"GET","key":"shared"}`)
	if !response.OK || response.Found == nil || *response.Found || response.Value != "" {
		t.Fatalf("second client did not observe deletion: %#v", response)
	}
}

type failingListener struct {
	err error
}

func (l failingListener) Accept() (net.Conn, error) { return nil, l.err }
func (failingListener) Close() error                { return nil }
func (failingListener) Addr() net.Addr              { return &net.TCPAddr{} }

func TestServeAcceptError(t *testing.T) {
	want := errors.New("accept failed")
	s := New(failingListener{err: want}, storage.New())
	if err := s.Serve(); !errors.Is(err, want) {
		t.Fatalf("Serve error = %v, want %v", err, want)
	}
}
