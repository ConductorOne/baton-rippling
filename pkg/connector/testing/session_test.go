package testing

import (
	"context"
	"testing"

	"github.com/conductorone/baton-sdk/pkg/types/sessions"
	"github.com/stretchr/testify/assert"
)

func TestMemorySessionStore_GetSet(t *testing.T) {
	ctx := context.Background()
	s := NewMemorySessionStore()

	v, found, err := s.Get(ctx, "missing")
	assert.NoError(t, err)
	assert.False(t, found)
	assert.Nil(t, v)

	assert.NoError(t, s.Set(ctx, "k", []byte("v")))
	v, found, err = s.Get(ctx, "k")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, []byte("v"), v)
}

func TestMemorySessionStore_PrefixIsolation(t *testing.T) {
	ctx := context.Background()
	s := NewMemorySessionStore()

	assert.NoError(t, s.Set(ctx, "k", []byte("a"), sessions.WithPrefix("p1")))
	assert.NoError(t, s.Set(ctx, "k", []byte("b"), sessions.WithPrefix("p2")))

	v, _, _ := s.Get(ctx, "k", sessions.WithPrefix("p1"))
	assert.Equal(t, []byte("a"), v)
	v, _, _ = s.Get(ctx, "k", sessions.WithPrefix("p2"))
	assert.Equal(t, []byte("b"), v)
	_, found, _ := s.Get(ctx, "k") // no prefix
	assert.False(t, found)
}

func TestMemorySessionStore_GetManySetMany(t *testing.T) {
	ctx := context.Background()
	s := NewMemorySessionStore()

	in := map[string][]byte{
		"a": []byte("1"),
		"b": []byte("2"),
		"c": []byte("3"),
	}
	assert.NoError(t, s.SetMany(ctx, in, sessions.WithPrefix("workers")))

	out, _, err := s.GetMany(ctx, []string{"a", "b", "missing"}, sessions.WithPrefix("workers"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("1"), out["a"])
	assert.Equal(t, []byte("2"), out["b"])
	_, ok := out["missing"]
	assert.False(t, ok)
}

func TestMemorySessionStore_DeleteAndClear(t *testing.T) {
	ctx := context.Background()
	s := NewMemorySessionStore()
	_ = s.Set(ctx, "k", []byte("v"), sessions.WithPrefix("p"))
	_ = s.Set(ctx, "other", []byte("z"), sessions.WithPrefix("q"))

	assert.NoError(t, s.Delete(ctx, "k", sessions.WithPrefix("p")))
	_, found, _ := s.Get(ctx, "k", sessions.WithPrefix("p"))
	assert.False(t, found)

	_, found, _ = s.Get(ctx, "other", sessions.WithPrefix("q"))
	assert.True(t, found, "Delete must scope to prefix")

	assert.NoError(t, s.Clear(ctx, sessions.WithPrefix("q")))
	_, found, _ = s.Get(ctx, "other", sessions.WithPrefix("q"))
	assert.False(t, found)
}

func TestMemorySessionStore_GetReturnsCopy(t *testing.T) {
	// Mutating the returned slice must not corrupt the stored entry.
	ctx := context.Background()
	s := NewMemorySessionStore()
	_ = s.Set(ctx, "k", []byte("hello"))

	v, _, _ := s.Get(ctx, "k")
	v[0] = 'X'

	v2, _, _ := s.Get(ctx, "k")
	assert.Equal(t, []byte("hello"), v2)
}
