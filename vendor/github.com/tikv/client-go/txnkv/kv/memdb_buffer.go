// Copyright 2015 PingCAP, Inc.
//
// Copyright 2015 Wenbin Xiao
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
	"fmt"

	"github.com/pingcap/goleveldb/leveldb"
	"github.com/pingcap/goleveldb/leveldb/comparer"
	"github.com/pingcap/goleveldb/leveldb/iterator"
	"github.com/pingcap/goleveldb/leveldb/memdb"
	"github.com/pingcap/goleveldb/leveldb/util"
	"github.com/pkg/errors"
	"github.com/tikv/client-go/config"
	"github.com/tikv/client-go/key"
)

// memDbBuffer implements the MemBuffer interface.
type memDbBuffer struct {
	db              *memdb.DB
	entrySizeLimit  int
	bufferLenLimit  int
	bufferSizeLimit int
}

type memDbIter struct {
	iter    iterator.Iterator
	reverse bool
}

// NewMemDbBuffer creates a new memDbBuffer.
func NewMemDbBuffer(conf *config.Txn, cap int) MemBuffer {
	if cap <= 0 {
		cap = conf.DefaultMembufCap
	}
	return &memDbBuffer{
		db:              memdb.New(comparer.DefaultComparer, cap),
		entrySizeLimit:  conf.EntrySizeLimit,
		bufferLenLimit:  conf.EntryCountLimit,
		bufferSizeLimit: conf.TotalSizeLimit,
	}
}

// Iter creates an Iterator.
func (m *memDbBuffer) Iter(ctx context.Context, k key.Key, upperBound key.Key) (Iterator, error) {
	i := &memDbIter{iter: m.db.NewIterator(&util.Range{Start: []byte(k), Limit: []byte(upperBound)}), reverse: false}

	err := i.Next(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return i, nil
}

func (m *memDbBuffer) SetCap(cap int) {

}

func (m *memDbBuffer) IterReverse(ctx context.Context, k key.Key) (Iterator, error) {
	var i *memDbIter
	if k == nil {
		i = &memDbIter{iter: m.db.NewIterator(&util.Range{}), reverse: true}
	} else {
		i = &memDbIter{iter: m.db.NewIterator(&util.Range{Limit: []byte(k)}), reverse: true}
	}
	i.iter.Last()
	return i, nil
}

// Get returns the value associated with key.
func (m *memDbBuffer) Get(ctx context.Context, k key.Key) ([]byte, error) {
	v, err := m.db.Get(k)
	if err == leveldb.ErrNotFound {
		return nil, ErrNotExist
	}
	return v, nil
}

// Set associates key with value.
func (m *memDbBuffer) Set(k key.Key, v []byte) error {
	if len(v) == 0 {
		return errors.WithStack(ErrCannotSetNilValue)
	}
	if len(k)+len(v) > m.entrySizeLimit {
		return errors.WithMessage(ErrEntryTooLarge, fmt.Sprintf("entry too large, size: %d", len(k)+len(v)))
	}

	err := m.db.Put(k, v)
	if m.Size() > m.bufferSizeLimit {
		return errors.WithMessage(ErrTxnTooLarge, fmt.Sprintf("transaction too large, size:%d", m.Size()))
	}
	if m.Len() > int(m.bufferLenLimit) {
		return errors.WithMessage(ErrTxnTooLarge, fmt.Sprintf("transaction too large, size:%d", m.Size()))
	}
	return errors.WithStack(err)
}

// Delete removes the entry from buffer with provided key.
func (m *memDbBuffer) Delete(k key.Key) error {
	err := m.db.Put(k, nil)
	return errors.WithStack(err)
}

// Size returns sum of keys and values length.
func (m *memDbBuffer) Size() int {
	return m.db.Size()
}

// Len returns the number of entries in the DB.
func (m *memDbBuffer) Len() int {
	return m.db.Len()
}

// Reset cleanup the MemBuffer.
func (m *memDbBuffer) Reset() {
	m.db.Reset()
}

// Next implements the Iterator Next.
func (i *memDbIter) Next(context.Context) error {
	if i.reverse {
		i.iter.Prev()
	} else {
		i.iter.Next()
	}
	return nil
}

// Valid implements the Iterator Valid.
func (i *memDbIter) Valid() bool {
	return i.iter.Valid()
}

// Key implements the Iterator Key.
func (i *memDbIter) Key() key.Key {
	return i.iter.Key()
}

// Value implements the Iterator Value.
func (i *memDbIter) Value() []byte {
	return i.iter.Value()
}

// Close Implements the Iterator Close.
func (i *memDbIter) Close() {
	i.iter.Release()
}

// WalkMemBuffer iterates all buffered kv pairs in memBuf
func WalkMemBuffer(memBuf MemBuffer, f func(k key.Key, v []byte) error) error {
	iter, err := memBuf.Iter(context.Background(), nil, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	defer iter.Close()
	for iter.Valid() {
		if err = f(iter.Key(), iter.Value()); err != nil {
			return errors.WithStack(err)
		}
		err = iter.Next(context.Background())
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
