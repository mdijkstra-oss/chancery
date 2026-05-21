package gemini

import (
	"sync"
	"time"
)

type CacheEntry struct {
	ResourceName string
	ExpireTime   time.Time
	TTL          time.Duration
	HasBeenRead  bool
}

type CacheStore struct {
	mu      sync.Mutex
	entries map[string]CacheEntry
	flights map[string]*inflight
}

type inflight struct {
	done  chan struct{}
	entry CacheEntry
	err   error
}

func NewCacheStore() *CacheStore {
	return &CacheStore{
		entries: make(map[string]CacheEntry),
		flights: make(map[string]*inflight),
	}
}

func IsValid(entry CacheEntry, now time.Time) bool {
	return now.Before(entry.ExpireTime)
}

func ShouldRenew(entry CacheEntry, now time.Time) bool {
	remaining := entry.ExpireTime.Sub(now)
	threshold := entry.TTL / 5
	return remaining > 0 && remaining < threshold
}

func (s *CacheStore) FindValid(hash string, now time.Time) (CacheEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[hash]
	if !ok {
		return CacheEntry{}, false
	}
	if !IsValid(entry, now) {
		delete(s.entries, hash)
		return CacheEntry{}, false
	}
	return entry, true
}

func (s *CacheStore) Store(hash string, entry CacheEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[hash] = entry
}

func (s *CacheStore) UpdateExpiry(hash string, expireTime time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[hash]
	if !ok {
		return
	}
	entry.ExpireTime = expireTime
	s.entries[hash] = entry
}

func (s *CacheStore) MarkFirstRead(hash string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[hash]
	if !ok {
		return false
	}
	if entry.HasBeenRead {
		return false
	}
	entry.HasBeenRead = true
	s.entries[hash] = entry
	return true
}

func (s *CacheStore) AcquireOrWait(hash string) (isCreator bool, fl *inflight) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.flights[hash]; ok {
		return false, existing
	}
	fl = &inflight{done: make(chan struct{})}
	s.flights[hash] = fl
	return true, fl
}

func (s *CacheStore) CompleteInflight(hash string, entry CacheEntry, err error) {
	s.mu.Lock()
	fl, ok := s.flights[hash]
	if ok {
		fl.entry = entry
		fl.err = err
		delete(s.flights, hash)
	}
	s.mu.Unlock()
	if ok {
		close(fl.done)
	}
	if err == nil {
		s.Store(hash, entry)
	}
}

func WaitInflight(fl *inflight) (CacheEntry, error) {
	<-fl.done
	return fl.entry, fl.err
}
