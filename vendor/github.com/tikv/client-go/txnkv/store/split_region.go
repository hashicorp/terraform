// Copyright 2017 PingCAP, Inc.
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
	log "github.com/sirupsen/logrus"
	"github.com/tikv/client-go/key"
	"github.com/tikv/client-go/retry"
	"github.com/tikv/client-go/rpc"
)

// SplitRegion splits the region contains splitKey into 2 regions: [start,
// splitKey) and [splitKey, end).
func SplitRegion(ctx context.Context, store *TiKVStore, splitKey key.Key) error {
	log.Infof("start split_region at %q", splitKey)
	bo := retry.NewBackoffer(ctx, retry.SplitRegionBackoff)
	sender := rpc.NewRegionRequestSender(store.GetRegionCache(), store.GetRPCClient())
	req := &rpc.Request{
		Type: rpc.CmdSplitRegion,
		SplitRegion: &kvrpcpb.SplitRegionRequest{
			SplitKey: splitKey,
		},
	}
	req.Context.Priority = kvrpcpb.CommandPri_Normal
	conf := store.GetConfig()
	for {
		loc, err := store.GetRegionCache().LocateKey(bo, splitKey)
		if err != nil {
			return err
		}
		if bytes.Equal(splitKey, loc.StartKey) {
			log.Infof("skip split_region region at %q", splitKey)
			return nil
		}
		res, err := sender.SendReq(bo, req, loc.Region, conf.RPC.ReadTimeoutShort)
		if err != nil {
			return err
		}
		regionErr, err := res.GetRegionError()
		if err != nil {
			return err
		}
		if regionErr != nil {
			err := bo.Backoff(retry.BoRegionMiss, errors.New(regionErr.String()))
			if err != nil {
				return err
			}
			continue
		}
		log.Infof("split_region at %q complete, new regions: %v, %v", splitKey, res.SplitRegion.GetLeft(), res.SplitRegion.GetRight())
		return nil
	}
}
