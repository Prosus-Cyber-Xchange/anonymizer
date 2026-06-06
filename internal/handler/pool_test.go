package handler

import (
	"bytes"
	"sync"
	"testing"

	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetResponseBuffer(t *testing.T) {
	t.Run("returns a bytes.Buffer", func(t *testing.T) {
		pool := NewBufferPool()
		buf := pool.GetResponseBuffer()

		require.NotNil(t, buf)
		assert.IsType(t, &bytes.Buffer{}, buf)
	})

	t.Run("returns an empty buffer", func(t *testing.T) {
		pool := NewBufferPool()
		buf := pool.GetResponseBuffer()

		assert.Equal(t, 0, buf.Len())
		assert.Equal(t, "", buf.String())
	})

	t.Run("returns a usable buffer for writing", func(t *testing.T) {
		pool := NewBufferPool()
		buf := pool.GetResponseBuffer()

		testData := "test response"
		buf.WriteString(testData)
		assert.Equal(t, testData, buf.String())
	})
}

func TestPutResponseBuffer(t *testing.T) {
	t.Run("resets buffer before returning to pool", func(t *testing.T) {
		pool := NewBufferPool()
		buf := pool.GetResponseBuffer()

		// Write some data to the buffer
		buf.WriteString("test data")
		assert.Greater(t, buf.Len(), 0)

		// Put it back
		pool.PutResponseBuffer(buf)

		// Get another buffer - should be reset
		buf2 := pool.GetResponseBuffer()
		assert.Equal(t, 0, buf2.Len())
	})

	t.Run("allows buffer reuse after reset", func(t *testing.T) {
		pool := NewBufferPool()
		buf := pool.GetResponseBuffer()

		buf.WriteString("first write")
		pool.PutResponseBuffer(buf)

		buf2 := pool.GetResponseBuffer()
		buf2.WriteString("second write")
		assert.Equal(t, "second write", buf2.String())
	})
}

func TestGetEntityMap(t *testing.T) {
	t.Run("returns a map for storing entities", func(t *testing.T) {
		pool := NewBufferPool()
		entityMap := pool.GetEntityMap()

		require.NotNil(t, entityMap)
		assert.IsType(t, make(map[pattern.Entity]struct{}), entityMap)
	})

	t.Run("returns an empty map", func(t *testing.T) {
		pool := NewBufferPool()
		entityMap := pool.GetEntityMap()

		assert.Equal(t, 0, len(entityMap))
	})

	t.Run("returns a usable map for storing entities", func(t *testing.T) {
		pool := NewBufferPool()
		entityMap := pool.GetEntityMap()

		entityMap[pattern.EntityEmail] = struct{}{}
		entityMap[pattern.EntityCPF] = struct{}{}

		assert.Equal(t, 2, len(entityMap))
		assert.Contains(t, entityMap, pattern.EntityEmail)
		assert.Contains(t, entityMap, pattern.EntityCPF)
	})
}

func TestPutEntityMap(t *testing.T) {
	t.Run("clears the map before returning to pool", func(t *testing.T) {
		pool := NewBufferPool()
		entityMap := pool.GetEntityMap()

		// Add some entities to the map
		entityMap[pattern.EntityEmail] = struct{}{}
		entityMap[pattern.EntityCPF] = struct{}{}
		assert.Equal(t, 2, len(entityMap))

		// Put it back
		pool.PutEntityMap(entityMap)

		// Get another map - should be empty
		entityMap2 := pool.GetEntityMap()
		assert.Equal(t, 0, len(entityMap2))
	})

	t.Run("allows map reuse after clearing", func(t *testing.T) {
		pool := NewBufferPool()
		entityMap := pool.GetEntityMap()

		entityMap[pattern.EntityEmail] = struct{}{}
		pool.PutEntityMap(entityMap)

		entityMap2 := pool.GetEntityMap()
		entityMap2[pattern.EntityIPAddress] = struct{}{}
		assert.Equal(t, 1, len(entityMap2))
		assert.Contains(t, entityMap2, pattern.EntityIPAddress)
	})
}

func TestBufferPoolConcurrency(t *testing.T) {
	t.Run("handles concurrent response buffer operations", func(t *testing.T) {
		pool := NewBufferPool()
		numGoroutines := 100
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				buf := pool.GetResponseBuffer()
				buf.WriteString("test")
				pool.PutResponseBuffer(buf)
			}()
		}

		wg.Wait()
	})

	t.Run("handles concurrent entity map operations", func(t *testing.T) {
		pool := NewBufferPool()
		numGoroutines := 100
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				entityMap := pool.GetEntityMap()
				entityMap[pattern.EntityEmail] = struct{}{}
				pool.PutEntityMap(entityMap)
			}()
		}

		wg.Wait()
	})

	t.Run("handles mixed concurrent operations", func(t *testing.T) {
		pool := NewBufferPool()
		numGoroutines := 50
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Get and put response buffer
				respBuf := pool.GetResponseBuffer()
				respBuf.WriteString("test")
				pool.PutResponseBuffer(respBuf)

				// Get and put entity map
				entityMap := pool.GetEntityMap()
				entityMap[pattern.EntityCPF] = struct{}{}
				pool.PutEntityMap(entityMap)
			}()
		}

		wg.Wait()
	})
}

func TestBufferPoolBufferReuse(t *testing.T) {
	t.Run("response buffer is reused from pool", func(t *testing.T) {
		pool := NewBufferPool()

		buf1 := pool.GetResponseBuffer()
		buf1Address := buf1
		pool.PutResponseBuffer(buf1)

		buf2 := pool.GetResponseBuffer()
		// Both should reference the same underlying buffer (or reused one)
		assert.Equal(t, buf1Address, buf2)
	})

	t.Run("entity map is reused from pool", func(t *testing.T) {
		pool := NewBufferPool()

		map1 := pool.GetEntityMap()
		pool.PutEntityMap(map1)

		map2 := pool.GetEntityMap()
		// Should be reused or new from pool
		assert.NotNil(t, map2)
		assert.Equal(t, 0, len(map2))
	})
}

func TestBufferPoolIsolation(t *testing.T) {
	t.Run("response buffers are independent", func(t *testing.T) {
		pool := NewBufferPool()

		buf1 := pool.GetResponseBuffer()
		buf2 := pool.GetResponseBuffer()

		buf1.WriteString("data1")
		buf2.WriteString("data2")

		assert.Equal(t, "data1", buf1.String())
		assert.Equal(t, "data2", buf2.String())
	})

	t.Run("entity maps are independent", func(t *testing.T) {
		pool := NewBufferPool()

		map1 := pool.GetEntityMap()
		map2 := pool.GetEntityMap()

		map1[pattern.EntityEmail] = struct{}{}
		map2[pattern.EntityCPF] = struct{}{}

		assert.Equal(t, 1, len(map1))
		assert.Equal(t, 1, len(map2))
		assert.Contains(t, map1, pattern.EntityEmail)
		assert.Contains(t, map2, pattern.EntityCPF)
	})
}

func BenchmarkBufferPoolResponseBuffer(b *testing.B) {
	pool := NewBufferPool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := pool.GetResponseBuffer()
		buf.WriteString("test data")
		pool.PutResponseBuffer(buf)
	}
}

func BenchmarkBufferPoolEntityMap(b *testing.B) {
	pool := NewBufferPool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := pool.GetEntityMap()
		m[pattern.EntityEmail] = struct{}{}
		pool.PutEntityMap(m)
	}
}
