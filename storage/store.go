package storage

import "sync"
import "hash/fnv"

type shard struct {
	mu   sync.RWMutex
	data map[string]string
}

const shardCount = 32

type Store struct {
	shards [shardCount]shard
}

func New() *Store {
	s := &Store{}

	for i := range s.shards {
		s.shards[i].data = make(map[string]string)
	}

	return s
}

func (s *Store) selectShard(key string) *shard {
	h := fnv.New32a()
	h.Write([]byte(key)) // feed the key into hasher

	index := h.Sum32() % shardCount
	return &s.shards[index]
}

func (s *Store) Set(key, value string) {
	sh := s.selectShard(key)

	sh.mu.Lock()
	defer sh.mu.Unlock()

	sh.data[key] = value
}

func (s *Store) Get(key string) (string, bool) {

	sh := s.selectShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()

	value, exists := sh.data[key]
	return value, exists
}

func (s *Store) Delete(key string) bool {
	sh := s.selectShard(key)

	sh.mu.Lock()
	defer sh.mu.Unlock()

	_, exists := sh.data[key]
	delete(sh.data, key)
	return exists
}
