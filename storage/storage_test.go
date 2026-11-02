package storage

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
)

func TestSetAndGet(t *testing.T) {
	store := New()
	store.Set("other", "keep")

	for _, want := range []string{"Alice", "Bob", ""} {
		store.Set("name", want)
		got, found := store.Get("name")
		if !found || got != want {
			t.Fatalf("Get(name) = %q, %v; want %q, true", got, found, want)
		}
	}
	if got, found := store.Get("other"); !found || got != "keep" {
		t.Fatal("setting name changed an unrelated key")
	}
}

func TestMissingKey(t *testing.T) {
	store := New()
	if value, found := store.Get("missing"); value != "" || found {
		t.Fatalf("Get(missing) = %q, %v; want empty string, false", value, found)
	}
	if store.Delete("missing") {
		t.Fatal("Delete(missing) returned true")
	}
}

func TestDelete(t *testing.T) {
	for _, value := range []string{"Alice", ""} {
		t.Run(fmt.Sprintf("value=%q", value), func(t *testing.T) {
			store := New()
			store.Set("name", value)
			store.Set("other", "keep")
			if !store.Delete("name") {
				t.Fatal("Delete returned false for an existing key")
			}
			if got, found := store.Get("name"); got != "" || found {
				t.Fatalf("after Delete: Get(name) = %q, %v; want empty string, false", got, found)
			}
			if store.Delete("name") {
				t.Fatal("deleting the key twice returned true")
			}
			if got, found := store.Get("other"); !found || got != "keep" {
				t.Fatal("deleting name changed an unrelated key")
			}
		})
	}
}

func TestConcurrentStoreAccess(t *testing.T) {
	const workers = 32
	const iterations = 100
	store := New()
	var wg sync.WaitGroup
	start := make(chan struct{})

	for id := 0; id < workers; id++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start
		
			key := fmt.Sprintf("worker-%d", id)
			for iteration := 0; iteration < iterations; iteration++ {
				value := strconv.Itoa(iteration)
				store.Set(key, value)
				if got, found := store.Get(key); !found || got != value {
					t.Errorf("Get(%q) = %q, %v; want %q, true", key, got, found, value)
					return
				}
				store.Set(key, "updated")
				if got, found := store.Get(key); !found || got != "updated" {
					t.Errorf("overwrite failed for %q: got %q, %v", key, got, found)
					return
				}
				if !store.Delete(key) {
					t.Errorf("Delete(%q) returned false for an existing key", key)
					return
				}
				if got, found := store.Get(key); got != "" || found {
					t.Errorf("key %q remains after deletion: %q, %v", key, got, found)
					return
				}
			}
		}(id)
	}
	close(start)
	wg.Wait()
}

func TestConcurrentWritesToSameKey(t *testing.T) {
	const workers = 32
	const iterations = 100
	store := New()
	store.Set("shared", "0")
	var wg sync.WaitGroup
	start := make(chan struct{})

	for id := 0; id < workers; id++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start
			value := strconv.Itoa(id)
			for iteration := 0; iteration < iterations; iteration++ {
				store.Set("shared", value)
				got, found := store.Get("shared")
		
				n, err := strconv.Atoi(got)
				if !found || err != nil || n < 0 || n >= workers {
					t.Errorf("invalid shared value: %q, found=%v", got, found)
					return
				}
			}
		}(id)
	}
	close(start)
	wg.Wait()

	got, found := store.Get("shared")
	n, err := strconv.Atoi(got)
	if !found || err != nil || n < 0 || n >= workers {
		t.Fatalf("invalid final shared value: %q, found=%v", got, found)
	}
}
