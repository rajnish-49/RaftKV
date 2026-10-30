package command_test

import (
	"reflect"
	"testing"

	"raftkv/command"
	"raftkv/protocol"
	"raftkv/storage"
)

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
