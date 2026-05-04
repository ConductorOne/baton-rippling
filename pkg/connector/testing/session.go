// Package testing provides shared test doubles for connector tests.
package testing

import (
	"context"
	"strings"
	"sync"

	"github.com/conductorone/baton-sdk/pkg/types/sessions"
)

// MemorySessionStore is a map-backed implementation of
// sessions.SessionStore for use in unit and integration tests. It
// honors the WithPrefix option so tests can exercise prefix-scoped
// gets and sets the same way production code does.
//
// Not optimized for performance and not a faithful reproduction of
// the SQLite-backed production store's size limits — tests should
// avoid storing more than a handful of MB.
type MemorySessionStore struct {
	mu   sync.Mutex
	data map[string][]byte
}

// NewMemorySessionStore returns an empty in-memory session store.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{data: make(map[string][]byte)}
}

func resolveBag(ctx context.Context, opts []sessions.SessionStoreOption) (*sessions.SessionStoreBag, error) {
	bag := &sessions.SessionStoreBag{}
	for _, opt := range opts {
		if err := opt(ctx, bag); err != nil {
			return nil, err
		}
	}
	return bag, nil
}

func keyWithPrefix(bag *sessions.SessionStoreBag, key string) string {
	if bag.Prefix == "" {
		return key
	}
	return bag.Prefix + ":" + key
}

func (s *MemorySessionStore) Get(ctx context.Context, key string, opt ...sessions.SessionStoreOption) ([]byte, bool, error) {
	bag, err := resolveBag(ctx, opt)
	if err != nil {
		return nil, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[keyWithPrefix(bag, key)]
	if !ok {
		return nil, false, nil
	}
	out := make([]byte, len(v))
	copy(out, v)
	return out, true, nil
}

func (s *MemorySessionStore) GetMany(ctx context.Context, keys []string, opt ...sessions.SessionStoreOption) (map[string][]byte, []string, error) {
	bag, err := resolveBag(ctx, opt)
	if err != nil {
		return nil, nil, err
	}
	out := make(map[string][]byte, len(keys))
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, key := range keys {
		if v, ok := s.data[keyWithPrefix(bag, key)]; ok {
			cp := make([]byte, len(v))
			copy(cp, v)
			out[key] = cp
		}
	}
	return out, nil, nil
}

func (s *MemorySessionStore) Set(ctx context.Context, key string, value []byte, opt ...sessions.SessionStoreOption) error {
	bag, err := resolveBag(ctx, opt)
	if err != nil {
		return err
	}
	cp := make([]byte, len(value))
	copy(cp, value)
	s.mu.Lock()
	s.data[keyWithPrefix(bag, key)] = cp
	s.mu.Unlock()
	return nil
}

func (s *MemorySessionStore) SetMany(ctx context.Context, values map[string][]byte, opt ...sessions.SessionStoreOption) error {
	bag, err := resolveBag(ctx, opt)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, v := range values {
		cp := make([]byte, len(v))
		copy(cp, v)
		s.data[keyWithPrefix(bag, key)] = cp
	}
	return nil
}

func (s *MemorySessionStore) Delete(ctx context.Context, key string, opt ...sessions.SessionStoreOption) error {
	bag, err := resolveBag(ctx, opt)
	if err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.data, keyWithPrefix(bag, key))
	s.mu.Unlock()
	return nil
}

func (s *MemorySessionStore) Clear(ctx context.Context, opt ...sessions.SessionStoreOption) error {
	bag, err := resolveBag(ctx, opt)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if bag.Prefix == "" {
		s.data = make(map[string][]byte)
		return nil
	}
	prefix := bag.Prefix + ":"
	for k := range s.data {
		if strings.HasPrefix(k, prefix) {
			delete(s.data, k)
		}
	}
	return nil
}

func (s *MemorySessionStore) GetAll(ctx context.Context, pageToken string, opt ...sessions.SessionStoreOption) (map[string][]byte, string, error) {
	bag, err := resolveBag(ctx, opt)
	if err != nil {
		return nil, "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string][]byte)
	prefix := ""
	if bag.Prefix != "" {
		prefix = bag.Prefix + ":"
	}
	for k, v := range s.data {
		if prefix != "" && !strings.HasPrefix(k, prefix) {
			continue
		}
		shortKey := strings.TrimPrefix(k, prefix)
		cp := make([]byte, len(v))
		copy(cp, v)
		out[shortKey] = cp
	}
	return out, "", nil
}

// Compile-time check that MemorySessionStore satisfies the interface.
var _ sessions.SessionStore = (*MemorySessionStore)(nil)
