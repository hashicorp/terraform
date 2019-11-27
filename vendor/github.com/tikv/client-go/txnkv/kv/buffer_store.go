// Copyright 2015 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package kv

import (
	"context"

	"github.com/tikv/client-go/config"
	"github.com/tikv/client-go/key"
)

// BufferStore wraps a Retriever for read and a MemBuffer for buffered write.
// Common usage pattern:
//	bs := NewBufferStore(r) // use BufferStore to wrap a Retriever
//	// ...
//	// read/write on bs
//	// ...
//	bs.SaveTo(m)	        // save above operations to a Mutator
type BufferStore struct {
	MemBuffer
	r Retriever
}

// NewBufferStore creates a BufferStore using r for read.
func NewBufferStore(r Retriever, conf *config.Txn) *BufferStore {
	return &BufferStore{
		r:         r,
		MemBuffer: &lazyMemBuffer{conf: conf},
	}
}

// Reset resets s.MemBuffer.
func (s *BufferStore) Reset() {
	s.MemBuffer.Reset()
}

// SetCap sets the MemBuffer capability.
func (s *BufferStore) SetCap(cap int) {
	s.MemBuffer.SetCap(cap)
}

// Get implements the Retriever interface.
func (s *BufferStore) Get(ctx context.Context, k key.Key) ([]byte, error) {
	val, err := s.MemBuffer.Get(ctx, k)
	if IsErrNotFound(err) {
		val, err = s.r.Get(ctx, k)
	}
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return nil, ErrNotExist
	}
	return val, nil
}

// Iter implements the Retriever interface.
func (s *BufferStore) Iter(ctx context.Context, k key.Key, upperBound key.Key) (Iterator, error) {
	bufferIt, err := s.MemBuffer.Iter(ctx, k, upperBound)
	if err != nil {
		return nil, err
	}
	retrieverIt, err := s.r.Iter(ctx, k, upperBound)
	if err != nil {
		return nil, err
	}
	return NewUnionIter(ctx, bufferIt, retrieverIt, false)
}

// IterReverse implements the Retriever interface.
func (s *BufferStore) IterReverse(ctx context.Context, k key.Key) (Iterator, error) {
	bufferIt, err := s.MemBuffer.IterReverse(ctx, k)
	if err != nil {
		return nil, err
	}
	retrieverIt, err := s.r.IterReverse(ctx, k)
	if err != nil {
		return nil, err
	}
	return NewUnionIter(ctx, bufferIt, retrieverIt, true)
}

// WalkBuffer iterates all buffered kv pairs.
func (s *BufferStore) WalkBuffer(f func(k key.Key, v []byte) error) error {
	return WalkMemBuffer(s.MemBuffer, f)
}

// SaveTo saves all buffered kv pairs into a Mutator.
func (s *BufferStore) SaveTo(m Mutator) error {
	err := s.WalkBuffer(func(k key.Key, v []byte) error {
		if len(v) == 0 {
			return m.Delete(k)
		}
		return m.Set(k, v)
	})
	return err
}
