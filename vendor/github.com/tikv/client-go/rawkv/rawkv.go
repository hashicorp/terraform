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

package rawkv

import (
	"bytes"
	"context"
	"time"

	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	pd "github.com/pingcap/pd/client"
	"github.com/pkg/errors"
	"github.com/tikv/client-go/config"
	"github.com/tikv/client-go/locate"
	"github.com/tikv/client-go/metrics"
	"github.com/tikv/client-go/retry"
	"github.com/tikv/client-go/rpc"
)

var (
	// ErrMaxScanLimitExceeded is returned when the limit for rawkv Scan is to large.
	ErrMaxScanLimitExceeded = errors.New("limit should be less than MaxRawKVScanLimit")
)

// Client is a rawkv client of TiKV server which is used as a key-value storage,
// only GET/PUT/DELETE commands are supported.
type Client struct {
	clusterID   uint64
	conf        *config.Config
	regionCache *locate.RegionCache
	pdClient    pd.Client
	rpcClient   rpc.Client
}

// NewClient creates a client with PD cluster addrs.
func NewClient(ctx context.Context, pdAddrs []string, conf config.Config) (*Client, error) {
	pdCli, err := pd.NewClient(pdAddrs, pd.SecurityOption{
		CAPath:   conf.RPC.Security.SSLCA,
		CertPath: conf.RPC.Security.SSLCert,
		KeyPath:  conf.RPC.Security.SSLKey,
	})
	if err != nil {
		return nil, err
	}
	return &Client{
		clusterID:   pdCli.GetClusterID(ctx),
		conf:        &conf,
		regionCache: locate.NewRegionCache(pdCli, &conf.RegionCache),
		pdClient:    pdCli,
		rpcClient:   rpc.NewRPCClient(&conf.RPC),
	}, nil
}

// Close closes the client.
func (c *Client) Close() error {
	c.pdClient.Close()
	return c.rpcClient.Close()
}

// ClusterID returns the TiKV cluster ID.
func (c *Client) ClusterID() uint64 {
	return c.clusterID
}

// Get queries value with the key. When the key does not exist, it returns `nil, nil`.
func (c *Client) Get(ctx context.Context, key []byte) ([]byte, error) {
	start := time.Now()
	defer func() { metrics.RawkvCmdHistogram.WithLabelValues("get").Observe(time.Since(start).Seconds()) }()

	req := &rpc.Request{
		Type: rpc.CmdRawGet,
		RawGet: &kvrpcpb.RawGetRequest{
			Key: key,
		},
	}
	resp, _, err := c.sendReq(ctx, key, req)
	if err != nil {
		return nil, err
	}
	cmdResp := resp.RawGet
	if cmdResp == nil {
		return nil, errors.WithStack(rpc.ErrBodyMissing)
	}
	if cmdResp.GetError() != "" {
		return nil, errors.New(cmdResp.GetError())
	}
	if len(cmdResp.Value) == 0 {
		return nil, nil
	}
	return cmdResp.Value, nil
}

// BatchGet queries values with the keys.
func (c *Client) BatchGet(ctx context.Context, keys [][]byte) ([][]byte, error) {
	start := time.Now()
	defer func() { metrics.RawkvCmdHistogram.WithLabelValues("batch_get").Observe(time.Since(start).Seconds()) }()

	bo := retry.NewBackoffer(ctx, retry.RawkvMaxBackoff)
	resp, err := c.sendBatchReq(bo, keys, rpc.CmdRawBatchGet)
	if err != nil {
		return nil, err
	}

	cmdResp := resp.RawBatchGet
	if cmdResp == nil {
		return nil, errors.WithStack(rpc.ErrBodyMissing)
	}

	keyToValue := make(map[string][]byte, len(keys))
	for _, pair := range cmdResp.Pairs {
		keyToValue[string(pair.Key)] = pair.Value
	}

	values := make([][]byte, len(keys))
	for i, key := range keys {
		values[i] = keyToValue[string(key)]
	}
	return values, nil
}

// Put stores a key-value pair to TiKV.
func (c *Client) Put(ctx context.Context, key, value []byte) error {
	start := time.Now()
	defer func() { metrics.RawkvCmdHistogram.WithLabelValues("put").Observe(time.Since(start).Seconds()) }()
	metrics.RawkvSizeHistogram.WithLabelValues("key").Observe(float64(len(key)))
	metrics.RawkvSizeHistogram.WithLabelValues("value").Observe(float64(len(value)))

	if len(value) == 0 {
		return errors.New("empty value is not supported")
	}

	req := &rpc.Request{
		Type: rpc.CmdRawPut,
		RawPut: &kvrpcpb.RawPutRequest{
			Key:   key,
			Value: value,
		},
	}
	resp, _, err := c.sendReq(ctx, key, req)
	if err != nil {
		return err
	}
	cmdResp := resp.RawPut
	if cmdResp == nil {
		return errors.WithStack(rpc.ErrBodyMissing)
	}
	if cmdResp.GetError() != "" {
		return errors.New(cmdResp.GetError())
	}
	return nil
}

// BatchPut stores key-value pairs to TiKV.
func (c *Client) BatchPut(ctx context.Context, keys, values [][]byte) error {
	start := time.Now()
	defer func() { metrics.RawkvCmdHistogram.WithLabelValues("batch_put").Observe(time.Since(start).Seconds()) }()

	if len(keys) != len(values) {
		return errors.New("the len of keys is not equal to the len of values")
	}
	for _, value := range values {
		if len(value) == 0 {
			return errors.New("empty value is not supported")
		}
	}
	bo := retry.NewBackoffer(ctx, retry.RawkvMaxBackoff)
	return c.sendBatchPut(bo, keys, values)
}

// Delete deletes a key-value pair from TiKV.
func (c *Client) Delete(ctx context.Context, key []byte) error {
	start := time.Now()
	defer func() { metrics.RawkvCmdHistogram.WithLabelValues("delete").Observe(time.Since(start).Seconds()) }()

	req := &rpc.Request{
		Type: rpc.CmdRawDelete,
		RawDelete: &kvrpcpb.RawDeleteRequest{
			Key: key,
		},
	}
	resp, _, err := c.sendReq(ctx, key, req)
	if err != nil {
		return err
	}
	cmdResp := resp.RawDelete
	if cmdResp == nil {
		return errors.WithStack(rpc.ErrBodyMissing)
	}
	if cmdResp.GetError() != "" {
		return errors.New(cmdResp.GetError())
	}
	return nil
}

// BatchDelete deletes key-value pairs from TiKV.
func (c *Client) BatchDelete(ctx context.Context, keys [][]byte) error {
	start := time.Now()
	defer func() { metrics.RawkvCmdHistogram.WithLabelValues("batch_delete").Observe(time.Since(start).Seconds()) }()

	bo := retry.NewBackoffer(ctx, retry.RawkvMaxBackoff)
	resp, err := c.sendBatchReq(bo, keys, rpc.CmdRawBatchDelete)
	if err != nil {
		return err
	}
	cmdResp := resp.RawBatchDelete
	if cmdResp == nil {
		return errors.WithStack(rpc.ErrBodyMissing)
	}
	if cmdResp.GetError() != "" {
		return errors.New(cmdResp.GetError())
	}
	return nil
}

// DeleteRange deletes all key-value pairs in a range from TiKV
func (c *Client) DeleteRange(ctx context.Context, startKey []byte, endKey []byte) error {
	start := time.Now()
	var err error
	defer func() { metrics.RawkvCmdHistogram.WithLabelValues("delete_range").Observe(time.Since(start).Seconds()) }()

	// Process each affected region respectively
	for !bytes.Equal(startKey, endKey) {
		var resp *rpc.Response
		var actualEndKey []byte
		resp, actualEndKey, err = c.sendDeleteRangeReq(ctx, startKey, endKey)
		if err != nil {
			return err
		}
		cmdResp := resp.RawDeleteRange
		if cmdResp == nil {
			return errors.WithStack(rpc.ErrBodyMissing)
		}
		if cmdResp.GetError() != "" {
			return errors.New(cmdResp.GetError())
		}
		startKey = actualEndKey
	}

	return nil
}

// Scan queries continuous kv pairs in range [startKey, endKey), up to limit pairs.
// If endKey is empty, it means unbounded.
// If you want to exclude the startKey or include the endKey, append a '\0' to the key. For example, to scan
// (startKey, endKey], you can write:
// `Scan(append(startKey, '\0'), append(endKey, '\0'), limit)`.
func (c *Client) Scan(ctx context.Context, startKey, endKey []byte, limit int) (keys [][]byte, values [][]byte, err error) {
	start := time.Now()
	defer func() { metrics.RawkvCmdHistogram.WithLabelValues("scan").Observe(time.Since(start).Seconds()) }()

	if limit > c.conf.Raw.MaxScanLimit {
		return nil, nil, errors.WithStack(ErrMaxScanLimitExceeded)
	}

	for len(keys) < limit {
		req := &rpc.Request{
			Type: rpc.CmdRawScan,
			RawScan: &kvrpcpb.RawScanRequest{
				StartKey: startKey,
				EndKey:   endKey,
				Limit:    uint32(limit - len(keys)),
			},
		}
		resp, loc, err := c.sendReq(ctx, startKey, req)
		if err != nil {
			return nil, nil, err
		}
		cmdResp := resp.RawScan
		if cmdResp == nil {
			return nil, nil, errors.WithStack(rpc.ErrBodyMissing)
		}
		for _, pair := range cmdResp.Kvs {
			keys = append(keys, pair.Key)
			values = append(values, pair.Value)
		}
		startKey = loc.EndKey
		if len(startKey) == 0 {
			break
		}
	}
	return
}

// ReverseScan queries continuous kv pairs in range [endKey, startKey), up to limit pairs.
// Direction is different from Scan, upper to lower.
// If endKey is empty, it means unbounded.
// If you want to include the startKey or exclude the endKey, append a '\0' to the key. For example, to scan
// (endKey, startKey], you can write:
// `ReverseScan(append(startKey, '\0'), append(endKey, '\0'), limit)`.
// It doesn't support Scanning from "", because locating the last Region is not yet implemented.
func (c *Client) ReverseScan(ctx context.Context, startKey, endKey []byte, limit int) (keys [][]byte, values [][]byte, err error) {
	start := time.Now()
	defer func() { metrics.RawkvCmdHistogram.WithLabelValues("reverse_scan").Observe(time.Since(start).Seconds()) }()

	if limit > c.conf.Raw.MaxScanLimit {
		return nil, nil, errors.WithStack(ErrMaxScanLimitExceeded)
	}

	for len(keys) < limit {
		req := &rpc.Request{
			Type: rpc.CmdRawScan,
			RawScan: &kvrpcpb.RawScanRequest{
				StartKey: startKey,
				EndKey:   endKey,
				Limit:    uint32(limit - len(keys)),
				Reverse:  true,
			},
		}
		resp, loc, err := c.sendReq(ctx, startKey, req)
		if err != nil {
			return nil, nil, err
		}
		cmdResp := resp.RawScan
		if cmdResp == nil {
			return nil, nil, errors.WithStack(rpc.ErrBodyMissing)
		}
		for _, pair := range cmdResp.Kvs {
			keys = append(keys, pair.Key)
			values = append(values, pair.Value)
		}
		startKey = loc.EndKey
		if len(startKey) == 0 {
			break
		}
	}
	return
}

func (c *Client) sendReq(ctx context.Context, key []byte, req *rpc.Request) (*rpc.Response, *locate.KeyLocation, error) {
	bo := retry.NewBackoffer(ctx, retry.RawkvMaxBackoff)
	sender := rpc.NewRegionRequestSender(c.regionCache, c.rpcClient)
	for {
		loc, err := c.regionCache.LocateKey(bo, key)
		if err != nil {
			return nil, nil, err
		}
		resp, err := sender.SendReq(bo, req, loc.Region, c.conf.RPC.ReadTimeoutShort)
		if err != nil {
			return nil, nil, err
		}
		regionErr, err := resp.GetRegionError()
		if err != nil {
			return nil, nil, err
		}
		if regionErr != nil {
			err := bo.Backoff(retry.BoRegionMiss, errors.New(regionErr.String()))
			if err != nil {
				return nil, nil, err
			}
			continue
		}
		return resp, loc, nil
	}
}

func (c *Client) sendBatchReq(bo *retry.Backoffer, keys [][]byte, cmdType rpc.CmdType) (*rpc.Response, error) { // split the keys
	groups, _, err := c.regionCache.GroupKeysByRegion(bo, keys)
	if err != nil {
		return nil, err
	}

	var batches []batch
	for regionID, groupKeys := range groups {
		batches = appendKeyBatches(batches, regionID, groupKeys, c.conf.Raw.BatchPairCount)
	}
	bo, cancel := bo.Fork()
	ches := make(chan singleBatchResp, len(batches))
	for _, batch := range batches {
		batch1 := batch
		go func() {
			singleBatchBackoffer, singleBatchCancel := bo.Fork()
			defer singleBatchCancel()
			ches <- c.doBatchReq(singleBatchBackoffer, batch1, cmdType)
		}()
	}

	var firstError error
	var resp *rpc.Response
	switch cmdType {
	case rpc.CmdRawBatchGet:
		resp = &rpc.Response{Type: rpc.CmdRawBatchGet, RawBatchGet: &kvrpcpb.RawBatchGetResponse{}}
	case rpc.CmdRawBatchDelete:
		resp = &rpc.Response{Type: rpc.CmdRawBatchDelete, RawBatchDelete: &kvrpcpb.RawBatchDeleteResponse{}}
	}
	for i := 0; i < len(batches); i++ {
		singleResp, ok := <-ches
		if ok {
			if singleResp.err != nil {
				cancel()
				if firstError == nil {
					firstError = singleResp.err
				}
			} else if cmdType == rpc.CmdRawBatchGet {
				cmdResp := singleResp.resp.RawBatchGet
				resp.RawBatchGet.Pairs = append(resp.RawBatchGet.Pairs, cmdResp.Pairs...)
			}
		}
	}

	return resp, firstError
}

func (c *Client) doBatchReq(bo *retry.Backoffer, batch batch, cmdType rpc.CmdType) singleBatchResp {
	var req *rpc.Request
	switch cmdType {
	case rpc.CmdRawBatchGet:
		req = &rpc.Request{
			Type: cmdType,
			RawBatchGet: &kvrpcpb.RawBatchGetRequest{
				Keys: batch.keys,
			},
		}
	case rpc.CmdRawBatchDelete:
		req = &rpc.Request{
			Type: cmdType,
			RawBatchDelete: &kvrpcpb.RawBatchDeleteRequest{
				Keys: batch.keys,
			},
		}
	}

	sender := rpc.NewRegionRequestSender(c.regionCache, c.rpcClient)
	resp, err := sender.SendReq(bo, req, batch.regionID, c.conf.RPC.ReadTimeoutShort)

	batchResp := singleBatchResp{}
	if err != nil {
		batchResp.err = err
		return batchResp
	}
	regionErr, err := resp.GetRegionError()
	if err != nil {
		batchResp.err = err
		return batchResp
	}
	if regionErr != nil {
		err := bo.Backoff(retry.BoRegionMiss, errors.New(regionErr.String()))
		if err != nil {
			batchResp.err = err
			return batchResp
		}
		resp, err = c.sendBatchReq(bo, batch.keys, cmdType)
		batchResp.resp = resp
		batchResp.err = err
		return batchResp
	}

	switch cmdType {
	case rpc.CmdRawBatchGet:
		batchResp.resp = resp
	case rpc.CmdRawBatchDelete:
		cmdResp := resp.RawBatchDelete
		if cmdResp == nil {
			batchResp.err = errors.WithStack(rpc.ErrBodyMissing)
			return batchResp
		}
		if cmdResp.GetError() != "" {
			batchResp.err = errors.New(cmdResp.GetError())
			return batchResp
		}
		batchResp.resp = resp
	}
	return batchResp
}

// sendDeleteRangeReq sends a raw delete range request and returns the response and the actual endKey.
// If the given range spans over more than one regions, the actual endKey is the end of the first region.
// We can't use sendReq directly, because we need to know the end of the region before we send the request
// TODO: Is there any better way to avoid duplicating code with func `sendReq` ?
func (c *Client) sendDeleteRangeReq(ctx context.Context, startKey []byte, endKey []byte) (*rpc.Response, []byte, error) {
	bo := retry.NewBackoffer(ctx, retry.RawkvMaxBackoff)
	sender := rpc.NewRegionRequestSender(c.regionCache, c.rpcClient)
	for {
		loc, err := c.regionCache.LocateKey(bo, startKey)
		if err != nil {
			return nil, nil, err
		}

		actualEndKey := endKey
		if len(loc.EndKey) > 0 && bytes.Compare(loc.EndKey, endKey) < 0 {
			actualEndKey = loc.EndKey
		}

		req := &rpc.Request{
			Type: rpc.CmdRawDeleteRange,
			RawDeleteRange: &kvrpcpb.RawDeleteRangeRequest{
				StartKey: startKey,
				EndKey:   actualEndKey,
			},
		}

		resp, err := sender.SendReq(bo, req, loc.Region, c.conf.RPC.ReadTimeoutShort)
		if err != nil {
			return nil, nil, err
		}
		regionErr, err := resp.GetRegionError()
		if err != nil {
			return nil, nil, err
		}
		if regionErr != nil {
			err := bo.Backoff(retry.BoRegionMiss, errors.New(regionErr.String()))
			if err != nil {
				return nil, nil, err
			}
			continue
		}
		return resp, actualEndKey, nil
	}
}

func (c *Client) sendBatchPut(bo *retry.Backoffer, keys, values [][]byte) error {
	keyToValue := make(map[string][]byte)
	for i, key := range keys {
		keyToValue[string(key)] = values[i]
	}
	groups, _, err := c.regionCache.GroupKeysByRegion(bo, keys)
	if err != nil {
		return err
	}
	var batches []batch
	// split the keys by size and RegionVerID
	for regionID, groupKeys := range groups {
		batches = appendBatches(batches, regionID, groupKeys, keyToValue, c.conf.Raw.MaxBatchPutSize)
	}
	bo, cancel := bo.Fork()
	ch := make(chan error, len(batches))
	for _, batch := range batches {
		batch1 := batch
		go func() {
			singleBatchBackoffer, singleBatchCancel := bo.Fork()
			defer singleBatchCancel()
			ch <- c.doBatchPut(singleBatchBackoffer, batch1)
		}()
	}

	for i := 0; i < len(batches); i++ {
		if e := <-ch; e != nil {
			cancel()
			// catch the first error
			if err == nil {
				err = e
			}
		}
	}
	return err
}

func appendKeyBatches(batches []batch, regionID locate.RegionVerID, groupKeys [][]byte, limit int) []batch {
	var keys [][]byte
	for start, count := 0, 0; start < len(groupKeys); start++ {
		if count > limit {
			batches = append(batches, batch{regionID: regionID, keys: keys})
			keys = make([][]byte, 0, limit)
			count = 0
		}
		keys = append(keys, groupKeys[start])
		count++
	}
	if len(keys) != 0 {
		batches = append(batches, batch{regionID: regionID, keys: keys})
	}
	return batches
}

func appendBatches(batches []batch, regionID locate.RegionVerID, groupKeys [][]byte, keyToValue map[string][]byte, limit int) []batch {
	var start, size int
	var keys, values [][]byte
	for start = 0; start < len(groupKeys); start++ {
		if size >= limit {
			batches = append(batches, batch{regionID: regionID, keys: keys, values: values})
			keys = make([][]byte, 0)
			values = make([][]byte, 0)
			size = 0
		}
		key := groupKeys[start]
		value := keyToValue[string(key)]
		keys = append(keys, key)
		values = append(values, value)
		size += len(key)
		size += len(value)
	}
	if len(keys) != 0 {
		batches = append(batches, batch{regionID: regionID, keys: keys, values: values})
	}
	return batches
}

func (c *Client) doBatchPut(bo *retry.Backoffer, batch batch) error {
	kvPair := make([]*kvrpcpb.KvPair, 0, len(batch.keys))
	for i, key := range batch.keys {
		kvPair = append(kvPair, &kvrpcpb.KvPair{Key: key, Value: batch.values[i]})
	}

	req := &rpc.Request{
		Type: rpc.CmdRawBatchPut,
		RawBatchPut: &kvrpcpb.RawBatchPutRequest{
			Pairs: kvPair,
		},
	}

	sender := rpc.NewRegionRequestSender(c.regionCache, c.rpcClient)
	resp, err := sender.SendReq(bo, req, batch.regionID, c.conf.RPC.ReadTimeoutShort)
	if err != nil {
		return err
	}
	regionErr, err := resp.GetRegionError()
	if err != nil {
		return err
	}
	if regionErr != nil {
		err := bo.Backoff(retry.BoRegionMiss, errors.New(regionErr.String()))
		if err != nil {
			return err
		}
		// recursive call
		return c.sendBatchPut(bo, batch.keys, batch.values)
	}

	cmdResp := resp.RawBatchPut
	if cmdResp == nil {
		return errors.WithStack(rpc.ErrBodyMissing)
	}
	if cmdResp.GetError() != "" {
		return errors.New(cmdResp.GetError())
	}
	return nil
}

type batch struct {
	regionID locate.RegionVerID
	keys     [][]byte
	values   [][]byte
}

type singleBatchResp struct {
	resp *rpc.Response
	err  error
}
