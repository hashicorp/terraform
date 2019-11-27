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

package txnkv

import (
	"context"
	"fmt"
	"time"

	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"github.com/tikv/client-go/key"
	"github.com/tikv/client-go/metrics"
	"github.com/tikv/client-go/txnkv/kv"
	"github.com/tikv/client-go/txnkv/store"
)

// Transaction is a key-value transaction.
type Transaction struct {
	tikvStore *store.TiKVStore
	snapshot  *store.TiKVSnapshot
	us        kv.UnionStore

	startTS   uint64
	startTime time.Time // Monotonic timestamp for recording txn time consuming.
	commitTS  uint64
	valid     bool
	lockKeys  [][]byte
}

func newTransaction(tikvStore *store.TiKVStore, ts uint64) *Transaction {
	metrics.TxnCounter.Inc()

	snapshot := tikvStore.GetSnapshot(ts)
	us := kv.NewUnionStore(&tikvStore.GetConfig().Txn, snapshot)
	return &Transaction{
		tikvStore: tikvStore,
		snapshot:  snapshot,
		us:        us,

		startTS:   ts,
		startTime: time.Now(),
		valid:     true,
	}
}

// Get implements transaction interface.
// kv.IsErrNotFound can be used to check the error is a not found error.
func (txn *Transaction) Get(ctx context.Context, k key.Key) ([]byte, error) {
	start := time.Now()
	defer func() { metrics.TxnCmdHistogram.WithLabelValues("get").Observe(time.Since(start).Seconds()) }()

	ret, err := txn.us.Get(ctx, k)
	if err != nil {
		return nil, err
	}

	err = txn.tikvStore.CheckVisibility(txn.startTS)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// BatchGet gets a batch of values from TiKV server.
func (txn *Transaction) BatchGet(ctx context.Context, keys []key.Key) (map[string][]byte, error) {
	start := time.Now()
	defer func() { metrics.TxnCmdHistogram.WithLabelValues("batch_get").Observe(time.Since(start).Seconds()) }()

	if txn.IsReadOnly() {
		return txn.snapshot.BatchGet(ctx, keys)
	}
	bufferValues := make([][]byte, len(keys))
	shrinkKeys := make([]key.Key, 0, len(keys))
	for i, key := range keys {
		val, err := txn.us.GetMemBuffer().Get(ctx, key)
		if kv.IsErrNotFound(err) {
			shrinkKeys = append(shrinkKeys, key)
			continue
		}
		if err != nil {
			return nil, err
		}
		if len(val) != 0 {
			bufferValues[i] = val
		}
	}
	storageValues, err := txn.snapshot.BatchGet(ctx, shrinkKeys)
	if err != nil {
		return nil, err
	}
	for i, key := range keys {
		if bufferValues[i] == nil {
			continue
		}
		storageValues[string(key)] = bufferValues[i]
	}
	return storageValues, nil
}

// Set sets the value for key k as v into kv store.
func (txn *Transaction) Set(k key.Key, v []byte) error {
	start := time.Now()
	defer func() { metrics.TxnCmdHistogram.WithLabelValues("set").Observe(time.Since(start).Seconds()) }()
	return txn.us.Set(k, v)
}

func (txn *Transaction) String() string {
	return fmt.Sprintf("txn-%d", txn.startTS)
}

// Iter creates an Iterator positioned on the first entry that k <= entry's key.
// If such entry is not found, it returns an invalid Iterator with no error.
// It yields only keys that < upperBound. If upperBound is nil, it means the upperBound is unbounded.
// The Iterator must be closed after use.
func (txn *Transaction) Iter(ctx context.Context, k key.Key, upperBound key.Key) (kv.Iterator, error) {
	start := time.Now()
	defer func() { metrics.TxnCmdHistogram.WithLabelValues("iter").Observe(time.Since(start).Seconds()) }()

	return txn.us.Iter(ctx, k, upperBound)
}

// IterReverse creates a reversed Iterator positioned on the first entry which key is less than k.
func (txn *Transaction) IterReverse(ctx context.Context, k key.Key) (kv.Iterator, error) {
	start := time.Now()
	defer func() { metrics.TxnCmdHistogram.WithLabelValues("iter_reverse").Observe(time.Since(start).Seconds()) }()
	return txn.us.IterReverse(ctx, k)
}

// IsReadOnly returns if there are pending key-value to commit in the transaction.
func (txn *Transaction) IsReadOnly() bool {
	return txn.us.GetMemBuffer().Len() == 0 && len(txn.lockKeys) == 0
}

// Delete removes the entry for key k from kv store.
func (txn *Transaction) Delete(k key.Key) error {
	start := time.Now()
	defer func() { metrics.TxnCmdHistogram.WithLabelValues("delete").Observe(time.Since(start).Seconds()) }()
	return txn.us.Delete(k)
}

// SetOption sets an option with a value, when val is nil, uses the default
// value of this option.
func (txn *Transaction) SetOption(opt kv.Option, val interface{}) {
	txn.us.SetOption(opt, val)
	switch opt {
	case kv.Priority:
		txn.snapshot.SetPriority(val.(int))
	case kv.NotFillCache:
		txn.snapshot.NotFillCache = val.(bool)
	case kv.SyncLog:
		txn.snapshot.SyncLog = val.(bool)
	case kv.KeyOnly:
		txn.snapshot.KeyOnly = val.(bool)
	}
}

// DelOption deletes an option.
func (txn *Transaction) DelOption(opt kv.Option) {
	txn.us.DelOption(opt)
}

func (txn *Transaction) close() {
	txn.valid = false
}

// Commit commits the transaction operations to KV store.
func (txn *Transaction) Commit(ctx context.Context) error {
	if !txn.valid {
		return kv.ErrInvalidTxn
	}
	defer txn.close()

	// gofail: var mockCommitError bool
	// if mockCommitError && kv.IsMockCommitErrorEnable() {
	//  kv.MockCommitErrorDisable()
	//	return errors.New("mock commit error")
	// }

	start := time.Now()
	defer func() {
		metrics.TxnCmdHistogram.WithLabelValues("commit").Observe(time.Since(start).Seconds())
		metrics.TxnHistogram.Observe(time.Since(txn.startTime).Seconds())
	}()

	mutations := make(map[string]*kvrpcpb.Mutation)
	err := txn.us.WalkBuffer(func(k key.Key, v []byte) error {
		op := kvrpcpb.Op_Put
		if c := txn.us.LookupConditionPair(k); c != nil && c.ShouldNotExist() {
			op = kvrpcpb.Op_Insert
		}
		if len(v) == 0 {
			op = kvrpcpb.Op_Del
		}
		mutations[string(k)] = &kvrpcpb.Mutation{
			Op:    op,
			Key:   k,
			Value: v,
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, lockKey := range txn.lockKeys {
		if _, ok := mutations[string(lockKey)]; !ok {
			mutations[string(lockKey)] = &kvrpcpb.Mutation{
				Op:  kvrpcpb.Op_Lock,
				Key: lockKey,
			}
		}
	}
	if len(mutations) == 0 {
		return nil
	}

	committer, err := store.NewTxnCommitter(txn.tikvStore, txn.startTS, txn.startTime, mutations)
	if err != nil || committer == nil {
		return err
	}

	// latches disabled
	if txn.tikvStore.GetTxnLatches() == nil {
		err = committer.Execute(ctx)
		log.Debug("[kv]", txn.startTS, " txnLatches disabled, 2pc directly:", err)
		return err
	}

	// latches enabled
	// for transactions which need to acquire latches
	start = time.Now()
	lock := txn.tikvStore.GetTxnLatches().Lock(txn.startTS, committer.GetKeys())
	localLatchTime := time.Since(start)
	if localLatchTime > 0 {
		metrics.LocalLatchWaitTimeHistogram.Observe(localLatchTime.Seconds())
	}
	defer txn.tikvStore.GetTxnLatches().UnLock(lock)
	if lock.IsStale() {
		err = errors.Errorf("startTS %d is stale", txn.startTS)
		return errors.WithMessage(err, store.TxnRetryableMark)
	}
	err = committer.Execute(ctx)
	if err == nil {
		lock.SetCommitTS(committer.GetCommitTS())
	}
	log.Debug("[kv]", txn.startTS, " txnLatches enabled while txn retryable:", err)
	return err
}

// Rollback undoes the transaction operations to KV store.
func (txn *Transaction) Rollback() error {
	if !txn.valid {
		return kv.ErrInvalidTxn
	}
	start := time.Now()
	defer func() {
		metrics.TxnCmdHistogram.WithLabelValues("rollback").Observe(time.Since(start).Seconds())
		metrics.TxnHistogram.Observe(time.Since(txn.startTime).Seconds())
	}()
	txn.close()
	log.Debugf("[kv] Rollback txn %d", txn.startTS)

	return nil
}

// LockKeys tries to lock the entries with the keys in KV store.
func (txn *Transaction) LockKeys(keys ...key.Key) error {
	start := time.Now()
	defer func() { metrics.TxnCmdHistogram.WithLabelValues("lock_keys").Observe(time.Since(start).Seconds()) }()
	for _, key := range keys {
		txn.lockKeys = append(txn.lockKeys, key)
	}
	return nil
}

// Valid returns if the transaction is valid.
// A transaction becomes invalid after commit or rollback.
func (txn *Transaction) Valid() bool {
	return txn.valid
}

// Len returns the count of key-value pairs in the transaction's memory buffer.
func (txn *Transaction) Len() int {
	return txn.us.Len()
}

// Size returns the length (in bytes) of the transaction's memory buffer.
func (txn *Transaction) Size() int {
	return txn.us.Size()
}
