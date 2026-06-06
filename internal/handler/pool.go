package handler

import (
	"bytes"
	"sync"

	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
)

// BufferPool manages reusable buffers to reduce GC pressure on tail requests
type BufferPool struct {
	responseBuffers  sync.Pool // *bytes.Buffer for response buffering
	entityMapBuffers sync.Pool // *map[pattern.Entity]struct{} for entity tracking
}

// NewBufferPool creates a new buffer pool
func NewBufferPool() *BufferPool {
	return &BufferPool{
		responseBuffers: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
		entityMapBuffers: sync.Pool{
			New: func() interface{} {
				m := make(map[pattern.Entity]struct{}, 16)
				return &m
			},
		},
	}
}

// GetResponseBuffer retrieves a response buffer from the pool
func (bp *BufferPool) GetResponseBuffer() *bytes.Buffer {
	buf := bp.responseBuffers.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutResponseBuffer returns a response buffer to the pool
func (bp *BufferPool) PutResponseBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bp.responseBuffers.Put(buf)
}

// GetEntityMap retrieves an entity map from the pool
func (bp *BufferPool) GetEntityMap() map[pattern.Entity]struct{} {
	return *bp.entityMapBuffers.Get().(*map[pattern.Entity]struct{})
}

// PutEntityMap returns an entity map to the pool
func (bp *BufferPool) PutEntityMap(m map[pattern.Entity]struct{}) {
	for k := range m {
		delete(m, k)
	}
	bp.entityMapBuffers.Put(&m)
}
