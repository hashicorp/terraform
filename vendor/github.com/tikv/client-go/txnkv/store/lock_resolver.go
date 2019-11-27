// Copyright 2016 PingCAP, Inc.
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
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tikv/client-go/config"
	"github.com/tikv/client-go/locate"
	"github.com/tikv/client-go/metrics"
	"github.com/tikv/client-go/retry"
	"github.com/tikv/client-go/rpc"
)

// LockResolver resolves locks and also caches resolved txn status.
type LockResolver struct {
	store *TiKVStore
	conf  *config.Config
	mu    struct {
		sync.RWMutex
		// resolved caches resolved txns (FIFO, txn id -> txnStatus).
		resolved       map[uint64]TxnStatus
		recentResolved *list.List
	}
}

func newLockResolver(store *TiKVStore) *LockResolver {
	r := &LockResolver{
		store: store,
		conf:  store.GetConfig(),
	}
	r.mu.resolved = make(map[uint64]TxnStatus)
	r.mu.recentResolved = list.New()
	return r
}

// NewLockResolver is exported for other pkg to use, suppress unused warning.
var _ = NewLockResolver

// NewLockResolver creates a LockResolver.
// It is exported for other pkg to use. For instance, binlog service needs
// to determine a transaction's commit state.
func NewLockResolver(ctx context.Context, etcdAddrs []string, conf config.Config) (*LockResolver, error) {
	s, err := NewStore(ctx, etcdAddrs, conf)
	if err != nil {
		return nil, err
	}

	return s.GetLockResolver(), nil
}

// TxnStatus represents a txn's final status. It should be Commit or Rollback.
type TxnStatus uint64

// IsCommitted returns true if the txn's final status is Commit.
func (s TxnStatus) IsCommitted() bool { return s > 0 }

// CommitTS returns the txn's commitTS. It is valid iff `IsCommitted` is true.
func (s TxnStatus) CommitTS() uint64 { return uint64(s) }

// Lock represents a lock from tikv server.
type Lock struct {
	Key     []byte
	Primary []byte
	TxnID   uint64
	TTL     uint64
}

// NewLock creates a new *Lock.
func NewLock(l *kvrpcpb.LockInfo, defaultTTL uint64) *Lock {
	ttl := l.GetLockTtl()
	if ttl == 0 {
		ttl = defaultTTL
	}
	return &Lock{
		Key:     l.GetKey(),
		Primary: l.GetPrimaryLock(),
		TxnID:   l.GetLockVersion(),
		TTL:     ttl,
	}
}

func (lr *LockResolver) saveResolved(txnID uint64, status TxnStatus) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if _, ok := lr.mu.resolved[txnID]; ok {
		return
	}
	lr.mu.resolved[txnID] = status
	lr.mu.recentResolved.PushBack(txnID)
	if len(lr.mu.resolved) > lr.conf.Txn.ResolveCacheSize {
		front := lr.mu.recentResolved.Front()
		delete(lr.mu.resolved, front.Value.(uint64))
		lr.mu.recentResolved.Remove(front)
	}
}

func (lr *LockResolver) getResolved(txnID uint64) (TxnStatus, bool) {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	s, ok := lr.mu.resolved[txnID]
	return s, ok
}

// BatchResolveLocks resolve locks in a batch
func (lr *LockResolver) BatchResolveLocks(bo *retry.Backoffer, locks []*Lock, loc locate.RegionVerID) (bool, error) {
	if len(locks) == 0 {
		return true, nil
	}

	metrics.LockResolverCounter.WithLabelValues("batch_resolve").Inc()

	var expiredLocks []*Lock
	for _, l := range locks {
		if lr.store.GetOracle().IsExpired(l.TxnID, l.TTL) {
			metrics.LockResolverCounter.WithLabelValues("expired").Inc()
			expiredLocks = append(expiredLocks, l)
		} else {
			metrics.LockResolverCounter.WithLabelValues("not_expired").Inc()
		}
	}
	if len(expiredLocks) != len(locks) {
		log.Errorf("BatchResolveLocks: get %d Locks, but only %d are expired, maybe safe point is wrong!", len(locks), len(expiredLocks))
		return false, nil
	}

	startTime := time.Now()
	txnInfos := make(map[uint64]uint64)
	for _, l := range expiredLocks {
		if _, ok := txnInfos[l.TxnID]; ok {
			continue
		}

		status, err := lr.getTxnStatus(bo, l.TxnID, l.Primary)
		if err != nil {
			return false, err
		}
		txnInfos[l.TxnID] = uint64(status)
	}
	log.Infof("BatchResolveLocks: it took %v to lookup %v txn status", time.Since(startTime), len(txnInfos))

	var listTxnInfos []*kvrpcpb.TxnInfo
	for txnID, status := range txnInfos {
		listTxnInfos = append(listTxnInfos, &kvrpcpb.TxnInfo{
			Txn:    txnID,
			Status: status,
		})
	}

	req := &rpc.Request{
		Type: rpc.CmdResolveLock,
		ResolveLock: &kvrpcpb.ResolveLockRequest{
			TxnInfos: listTxnInfos,
		},
	}
	startTime = time.Now()
	resp, err := lr.store.SendReq(bo, req, loc, lr.conf.RPC.ReadTimeoutShort)
	if err != nil {
		return false, err
	}

	regionErr, err := resp.GetRegionError()
	if err != nil {
		return false, err
	}

	if regionErr != nil {
		err = bo.Backoff(retry.BoRegionMiss, errors.New(regionErr.String()))
		if err != nil {
			return false, err
		}
		return false, nil
	}

	cmdResp := resp.ResolveLock
	if cmdResp == nil {
		return false, errors.WithStack(rpc.ErrBodyMissing)
	}
	if keyErr := cmdResp.GetError(); keyErr != nil {
		return false, errors.Errorf("unexpected resolve err: %s", keyErr)
	}

	log.Infof("BatchResolveLocks: it took %v to resolve %v locks in a batch.", time.Since(startTime), len(expiredLocks))
	return true, nil
}

// ResolveLocks tries to resolve Locks. The resolving process is in 3 steps:
// 1) Use the `lockTTL` to pick up all expired locks. Only locks that are too
//    old are considered orphan locks and will be handled later. If all locks
//    are expired then all locks will be resolved so the returned `ok` will be
//    true, otherwise caller should sleep a while before retry.
// 2) For each lock, query the primary key to get txn(which left the lock)'s
//    commit status.
// 3) Send `ResolveLock` cmd to the lock's region to resolve all locks belong to
//    the same transaction.
func (lr *LockResolver) ResolveLocks(bo *retry.Backoffer, locks []*Lock) (ok bool, err error) {
	if len(locks) == 0 {
		return true, nil
	}

	metrics.LockResolverCounter.WithLabelValues("resolve").Inc()

	var expiredLocks []*Lock
	for _, l := range locks {
		if lr.store.GetOracle().IsExpired(l.TxnID, l.TTL) {
			metrics.LockResolverCounter.WithLabelValues("expired").Inc()
			expiredLocks = append(expiredLocks, l)
		} else {
			metrics.LockResolverCounter.WithLabelValues("not_expired").Inc()
		}
	}
	if len(expiredLocks) == 0 {
		return false, nil
	}

	// TxnID -> []Region, record resolved Regions.
	// TODO: Maybe put it in LockResolver and share by all txns.
	cleanTxns := make(map[uint64]map[locate.RegionVerID]struct{})
	for _, l := range expiredLocks {
		status, err := lr.getTxnStatus(bo, l.TxnID, l.Primary)
		if err != nil {
			return false, err
		}

		cleanRegions := cleanTxns[l.TxnID]
		if cleanRegions == nil {
			cleanRegions = make(map[locate.RegionVerID]struct{})
			cleanTxns[l.TxnID] = cleanRegions
		}

		err = lr.resolveLock(bo, l, status, cleanRegions)
		if err != nil {
			return false, err
		}
	}
	return len(expiredLocks) == len(locks), nil
}

// GetTxnStatus queries tikv-server for a txn's status (commit/rollback).
// If the primary key is still locked, it will launch a Rollback to abort it.
// To avoid unnecessarily aborting too many txns, it is wiser to wait a few
// seconds before calling it after Prewrite.
func (lr *LockResolver) GetTxnStatus(ctx context.Context, txnID uint64, primary []byte) (TxnStatus, error) {
	bo := retry.NewBackoffer(ctx, retry.CleanupMaxBackoff)
	return lr.getTxnStatus(bo, txnID, primary)
}

func (lr *LockResolver) getTxnStatus(bo *retry.Backoffer, txnID uint64, primary []byte) (TxnStatus, error) {
	if s, ok := lr.getResolved(txnID); ok {
		return s, nil
	}

	metrics.LockResolverCounter.WithLabelValues("query_txn_status").Inc()

	var status TxnStatus
	req := &rpc.Request{
		Type: rpc.CmdCleanup,
		Cleanup: &kvrpcpb.CleanupRequest{
			Key:          primary,
			StartVersion: txnID,
		},
	}
	for {
		loc, err := lr.store.GetRegionCache().LocateKey(bo, primary)
		if err != nil {
			return status, err
		}
		resp, err := lr.store.SendReq(bo, req, loc.Region, lr.conf.RPC.ReadTimeoutShort)
		if err != nil {
			return status, err
		}
		regionErr, err := resp.GetRegionError()
		if err != nil {
			return status, err
		}
		if regionErr != nil {
			err = bo.Backoff(retry.BoRegionMiss, errors.New(regionErr.String()))
			if err != nil {
				return status, err
			}
			continue
		}
		cmdResp := resp.Cleanup
		if cmdResp == nil {
			return status, errors.WithStack(rpc.ErrBodyMissing)
		}
		if keyErr := cmdResp.GetError(); keyErr != nil {
			err = errors.Errorf("unexpected cleanup err: %s, tid: %v", keyErr, txnID)
			log.Error(err)
			return status, err
		}
		if cmdResp.CommitVersion != 0 {
			status = TxnStatus(cmdResp.GetCommitVersion())
			metrics.LockResolverCounter.WithLabelValues("query_txn_status_committed").Inc()
		} else {
			metrics.LockResolverCounter.WithLabelValues("query_txn_status_rolled_back").Inc()
		}
		lr.saveResolved(txnID, status)
		return status, nil
	}
}

func (lr *LockResolver) resolveLock(bo *retry.Backoffer, l *Lock, status TxnStatus, cleanRegions map[locate.RegionVerID]struct{}) error {
	metrics.LockResolverCounter.WithLabelValues("query_resolve_locks").Inc()
	for {
		loc, err := lr.store.GetRegionCache().LocateKey(bo, l.Key)
		if err != nil {
			return err
		}
		if _, ok := cleanRegions[loc.Region]; ok {
			return nil
		}
		req := &rpc.Request{
			Type: rpc.CmdResolveLock,
			ResolveLock: &kvrpcpb.ResolveLockRequest{
				StartVersion: l.TxnID,
			},
		}
		if status.IsCommitted() {
			req.ResolveLock.CommitVersion = status.CommitTS()
		}
		resp, err := lr.store.SendReq(bo, req, loc.Region, lr.conf.RPC.ReadTimeoutShort)
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
			continue
		}
		cmdResp := resp.ResolveLock
		if cmdResp == nil {
			return errors.WithStack(rpc.ErrBodyMissing)
		}
		if keyErr := cmdResp.GetError(); keyErr != nil {
			err = errors.Errorf("unexpected resolve err: %s, lock: %v", keyErr, l)
			log.Error(err)
			return err
		}
		cleanRegions[loc.Region] = struct{}{}
		return nil
	}
}
