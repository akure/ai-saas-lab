package kernel

import "sync"

// APIKeyRecord is the in-memory representation of a key used by the auth layer.
// It allows the MVP to model activation state without introducing a database yet.
type APIKeyRecord struct {
	Key    string
	Plan   string
	Active bool
}

// Store is deliberately dumb — just thread-safe maps. In production this
// would be Postgres/Redis; the migration and policy code above doesn't care
// what's underneath, which is exactly why swapping it later is a one-file change.
type Store struct {
	mu            sync.RWMutex
	apiKeys       map[string]APIKeyRecord // key -> record
	usage         map[string]int          // key -> tokens used today
	subscriptions map[string]State        // key -> subscription state
}

func NewStore() *Store {
	return &Store{
		apiKeys:       make(map[string]APIKeyRecord),
		usage:         make(map[string]int),
		subscriptions: make(map[string]State),
	}
}

func (s *Store) SeedAPIKey(key, plan string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key == "" {
		return
	}
	s.apiKeys[key] = APIKeyRecord{Key: key, Plan: plan, Active: true}
}

func (s *Store) IsValidAPIKey(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.apiKeys[key]
	return ok && rec.Active
}

func (s *Store) ActivateAPIKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.apiKeys[key]; ok {
		rec.Active = true
		s.apiKeys[key] = rec
	}
}

func (s *Store) RevokeAPIKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.apiKeys[key]; ok {
		rec.Active = false
		s.apiKeys[key] = rec
	}
}

func (s *Store) APIKeyInfo(key string) (APIKeyRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.apiKeys[key]
	return rec, ok
}

func (s *Store) AddUsage(key string, tokens int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usage[key] += tokens
	return s.usage[key]
}

func (s *Store) UsageFor(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.usage[key]
}

func (s *Store) SetSubscriptionState(key string, st State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscriptions[key] = st
}

func (s *Store) SubscriptionState(key string) State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if st, ok := s.subscriptions[key]; ok {
		return st
	}
	return "unknown"
}
