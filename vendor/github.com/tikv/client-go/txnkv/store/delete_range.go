// Copyright 2018 PingCAP, Inc.
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

	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pkg/errors"
	"github.com/tikv/client-go/retry"
	"github.com/tikv/client-go/rpc"
)

// DeleteRangeTask is used to delete all keys in a range. After
// performing DeleteRange, it keeps how many ranges it affects and
// if the task was canceled or not.
type DeleteRangeTask struct {
	completedRegions int
	canceled         bool
	store            *TiKVStore
	ctx              context.Context
	startKey         []byte
	endKey           []byte
}

// NewDeleteRangeTask creates a DeleteRangeTask. Deleting will not be performed right away.
// WARNING: Currently, this API may leave some waste key-value pairs uncleaned in TiKV. Be careful while using it.
func NewDeleteRangeTask(ctx context.Context, store *TiKVStore, startKey []byte, endKey []byte) *DeleteRangeTask {
	return &DeleteRangeTask{
		completedRegions: 0,
		canceled:         false,
		store:            store,
		ctx:              ctx,
		startKey:         startKey,
		endKey:           endKey,
	}
}

// Execute performs the delete range operation.
func (t *DeleteRangeTask) Execute() error {
	conf := t.store.GetConfig()

	startKey, rangeEndKey := t.startKey, t.endKey
	for {
		select {
		case <-t.ctx.Done():
			t.canceled = true
			return nil
		default:
		}
		bo := retry.NewBackoffer(t.ctx, retry.DeleteRangeOneRegionMaxBackoff)
		loc, err := t.store.GetRegionCache().LocateKey(bo, startKey)
		if err != nil {
			return err
		}

		// Delete to the end of the region, except if it's the last region overlapping the range
		endKey := loc.EndKey
		// If it is the last region
		if loc.Contains(rangeEndKey) {
			endKey = rangeEndKey
		}

		req := &rpc.Request{
			Type: rpc.CmdDeleteRange,
			DeleteRange: &kvrpcpb.DeleteRangeRequest{
				StartKey: startKey,
				EndKey:   endKey,
			},
		}

		resp, err := t.store.SendReq(bo, req, loc.Region, conf.RPC.ReadTimeoutMedium)
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
		deleteRangeResp := resp.DeleteRange
		if deleteRangeResp == nil {
			return errors.WithStack(rpc.ErrBodyMissing)
		}
		if err := deleteRangeResp.GetError(); err != "" {
			return errors.Errorf("unexpected delete range err: %v", err)
		}
		t.completedRegions++
		if bytes.Equal(endKey, rangeEndKey) {
			break
		}
		startKey = endKey
	}

	return nil
}

// CompletedRegions returns the number of regions that are affected by this delete range task
func (t *DeleteRangeTask) CompletedRegions() int {
	return t.completedRegions
}

// IsCanceled returns true if the delete range operation was canceled on the half way
func (t *DeleteRangeTask) IsCanceled() bool {
	return t.canceled
}
