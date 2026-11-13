package storage

import (
	"fmt"
	"sync/atomic"
	"testing"
)

const benchmarkKeyCount = 4096
const benchmarkValue = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// Prepare keys and populate the store outside the measured interval. Set
// benchmarks overwrite existing keys so map growth is not part of the result.
func prepareBenchmarkStore(b *testing.B, keyCount int) (*Store, []string) {
	b.Helper()
	store := New()
	keys := make([]string, keyCount)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%08d", i)
		store.Set(keys[i], benchmarkValue)
	}
	b.ReportAllocs()
	return store, keys
}

func BenchmarkStoreGet(b *testing.B) {
	store, keys := prepareBenchmarkStore(b, benchmarkKeyCount)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, found := store.Get(keys[i%len(keys)]); !found {
			b.Fatal("benchmark key is missing")
		}
	}
}

func BenchmarkStoreSet(b *testing.B) {
	store, keys := prepareBenchmarkStore(b, benchmarkKeyCount)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Set(keys[i%len(keys)], benchmarkValue)
	}
}

// Each ten operations contain nine reads and one write. One reported operation
// is one store method call, rather than an entire read/write batch.
func BenchmarkStoreMixed(b *testing.B) {
	store, keys := prepareBenchmarkStore(b, benchmarkKeyCount)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := keys[i%len(keys)]
		if i%10 == 0 {
			store.Set(key, benchmarkValue)
		} else if _, found := store.Get(key); !found {
			b.Fatal("benchmark key is missing")
		}
	}
}

func BenchmarkStoreGetParallel(b *testing.B) {
	for _, keyCount := range []int{1, benchmarkKeyCount} {
		b.Run(fmt.Sprintf("Keys%d", keyCount), func(b *testing.B) {
			store, keys := prepareBenchmarkStore(b, keyCount)
			var workers atomic.Uint64
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				// Assign each worker a different starting key. This counter is
				// touched once per worker, not once per store operation.
				i := int(workers.Add(1)-1) * 997
				for pb.Next() {
					if _, found := store.Get(keys[i%len(keys)]); !found {
						b.Error("benchmark key is missing")
						return
					}
					i++
				}
			})
		})
	}
}

func BenchmarkStoreSetParallel(b *testing.B) {
	for _, keyCount := range []int{1, benchmarkKeyCount} {
		b.Run(fmt.Sprintf("Keys%d", keyCount), func(b *testing.B) {
			store, keys := prepareBenchmarkStore(b, keyCount)
			var workers atomic.Uint64
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := int(workers.Add(1)-1) * 997
				for pb.Next() {
					store.Set(keys[i%len(keys)], benchmarkValue)
					i++
				}
			})
		})
	}
}

func BenchmarkStoreMixedParallel(b *testing.B) {
	for _, keyCount := range []int{1, benchmarkKeyCount} {
		b.Run(fmt.Sprintf("Keys%d", keyCount), func(b *testing.B) {
			store, keys := prepareBenchmarkStore(b, keyCount)
			var workers atomic.Uint64
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := int(workers.Add(1)-1) * 997
				operation := 0
				for pb.Next() {
					key := keys[i%len(keys)]
					if operation%10 == 0 {
						store.Set(key, benchmarkValue)
					} else if _, found := store.Get(key); !found {
						b.Error("benchmark key is missing")
						return
					}
					i++
					operation++
				}
			})
		})
	}
}
