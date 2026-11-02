package command_test

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"raftkv/command"
	"raftkv/protocol"
	"raftkv/storage"
)

func TestExecuteSetRejectsValueThatCannotBeReturned(t *testing.T) {
	// This valid SET frame is about 200 KB, but JSON encoding its GET response
	// expands each '<' into six bytes and exceeds the 1 MiB frame limit.
	payload := []byte(`{"op":"SET","key":"name","value":"` + strings.Repeat("<", 200000) + `"}`)
	var requestWire bytes.Buffer
	if err := protocol.WriteFrame(&requestWire, payload); err != nil {
		t.Fatal(err)
	}
	req, err := protocol.DecodeRequest(payload)
	if err != nil {
		t.Fatal(err)
	}

	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing=%v", existing), func(t *testing.T) {
			store := storage.New()
			want := ""
			if existing {
				want = "original"
				store.Set(req.Key, want)
			}
			response := command.Execute(store, req)
			if response.OK || response.Error == "" {
				t.Fatalf("oversized GET response should prevent SET, got %#v", response)
			}
			if got, found := store.Get(req.Key); got != want || found != existing {
				t.Fatalf("rejected SET changed store: got %q, %v; want %q, %v", got, found, want, existing)
			}
			encoded, err := protocol.EncodeResponse(response)
			if err != nil {
				t.Fatal(err)
			}
			var responseWire bytes.Buffer
			if err := protocol.WriteFrame(&responseWire, encoded); err != nil {
				t.Fatalf("cannot send rejection response: %v", err)
			}
		})
	}
}

func TestExecuteSetResponseSizeBoundary(t *testing.T) {
	const frameLimit = 1 << 20
	const overhead = len(`{"ok":true,"found":true,"value":""}`)
	for _, extra := range []int{0, 1} {
		t.Run(fmt.Sprintf("extra=%d", extra), func(t *testing.T) {
			store := storage.New()
			value := strings.Repeat("a", frameLimit-overhead+extra)
			response := command.Execute(store, protocol.Request{Op: protocol.OpSet, Key: "k", Value: value})
			if extra == 1 {
				if response.OK || response.Error == "" {
					t.Fatalf("oversized value was accepted: %#v", response)
				}
				if _, found := store.Get("k"); found {
					t.Fatal("rejected SET created the key")
				}
				return
			}
			if !response.OK || response.Error != "" {
				t.Fatalf("valid boundary value was rejected: %#v", response)
			}
			get := command.Execute(store, protocol.Request{Op: protocol.OpGet, Key: "k"})
			if !get.OK || get.Found == nil || !*get.Found || get.Value != value {
				t.Fatal("accepted value could not be retrieved intact")
			}
			encoded, err := protocol.EncodeResponse(get)
			if err != nil || len(encoded) != frameLimit {
				t.Fatalf("GET response: got %d bytes, %v; want %d bytes, nil", len(encoded), err, frameLimit)
			}
			var wire bytes.Buffer
			if err := protocol.WriteFrame(&wire, encoded); err != nil {
				t.Fatal(err)
			}
			decoded, err := protocol.ReadFrame(&wire)
			if err != nil || !bytes.Equal(decoded, encoded) {
				t.Fatalf("GET response did not survive framing: %v", err)
			}
		})
	}
}

func TestExecuteSet(t *testing.T) {
	store := storage.New()
	store.Set("other", "keep")

	for _, value := range []string{"Alice", "Bob", ""} {
		response := command.Execute(store, protocol.Request{
			Op: protocol.OpSet, Key: "name", Value: value,
		})
		if want := (protocol.Response{OK: true}); !reflect.DeepEqual(response, want) {
			t.Fatalf("SET response = %#v, want %#v", response, want)
		}
		if got, found := store.Get("name"); !found || got != value {
			t.Fatalf("after SET: stored value = %q, found = %v; want %q, true", got, found, value)
		}
	}
	if got, found := store.Get("other"); !found || got != "keep" {
		t.Fatal("SET changed an unrelated key")
	}
}

func TestExecuteGet(t *testing.T) {
	store := storage.New()
	store.Set("name", "Alice")
	store.Set("empty", "")

	tests := []struct {
		name  string
		key   string
		value string
		found bool
	}{
		{name: "existing key", key: "name", value: "Alice", found: true},
		{name: "empty value", key: "empty", found: true},
		{name: "missing key", key: "missing", found: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := command.Execute(store, protocol.Request{Op: protocol.OpGet, Key: tt.key})
			want := protocol.Response{OK: true, Found: &tt.found, Value: tt.value}
			if !reflect.DeepEqual(response, want) {
				t.Errorf("GET response = %#v, want %#v", response, want)
			}
			if got, found := store.Get(tt.key); got != tt.value || found != tt.found {
				t.Error("GET changed the stored value or key existence")
			}
		})
	}
}

func TestExecuteDelete(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		exists bool
	}{
		{name: "existing key", value: "Alice", exists: true},
		{name: "empty value", exists: true},
		{name: "missing key", exists: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := storage.New()
			store.Set("other", "keep")
			if tt.exists {
				store.Set("name", tt.value)
			}

			response := command.Execute(store, protocol.Request{Op: protocol.OpDelete, Key: "name"})
			want := protocol.Response{OK: true, Found: &tt.exists}
			if !reflect.DeepEqual(response, want) {
				t.Errorf("DELETE response = %#v, want %#v", response, want)
			}
			if _, found := store.Get("name"); found {
				t.Error("key still exists after DELETE")
			}
			if got, found := store.Get("other"); !found || got != "keep" {
				t.Error("DELETE changed an unrelated key")
			}
		})
	}
}

func TestExecuteUnknownOperation(t *testing.T) {
	store := storage.New()
	store.Set("name", "Alice")

	response := command.Execute(store, protocol.Request{Op: "INVALID", Key: "name", Value: "Bob"})
	if response.OK || response.Error == "" {
		t.Fatalf("unknown operation should fail with an error, got %#v", response)
	}
	if response.Found != nil || response.Value != "" {
		t.Errorf("error response should not contain a lookup result, got %#v", response)
	}
	if got, found := store.Get("name"); !found || got != "Alice" {
		t.Fatal("unknown operation changed the store")
	}
}
