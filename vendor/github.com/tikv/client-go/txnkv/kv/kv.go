// Copyright 2019 PingCAP, Inc.
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

	"github.com/tikv/client-go/key"
)

// Priority value for transaction priority.
const (
	PriorityNormal = iota
	PriorityLow
	PriorityHigh
)

// IsoLevel is the transaction's isolation level.
type IsoLevel int

const (
	// SI stands for 'snapshot isolation'.
	SI IsoLevel = iota
	// RC stands for 'read committed'.
	RC
)

// Retriever is the interface wraps the basic Get and Seek methods.
type Retriever interface {
	// Get gets the value for key k from kv store.
	// If corresponding kv pair does not exist, it returns nil and ErrNotExist.
	Get(ctx context.Context, k key.Key) ([]byte, error)
	// Iter creates an Iterator positioned on the first entry that k <= entry's key.
	// If such entry is not found, it returns an invalid Iterator with no error.
	// It yields only keys that < upperBound. If upperBound is nil, it means the upperBound is unbounded.
	// The Iterator must be closed after use.
	Iter(ctx context.Context, k key.Key, upperBound key.Key) (Iterator, error)

	// IterReverse creates a reversed Iterator positioned on the first entry which key is less than k.
	// The returned iterator will iterate from greater key to smaller key.
	// If k is nil, the returned iterator will be positioned at the last key.
	// TODO: Add lower bound limit
	IterReverse(ctx context.Context, k key.Key) (Iterator, error)
}

// Mutator is the interface wraps the basic Set and Delete methods.
type Mutator interface {
	// Set sets the value for key k as v into kv store.
	// v must NOT be nil or empty, otherwise it returns ErrCannotSetNilValue.
	Set(k key.Key, v []byte) error
	// Delete removes the entry for key k from kv store.
	Delete(k key.Key) error
}

// RetrieverMutator is the interface that groups Retriever and Mutator interfaces.
type RetrieverMutator interface {
	Retriever
	Mutator
}

// MemBuffer is an in-memory kv collection, can be used to buffer write operations.
type MemBuffer interface {
	RetrieverMutator
	// Size returns sum of keys and values length.
	Size() int
	// Len returns the number of entries in the DB.
	Len() int
	// Reset cleanup the MemBuffer
	Reset()
	// SetCap sets the MemBuffer capability, to reduce memory allocations.
	// Please call it before you use the MemBuffer, otherwise it will not works.
	SetCap(cap int)
}

// Snapshot defines the interface for the snapshot fetched from KV store.
type Snapshot interface {
	Retriever
	// BatchGet gets a batch of values from snapshot.
	BatchGet(ctx context.Context, keys []key.Key) (map[string][]byte, error)
	// SetPriority snapshot set the priority
	SetPriority(priority int)
}

// Iterator is the interface for a iterator on KV store.
type Iterator interface {
	Valid() bool
	Key() key.Key
	Value() []byte
	Next(context.Context) error
	Close()
}

// Transaction options
const (
	// PresumeKeyNotExists indicates that when dealing with a Get operation but failing to read data from cache,
	// we presume that the key does not exist in Store. The actual existence will be checked before the
	// transaction's commit.
	// This option is an optimization for frequent checks during a transaction, e.g. batch inserts.
	PresumeKeyNotExists Option = iota + 1
	// PresumeKeyNotExistsError is the option key for error.
	// When PresumeKeyNotExists is set and condition is not match, should throw the error.
	PresumeKeyNotExistsError
	// BinlogInfo contains the binlog data and client.
	BinlogInfo
	// SchemaChecker is used for checking schema-validity.
	SchemaChecker
	// IsolationLevel sets isolation level for current transaction. The default level is SI.
	IsolationLevel
	// Priority marks the priority of this transaction.
	Priority
	// NotFillCache makes this request do not touch the LRU cache of the underlying storage.
	NotFillCache
	// SyncLog decides whether the WAL(write-ahead log) of this request should be synchronized.
	SyncLog
	// KeyOnly retrieve only keys, it can be used in scan now.
	KeyOnly
)
