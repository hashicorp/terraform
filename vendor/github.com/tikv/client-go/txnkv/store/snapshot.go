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

package store

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"unsafe"

	pb "github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tikv/client-go/config"
	"github.com/tikv/client-go/key"
	"github.com/tikv/client-go/metrics"
	"github.com/tikv/client-go/retry"
	"github.com/tikv/client-go/rpc"
	"github.com/tikv/client-go/txnkv/kv"
)

// TiKVSnapshot supports read from TiKV.
type TiKVSnapshot struct {
	store *TiKVStore
	ts    uint64
	conf  *config.Config

	Priority     pb.CommandPri
	NotFillCache bool
	SyncLog      bool
	KeyOnly      bool
}

func newTiKVSnapshot(store *TiKVStore, ts uint64) *TiKVSnapshot {
	metrics.SnapshotCounter.Inc()

	return &TiKVSnapshot{
		store:    store,
		ts:       ts,
		conf:     store.GetConfig(),
		Priority: pb.CommandPri_Normal,
	}
}

// BatchGet gets all the keys' value from kv-server and returns a map contains key/value pairs.
// The map will not contain nonexistent keys.
func (s *TiKVSnapshot) BatchGet(ctx context.Context, keys []key.Key) (map[string][]byte, error) {
	m := make(map[string][]byte)
	if len(keys) == 0 {
		return m, nil
	}

	// We want [][]byte instead of []key.Key, use some magic to save memory.
	bytesKeys := *(*[][]byte)(unsafe.Pointer(&keys))
	bo := retry.NewBackoffer(ctx, retry.BatchGetMaxBackoff)

	// Create a map to collect key-values from region servers.
	var mu sync.Mutex
	err := s.batchGetKeysByRegions(bo, bytesKeys, func(k, v []byte) {
		if len(v) == 0 {
			return
		}
		mu.Lock()
		m[string(k)] = v
		mu.Unlock()
	})
	if err != nil {
		return nil, err
	}

	err = s.store.CheckVisibility(s.ts)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (s *TiKVSnapshot) batchGetKeysByRegions(bo *retry.Backoffer, keys [][]byte, collectF func(k, v []byte)) error {
	groups, _, err := s.store.regionCache.GroupKeysByRegion(bo, keys)
	if err != nil {
		return err
	}

	metrics.TxnRegionsNumHistogram.WithLabelValues("snapshot").Observe(float64(len(groups)))

	var batches []batchKeys
	for id, g := range groups {
		batches = appendBatchBySize(batches, id, g, func([]byte) int { return 1 }, s.conf.Txn.BatchGetSize)
	}

	if len(batches) == 0 {
		return nil
	}
	if len(batches) == 1 {
		return s.batchGetSingleRegion(bo, batches[0], collectF)
	}
	ch := make(chan error)
	for _, batch1 := range batches {
		batch := batch1
		go func() {
			backoffer, cancel := bo.Fork()
			defer cancel()
			ch <- s.batchGetSingleRegion(backoffer, batch, collectF)
		}()
	}
	for i := 0; i < len(batches); i++ {
		if e := <-ch; e != nil {
			log.Debugf("snapshot batchGet failed: %v, tid: %d", e, s.ts)
			err = e
		}
	}
	return err
}

func (s *TiKVSnapshot) batchGetSingleRegion(bo *retry.Backoffer, batch batchKeys, collectF func(k, v []byte)) error {
	sender := rpc.NewRegionRequestSender(s.store.GetRegionCache(), s.store.GetRPCClient())

	pending := batch.keys
	for {
		req := &rpc.Request{
			Type: rpc.CmdBatchGet,
			BatchGet: &pb.BatchGetRequest{
				Keys:    pending,
				Version: s.ts,
			},
			Context: pb.Context{
				Priority:     s.Priority,
				NotFillCache: s.NotFillCache,
			},
		}
		resp, err := sender.SendReq(bo, req, batch.region, s.conf.RPC.ReadTimeoutMedium)
		if err != nil {
			return err
		}
		regionErr, err := resp.GetRegionError()
		if err != nil {
			return err
		}
		if regionErr != nil {
			err = bo.Backoff(retry.BoRegionMiss, errors.New(regionErr.String()))
			if err != nil {
				return err
			}
			return s.batchGetKeysByRegions(bo, pending, collectF)
		}
		batchGetResp := resp.BatchGet
		if batchGetResp == nil {
			return errors.WithStack(rpc.ErrBodyMissing)
		}
		var (
			lockedKeys [][]byte
			locks      []*Lock
		)
		for _, pair := range batchGetResp.Pairs {
			keyErr := pair.GetError()
			if keyErr == nil {
				collectF(pair.GetKey(), pair.GetValue())
				continue
			}
			lock, err := extractLockFromKeyErr(keyErr, s.conf.Txn.DefaultLockTTL)
			if err != nil {
				return err
			}
			lockedKeys = append(lockedKeys, lock.Key)
			locks = append(locks, lock)
		}
		if len(lockedKeys) > 0 {
			ok, err := s.store.lockResolver.ResolveLocks(bo, locks)
			if err != nil {
				return err
			}
			if !ok {
				err = bo.Backoff(retry.BoTxnLockFast, errors.Errorf("batchGet lockedKeys: %d", len(lockedKeys)))
				if err != nil {
					return err
				}
			}
			pending = lockedKeys
			continue
		}
		return nil
	}
}

// Get gets the value for key k from snapshot.
func (s *TiKVSnapshot) Get(ctx context.Context, k key.Key) ([]byte, error) {
	val, err := s.get(retry.NewBackoffer(ctx, retry.GetMaxBackoff), k)
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return nil, kv.ErrNotExist
	}
	return val, nil
}

func (s *TiKVSnapshot) get(bo *retry.Backoffer, k key.Key) ([]byte, error) {
	sender := rpc.NewRegionRequestSender(s.store.GetRegionCache(), s.store.GetRPCClient())

	req := &rpc.Request{
		Type: rpc.CmdGet,
		Get: &pb.GetRequest{
			Key:     k,
			Version: s.ts,
		},
		Context: pb.Context{
			Priority:     s.Priority,
			NotFillCache: s.NotFillCache,
		},
	}
	for {
		loc, err := s.store.regionCache.LocateKey(bo, k)
		if err != nil {
			return nil, err
		}
		resp, err := sender.SendReq(bo, req, loc.Region, s.conf.RPC.ReadTimeoutShort)
		if err != nil {
			return nil, err
		}
		regionErr, err := resp.GetRegionError()
		if err != nil {
			return nil, err
		}
		if regionErr != nil {
			err = bo.Backoff(retry.BoRegionMiss, errors.New(regionErr.String()))
			if err != nil {
				return nil, err
			}
			continue
		}
		cmdGetResp := resp.Get
		if cmdGetResp == nil {
			return nil, errors.WithStack(rpc.ErrBodyMissing)
		}
		val := cmdGetResp.GetValue()
		if keyErr := cmdGetResp.GetError(); keyErr != nil {
			lock, err := extractLockFromKeyErr(keyErr, s.conf.Txn.DefaultLockTTL)
			if err != nil {
				return nil, err
			}
			ok, err := s.store.lockResolver.ResolveLocks(bo, []*Lock{lock})
			if err != nil {
				return nil, err
			}
			if !ok {
				err = bo.Backoff(retry.BoTxnLockFast, errors.New(keyErr.String()))
				if err != nil {
					return nil, err
				}
			}
			continue
		}
		return val, nil
	}
}

// Iter returns a list of key-value pair after `k`.
func (s *TiKVSnapshot) Iter(ctx context.Context, k key.Key, upperBound key.Key) (kv.Iterator, error) {
	scanner, err := newScanner(ctx, s, k, upperBound, s.conf.Txn.ScanBatchSize)
	return scanner, err
}

// IterReverse creates a reversed Iterator positioned on the first entry which key is less than k.
func (s *TiKVSnapshot) IterReverse(ctx context.Context, k key.Key) (kv.Iterator, error) {
	return nil, ErrNotImplemented
}

// SetPriority sets the priority of read requests.
func (s *TiKVSnapshot) SetPriority(priority int) {
	s.Priority = pb.CommandPri(priority)
}

func extractLockFromKeyErr(keyErr *pb.KeyError, defaultTTL uint64) (*Lock, error) {
	if locked := keyErr.GetLocked(); locked != nil {
		return NewLock(locked, defaultTTL), nil
	}
	if keyErr.Conflict != nil {
		err := errors.New(conflictToString(keyErr.Conflict))
		return nil, errors.WithMessage(err, TxnRetryableMark)
	}
	if keyErr.Retryable != "" {
		err := errors.Errorf("tikv restarts txn: %s", keyErr.GetRetryable())
		log.Debug(err)
		return nil, errors.WithMessage(err, TxnRetryableMark)
	}
	if keyErr.Abort != "" {
		err := errors.Errorf("tikv aborts txn: %s", keyErr.GetAbort())
		log.Warn(err)
		return nil, err
	}
	return nil, errors.Errorf("unexpected KeyError: %s", keyErr.String())
}

func conflictToString(conflict *pb.WriteConflict) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "WriteConflict: startTS=%d, conflictTS=%d, key=%q, primary=%q", conflict.StartTs, conflict.ConflictTs, conflict.Key, conflict.Primary)
	return buf.String()
}
