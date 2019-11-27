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
	"bytes"
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tikv/client-go/config"
	"github.com/tikv/client-go/metrics"
	"github.com/tikv/client-go/retry"
	"github.com/tikv/client-go/rpc"
	"github.com/tikv/client-go/txnkv/kv"
)

type commitAction int

const (
	actionPrewrite commitAction = 1
	actionCommit   commitAction = 2
	actionCleanup  commitAction = 3
)

func (ca commitAction) String() string {
	switch ca {
	case actionPrewrite:
		return "prewrite"
	case actionCommit:
		return "commit"
	case actionCleanup:
		return "cleanup"
	}
	return "unknown"
}

// MetricsTag returns detail tag for metrics.
func (ca commitAction) MetricsTag() string {
	return "2pc_" + ca.String()
}

// TxnCommitter executes a two-phase commit protocol.
type TxnCommitter struct {
	Priority pb.CommandPri
	SyncLog  bool
	ConnID   uint64 // ConnID is used for log.

	store     *TiKVStore
	conf      *config.Config
	startTS   uint64
	keys      [][]byte
	mutations map[string]*pb.Mutation
	lockTTL   uint64
	commitTS  uint64
	mu        struct {
		sync.RWMutex
		committed       bool
		undeterminedErr error // undeterminedErr saves the rpc error we encounter when commit primary key.
	}
	cleanWg sync.WaitGroup
	// maxTxnTimeUse represents max time a Txn may use (in ms) from its startTS to commitTS.
	// We use it to guarantee GC worker will not influence any active txn. The value
	// should be less than GC life time.
	maxTxnTimeUse uint64
	detail        CommitDetails
}

// NewTxnCommitter creates a TxnCommitter.
func NewTxnCommitter(store *TiKVStore, startTS uint64, startTime time.Time, mutations map[string]*pb.Mutation) (*TxnCommitter, error) {
	var (
		keys    [][]byte
		size    int
		putCnt  int
		delCnt  int
		lockCnt int
	)

	conf := store.GetConfig()
	for key, mut := range mutations {
		switch mut.Op {
		case pb.Op_Put, pb.Op_Insert:
			putCnt++
		case pb.Op_Del:
			delCnt++
		case pb.Op_Lock:
			lockCnt++
		}
		keys = append(keys, []byte(key))
		entrySize := len(mut.Key) + len(mut.Value)
		if entrySize > conf.Txn.EntrySizeLimit {
			return nil, kv.ErrEntryTooLarge
		}
		size += entrySize
	}

	if putCnt == 0 && delCnt == 0 {
		return nil, nil
	}

	if len(keys) > int(conf.Txn.EntryCountLimit) || size > conf.Txn.TotalSizeLimit {
		return nil, kv.ErrTxnTooLarge
	}

	// Convert from sec to ms
	maxTxnTimeUse := uint64(conf.Txn.MaxTimeUse) * 1000

	metrics.TxnWriteKVCountHistogram.Observe(float64(len(keys)))
	metrics.TxnWriteSizeHistogram.Observe(float64(size))
	return &TxnCommitter{
		store:         store,
		conf:          conf,
		startTS:       startTS,
		keys:          keys,
		mutations:     mutations,
		lockTTL:       txnLockTTL(conf, startTime, size),
		maxTxnTimeUse: maxTxnTimeUse,
		detail:        CommitDetails{WriteSize: size, WriteKeys: len(keys)},
	}, nil
}

func (c *TxnCommitter) primary() []byte {
	return c.keys[0]
}

const bytesPerMiB = 1024 * 1024

func txnLockTTL(conf *config.Config, startTime time.Time, txnSize int) uint64 {
	// Increase lockTTL for large transactions.
	// The formula is `ttl = ttlFactor * sqrt(sizeInMiB)`.
	// When writeSize is less than 256KB, the base ttl is defaultTTL (3s);
	// When writeSize is 1MiB, 100MiB, or 400MiB, ttl is 6s, 60s, 120s correspondingly;
	lockTTL := conf.Txn.DefaultLockTTL
	if txnSize >= conf.Txn.CommitBatchSize {
		sizeMiB := float64(txnSize) / bytesPerMiB
		lockTTL = uint64(float64(conf.Txn.TTLFactor) * math.Sqrt(sizeMiB))
		if lockTTL < conf.Txn.DefaultLockTTL {
			lockTTL = conf.Txn.DefaultLockTTL
		}
		if lockTTL > conf.Txn.MaxLockTTL {
			lockTTL = conf.Txn.MaxLockTTL
		}
	}

	// Increase lockTTL by the transaction's read time.
	// When resolving a lock, we compare current ts and startTS+lockTTL to decide whether to clean up. If a txn
	// takes a long time to read, increasing its TTL will help to prevent it from been aborted soon after prewrite.
	elapsed := time.Since(startTime) / time.Millisecond
	return lockTTL + uint64(elapsed)
}

// doActionOnKeys groups keys into primary batch and secondary batches, if primary batch exists in the key,
// it does action on primary batch first, then on secondary batches. If action is commit, secondary batches
// is done in background goroutine.
func (c *TxnCommitter) doActionOnKeys(bo *retry.Backoffer, action commitAction, keys [][]byte) error {
	if len(keys) == 0 {
		return nil
	}
	groups, firstRegion, err := c.store.GetRegionCache().GroupKeysByRegion(bo, keys)
	if err != nil {
		return err
	}

	metrics.TxnRegionsNumHistogram.WithLabelValues(action.MetricsTag()).Observe(float64(len(groups)))

	var batches []batchKeys
	var sizeFunc = c.keySize
	if action == actionPrewrite {
		sizeFunc = c.keyValueSize
		atomic.AddInt32(&c.detail.PrewriteRegionNum, int32(len(groups)))
	}
	// Make sure the group that contains primary key goes first.
	commitBatchSize := c.conf.Txn.CommitBatchSize
	batches = appendBatchBySize(batches, firstRegion, groups[firstRegion], sizeFunc, commitBatchSize)
	delete(groups, firstRegion)
	for id, g := range groups {
		batches = appendBatchBySize(batches, id, g, sizeFunc, commitBatchSize)
	}

	firstIsPrimary := bytes.Equal(keys[0], c.primary())
	if firstIsPrimary && (action == actionCommit || action == actionCleanup) {
		// primary should be committed/cleanup first
		err = c.doActionOnBatches(bo, action, batches[:1])
		if err != nil {
			return err
		}
		batches = batches[1:]
	}
	if action == actionCommit {
		// Commit secondary batches in background goroutine to reduce latency.
		// The backoffer instance is created outside of the goroutine to avoid
		// potencial data race in unit test since `CommitMaxBackoff` will be updated
		// by test suites.
		secondaryBo := retry.NewBackoffer(context.Background(), retry.CommitMaxBackoff)
		go func() {
			e := c.doActionOnBatches(secondaryBo, action, batches)
			if e != nil {
				log.Debugf("con:%d 2PC async doActionOnBatches %s err: %v", c.ConnID, action, e)
				metrics.SecondaryLockCleanupFailureCounter.WithLabelValues("commit").Inc()
			}
		}()
	} else {
		err = c.doActionOnBatches(bo, action, batches)
	}
	return err
}

// doActionOnBatches does action to batches in parallel.
func (c *TxnCommitter) doActionOnBatches(bo *retry.Backoffer, action commitAction, batches []batchKeys) error {
	if len(batches) == 0 {
		return nil
	}
	var singleBatchActionFunc func(bo *retry.Backoffer, batch batchKeys) error
	switch action {
	case actionPrewrite:
		singleBatchActionFunc = c.prewriteSingleBatch
	case actionCommit:
		singleBatchActionFunc = c.commitSingleBatch
	case actionCleanup:
		singleBatchActionFunc = c.cleanupSingleBatch
	}
	if len(batches) == 1 {
		e := singleBatchActionFunc(bo, batches[0])
		if e != nil {
			log.Debugf("con:%d 2PC doActionOnBatches %s failed: %v, tid: %d", c.ConnID, action, e, c.startTS)
		}
		return e
	}

	// For prewrite, stop sending other requests after receiving first error.
	backoffer := bo
	var cancel context.CancelFunc
	if action == actionPrewrite {
		backoffer, cancel = bo.Fork()
		defer cancel()
	}

	// Concurrently do the work for each batch.
	ch := make(chan error, len(batches))
	for _, batch1 := range batches {
		batch := batch1
		go func() {
			if action == actionCommit {
				// Because the secondary batches of the commit actions are implemented to be
				// committed asynchronously in background goroutines, we should not
				// fork a child context and call cancel() while the foreground goroutine exits.
				// Otherwise the background goroutines will be canceled exceptionally.
				// Here we makes a new clone of the original backoffer for this goroutine
				// exclusively to avoid the data race when using the same backoffer
				// in concurrent goroutines.
				singleBatchBackoffer := backoffer.Clone()
				ch <- singleBatchActionFunc(singleBatchBackoffer, batch)
			} else {
				singleBatchBackoffer, singleBatchCancel := backoffer.Fork()
				defer singleBatchCancel()
				ch <- singleBatchActionFunc(singleBatchBackoffer, batch)
			}
		}()
	}
	var err error
	for i := 0; i < len(batches); i++ {
		if e := <-ch; e != nil {
			log.Debugf("con:%d 2PC doActionOnBatches %s failed: %v, tid: %d", c.ConnID, action, e, c.startTS)
			// Cancel other requests and return the first error.
			if cancel != nil {
				log.Debugf("con:%d 2PC doActionOnBatches %s to cancel other actions, tid: %d", c.ConnID, action, c.startTS)
				cancel()
			}
			if err == nil {
				err = e
			}
		}
	}
	return err
}

func (c *TxnCommitter) keyValueSize(key []byte) int {
	size := len(key)
	if mutation := c.mutations[string(key)]; mutation != nil {
		size += len(mutation.Value)
	}
	return size
}

func (c *TxnCommitter) keySize(key []byte) int {
	return len(key)
}

func (c *TxnCommitter) prewriteSingleBatch(bo *retry.Backoffer, batch batchKeys) error {
	mutations := make([]*pb.Mutation, len(batch.keys))
	for i, k := range batch.keys {
		mutations[i] = c.mutations[string(k)]
	}

	req := &rpc.Request{
		Type: rpc.CmdPrewrite,
		Prewrite: &pb.PrewriteRequest{
			Mutations:    mutations,
			PrimaryLock:  c.primary(),
			StartVersion: c.startTS,
			LockTtl:      c.lockTTL,
		},
		Context: pb.Context{
			Priority: c.Priority,
			SyncLog:  c.SyncLog,
		},
	}
	for {
		resp, err := c.store.SendReq(bo, req, batch.region, c.conf.RPC.ReadTimeoutShort)
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
			return c.prewriteKeys(bo, batch.keys)
		}
		prewriteResp := resp.Prewrite
		if prewriteResp == nil {
			return errors.WithStack(rpc.ErrBodyMissing)
		}
		keyErrs := prewriteResp.GetErrors()
		if len(keyErrs) == 0 {
			return nil
		}
		var locks []*Lock
		for _, keyErr := range keyErrs {
			// Check already exists error
			if alreadyExist := keyErr.GetAlreadyExist(); alreadyExist != nil {
				return errors.WithStack(ErrKeyAlreadyExist(alreadyExist.GetKey()))
			}

			// Extract lock from key error
			lock, err1 := extractLockFromKeyErr(keyErr, c.conf.Txn.DefaultLockTTL)
			if err1 != nil {
				return err1
			}
			log.Debugf("con:%d 2PC prewrite encounters lock: %v", c.ConnID, lock)
			locks = append(locks, lock)
		}
		start := time.Now()
		ok, err := c.store.GetLockResolver().ResolveLocks(bo, locks)
		if err != nil {
			return err
		}
		atomic.AddInt64(&c.detail.ResolveLockTime, int64(time.Since(start)))
		if !ok {
			err = bo.Backoff(retry.BoTxnLock, errors.Errorf("2PC prewrite lockedKeys: %d", len(locks)))
			if err != nil {
				return err
			}
		}
	}
}

func (c *TxnCommitter) setUndeterminedErr(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mu.undeterminedErr = err
}

func (c *TxnCommitter) getUndeterminedErr() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mu.undeterminedErr
}

func (c *TxnCommitter) commitSingleBatch(bo *retry.Backoffer, batch batchKeys) error {
	req := &rpc.Request{
		Type: rpc.CmdCommit,
		Commit: &pb.CommitRequest{
			StartVersion:  c.startTS,
			Keys:          batch.keys,
			CommitVersion: c.commitTS,
		},
		Context: pb.Context{
			Priority: c.Priority,
			SyncLog:  c.SyncLog,
		},
	}
	req.Context.Priority = c.Priority

	sender := rpc.NewRegionRequestSender(c.store.GetRegionCache(), c.store.GetRPCClient())
	resp, err := sender.SendReq(bo, req, batch.region, c.conf.RPC.ReadTimeoutShort)

	// If we fail to receive response for the request that commits primary key, it will be undetermined whether this
	// transaction has been successfully committed.
	// Under this circumstance,  we can not declare the commit is complete (may lead to data lost), nor can we throw
	// an error (may lead to the duplicated key error when upper level restarts the transaction). Currently the best
	// solution is to populate this error and let upper layer drop the connection to the corresponding mysql client.
	isPrimary := bytes.Equal(batch.keys[0], c.primary())
	if isPrimary && sender.RPCError() != nil {
		c.setUndeterminedErr(sender.RPCError())
	}

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
		// re-split keys and commit again.
		return c.commitKeys(bo, batch.keys)
	}
	commitResp := resp.Commit
	if commitResp == nil {
		return errors.WithStack(rpc.ErrBodyMissing)
	}
	// Here we can make sure tikv has processed the commit primary key request. So
	// we can clean undetermined error.
	if isPrimary {
		c.setUndeterminedErr(nil)
	}
	if keyErr := commitResp.GetError(); keyErr != nil {
		c.mu.RLock()
		defer c.mu.RUnlock()
		err = errors.Errorf("con:%d 2PC commit failed: %v", c.ConnID, keyErr.String())
		if c.mu.committed {
			// No secondary key could be rolled back after it's primary key is committed.
			// There must be a serious bug somewhere.
			log.Errorf("2PC failed commit key after primary key committed: %v, tid: %d", err, c.startTS)
			return err
		}
		// The transaction maybe rolled back by concurrent transactions.
		log.Debugf("2PC failed commit primary key: %v, retry later, tid: %d", err, c.startTS)
		return errors.WithMessage(err, TxnRetryableMark)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	// Group that contains primary key is always the first.
	// We mark transaction's status committed when we receive the first success response.
	c.mu.committed = true
	return nil
}

func (c *TxnCommitter) cleanupSingleBatch(bo *retry.Backoffer, batch batchKeys) error {
	req := &rpc.Request{
		Type: rpc.CmdBatchRollback,
		BatchRollback: &pb.BatchRollbackRequest{
			Keys:         batch.keys,
			StartVersion: c.startTS,
		},
		Context: pb.Context{
			Priority: c.Priority,
			SyncLog:  c.SyncLog,
		},
	}
	resp, err := c.store.SendReq(bo, req, batch.region, c.conf.RPC.ReadTimeoutShort)
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
		return c.cleanupKeys(bo, batch.keys)
	}
	if keyErr := resp.BatchRollback.GetError(); keyErr != nil {
		err = errors.Errorf("con:%d 2PC cleanup failed: %s", c.ConnID, keyErr)
		log.Debugf("2PC failed cleanup key: %v, tid: %d", err, c.startTS)
		return err
	}
	return nil
}

func (c *TxnCommitter) prewriteKeys(bo *retry.Backoffer, keys [][]byte) error {
	return c.doActionOnKeys(bo, actionPrewrite, keys)
}

func (c *TxnCommitter) commitKeys(bo *retry.Backoffer, keys [][]byte) error {
	return c.doActionOnKeys(bo, actionCommit, keys)
}

func (c *TxnCommitter) cleanupKeys(bo *retry.Backoffer, keys [][]byte) error {
	return c.doActionOnKeys(bo, actionCleanup, keys)
}

// Execute executes the two-phase commit protocol.
func (c *TxnCommitter) Execute(ctx context.Context) error {
	defer func() {
		// Always clean up all written keys if the txn does not commit.
		c.mu.RLock()
		committed := c.mu.committed
		undetermined := c.mu.undeterminedErr != nil
		c.mu.RUnlock()
		if !committed && !undetermined {
			c.cleanWg.Add(1)
			go func() {
				err := c.cleanupKeys(retry.NewBackoffer(context.Background(), retry.CleanupMaxBackoff), c.keys)
				if err != nil {
					metrics.SecondaryLockCleanupFailureCounter.WithLabelValues("rollback").Inc()
					log.Infof("con:%d 2PC cleanup err: %v, tid: %d", c.ConnID, err, c.startTS)
				} else {
					log.Infof("con:%d 2PC clean up done, tid: %d", c.ConnID, c.startTS)
				}
				c.cleanWg.Done()
			}()
		}
	}()

	prewriteBo := retry.NewBackoffer(ctx, retry.PrewriteMaxBackoff)
	start := time.Now()
	err := c.prewriteKeys(prewriteBo, c.keys)
	c.detail.PrewriteTime = time.Since(start)
	c.detail.TotalBackoffTime += prewriteBo.TotalSleep()

	if err != nil {
		log.Debugf("con:%d 2PC failed on prewrite: %v, tid: %d", c.ConnID, err, c.startTS)
		return err
	}

	start = time.Now()
	commitTS, err := c.store.GetTimestampWithRetry(retry.NewBackoffer(ctx, retry.TsoMaxBackoff))
	if err != nil {
		log.Warnf("con:%d 2PC get commitTS failed: %v, tid: %d", c.ConnID, err, c.startTS)
		return err
	}
	c.detail.GetCommitTsTime = time.Since(start)

	// check commitTS
	if commitTS <= c.startTS {
		err = errors.Errorf("con:%d Invalid transaction tso with start_ts=%v while commit_ts=%v",
			c.ConnID, c.startTS, commitTS)
		log.Error(err)
		return err
	}
	c.commitTS = commitTS

	if c.store.GetOracle().IsExpired(c.startTS, c.maxTxnTimeUse) {
		err = errors.Errorf("con:%d txn takes too much time, start: %d, commit: %d", c.ConnID, c.startTS, c.commitTS)
		return errors.WithMessage(err, TxnRetryableMark)
	}

	start = time.Now()
	commitBo := retry.NewBackoffer(ctx, retry.CommitMaxBackoff)
	err = c.commitKeys(commitBo, c.keys)
	c.detail.CommitTime = time.Since(start)
	c.detail.TotalBackoffTime += commitBo.TotalSleep()
	if err != nil {
		if undeterminedErr := c.getUndeterminedErr(); undeterminedErr != nil {
			log.Warnf("con:%d 2PC commit result undetermined, err: %v, rpcErr: %v, tid: %v", c.ConnID, err, undeterminedErr, c.startTS)
			log.Error(err)
			err = errors.WithStack(ErrResultUndetermined)
		}
		if !c.mu.committed {
			log.Debugf("con:%d 2PC failed on commit: %v, tid: %d", c.ConnID, err, c.startTS)
			return err
		}
		log.Debugf("con:%d 2PC succeed with error: %v, tid: %d", c.ConnID, err, c.startTS)
	}
	return nil
}

// GetKeys returns all keys of the committer.
func (c *TxnCommitter) GetKeys() [][]byte {
	return c.keys
}

// GetCommitTS returns the commit timestamp of the transaction.
func (c *TxnCommitter) GetCommitTS() uint64 {
	return c.commitTS
}
