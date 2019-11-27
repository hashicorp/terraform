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

package config

import "time"

// Txn contains the configurations for transactional kv.
type Txn struct {
	// EntrySizeLimit is limit of single entry size (len(key) + len(value)).
	EntrySizeLimit int

	// EntryCountLimit is a limit of the number of entries in the MemBuffer.
	EntryCountLimit int

	// TotalSizeLimit is limit of the sum of all entry size.
	TotalSizeLimit int

	// MaxTimeUse is the max time a transaction can run.
	MaxTimeUse int

	// DefaultMembufCap is the default transaction membuf capability.
	DefaultMembufCap int

	// TiKV recommends each RPC packet should be less than ~1MB. We keep each
	// packet's Key+Value size below 16KB by default.
	CommitBatchSize int

	// ScanBatchSize is the limit of an iterator's scan request.
	ScanBatchSize int

	// BatchGetSize is the max number of keys in a BatchGet request.
	BatchGetSize int

	// By default, locks after 3000ms is considered unusual (the client created the
	// lock might be dead). Other client may cleanup this kind of lock.
	// For locks created recently, we will do backoff and retry.
	DefaultLockTTL uint64

	// The maximum value of a txn's lock TTL.
	MaxLockTTL uint64

	// ttl = ttlFactor * sqrt(writeSizeInMiB)
	TTLFactor int

	// ResolveCacheSize is max number of cached txn status.
	ResolveCacheSize int

	GcSavedSafePoint               string
	GcSafePointCacheInterval       time.Duration
	GcCPUTimeInaccuracyBound       time.Duration
	GcSafePointUpdateInterval      time.Duration
	GcSafePointQuickRepeatInterval time.Duration

	GCTimeout                 time.Duration
	UnsafeDestroyRangeTimeout time.Duration

	TsoSlowThreshold     time.Duration
	OracleUpdateInterval time.Duration

	Latch Latch
}

// DefaultTxn returns the default txn config.
func DefaultTxn() Txn {
	return Txn{
		EntrySizeLimit:                 6 * 1024 * 1024,
		EntryCountLimit:                300 * 1000,
		TotalSizeLimit:                 100 * 1024 * 1024,
		MaxTimeUse:                     590,
		DefaultMembufCap:               4 * 1024,
		CommitBatchSize:                16 * 1024,
		ScanBatchSize:                  256,
		BatchGetSize:                   5120,
		DefaultLockTTL:                 3000,
		MaxLockTTL:                     120000,
		TTLFactor:                      6000,
		ResolveCacheSize:               2048,
		GcSavedSafePoint:               "/tidb/store/gcworker/saved_safe_point",
		GcSafePointCacheInterval:       time.Second * 100,
		GcCPUTimeInaccuracyBound:       time.Second,
		GcSafePointUpdateInterval:      time.Second * 10,
		GcSafePointQuickRepeatInterval: time.Second,
		GCTimeout:                      5 * time.Minute,
		UnsafeDestroyRangeTimeout:      5 * time.Minute,
		TsoSlowThreshold:               30 * time.Millisecond,
		OracleUpdateInterval:           2 * time.Second,
		Latch:                          DefaultLatch(),
	}
}

// Latch is the configuration for local latch.
type Latch struct {
	// Enable it when there are lots of conflicts between transactions.
	Enable         bool
	Capacity       uint
	ExpireDuration time.Duration
	CheckInterval  time.Duration
	CheckCounter   int
	ListCount      int
	LockChanSize   int
}

// DefaultLatch returns the default Latch config.
func DefaultLatch() Latch {
	return Latch{
		Enable:         false,
		Capacity:       2048000,
		ExpireDuration: 2 * time.Minute,
		CheckInterval:  time.Minute,
		CheckCounter:   50000,
		ListCount:      5,
		LockChanSize:   100,
	}
}
