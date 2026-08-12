package mcachedb

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCache_SetGet(t *testing.T) {
	c, _ := New()
	user := NewUser("1", "alice", "alice@test.com", 25)
	c.Set(user)

	got, err := c.Get("1")
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, "alice", got.(*User).Username)
}

func TestCache_SetReturnsIsNew(t *testing.T) {
	c, _ := New()
	u := NewUser("1", "alice", "alice@test.com", 25)
	assert.True(t, c.Set(u), "first Set should return isNew=true")
	assert.False(t, c.Set(u), "second Set should return isNew=false")
}

func TestCache_GetMiss(t *testing.T) {
	c, _ := New()
	got, err := c.Get("missing")
	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestCache_Delete(t *testing.T) {
	c, _ := New()
	u := NewUser("1", "alice", "alice@test.com", 25)
	c.Set(u)
	c.Delete("1")

	got, _ := c.Get("1")
	assert.Nil(t, got)
}

func TestCache_Load(t *testing.T) {
	c, _ := New()
	u := NewUser("1", "alice", "alice@test.com", 25)
	c.Load(u)

	got, err := c.Get("1")
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, "alice", got.(*User).Username)
}

func TestCache_LoadVersionGuard(t *testing.T) {
	c, _ := New()
	u := NewUser("1", "alice", "alice@test.com", 25)
	u.Ver = 5
	c.Load(u)

	// Load with older version should be rejected
	older := NewUser("1", "stale", "stale@test.com", 99)
	older.Ver = 3
	c.Load(older)

	got, _ := c.Get("1")
	assert.Equal(t, "alice", got.(*User).Username) // should still be alice
}

func TestCache_LoadSkipsDirty(t *testing.T) {
	c, _ := New()
	u := NewUser("1", "dirty", "d@test.com", 1)
	// Set via setEntry to mark dirty
	c.setEntry(u, 1)

	// Load should not overwrite dirty entry
	older := NewUser("1", "clean", "c@test.com", 2)
	older.Ver = 2
	c.Load(older)

	// dirty entry has no entity version set by setEntry to 1 (BaseEntity.Ver starts at 1)
	// The load has Ver=2 which is higher, but it's blocked by dirty flag
	assert.True(t, c.IsDirty("1"))
}

func TestCache_Remove(t *testing.T) {
	c, _ := New()
	u := NewUser("1", "alice", "alice@test.com", 25)
	c.Set(u)
	c.Remove("1")

	got, _ := c.Get("1")
	assert.Nil(t, got)
	assert.Equal(t, 0, c.Len())
}

func TestCache_Clear(t *testing.T) {
	c, _ := New()
	for i := 0; i < 5; i++ {
		c.Set(NewUser(fmt.Sprintf("%d", i), "u", "u@test.com", i))
	}
	assert.Equal(t, 5, c.Len())
	c.Clear()
	assert.Equal(t, 0, c.Len())
}

func TestCache_MGet(t *testing.T) {
	c, _ := New()
	c.Set(NewUser("1", "alice", "alice@test.com", 25))
	c.Set(NewUser("2", "bob", "bob@test.com", 30))

	result, err := c.MGet([]string{"1", "2", "3"})
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "alice", result["1"].(*User).Username)
	assert.Equal(t, "bob", result["2"].(*User).Username)
	assert.Nil(t, result["3"])
}

func TestCache_Expiry(t *testing.T) {
	c, _ := New(WithDefaultExpiration(50 * time.Millisecond))
	c.Set(NewUser("1", "alice", "alice@test.com", 25))

	got, _ := c.Get("1")
	assert.NotNil(t, got)

	time.Sleep(80 * time.Millisecond)
	got, _ = c.Get("1")
	assert.Nil(t, got)
}

func TestCache_Stats(t *testing.T) {
	c, _ := New()
	c.Set(NewUser("1", "alice", "alice@test.com", 25))
	c.Get("1") // hit
	c.Get("2") // miss

	s := c.Stats()
	assert.Equal(t, int64(1), s.CacheHits)
	assert.Equal(t, int64(1), s.CacheMisses)
	assert.Equal(t, 1, s.CacheSize)
}

func TestCache_MaxCacheSize(t *testing.T) {
	c, _ := New(WithMaxCacheSize(3))
	for i := 0; i < 5; i++ {
		c.Set(NewUser(fmt.Sprintf("%d", i), "u", "u@test.com", i))
	}
	assert.LessOrEqual(t, c.Len(), 3)
}

func TestCache_CloseIdempotent(t *testing.T) {
	c, _ := New()
	assert.NotPanics(t, func() {
		assert.NoError(t, c.Close())
		assert.NoError(t, c.Close())
	})
}

func TestCache_Concurrent(t *testing.T) {
	c, _ := New()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			u := NewUser(fmt.Sprintf("%d", n%10), "u", "u@test.com", n)
			c.Set(u)
			c.Get(fmt.Sprintf("%d", n%10))
		}(i)
	}
	wg.Wait()
	assert.LessOrEqual(t, c.Len(), 10)
}
