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

	"github.com/tikv/client-go/config"
	"github.com/tikv/client-go/retry"
	"github.com/tikv/client-go/txnkv/store"
)

// Client is a transactional client of TiKV server.
type Client struct {
	tikvStore *store.TiKVStore
}

// NewClient creates a client with PD addresses.
func NewClient(ctx context.Context, pdAddrs []string, config config.Config) (*Client, error) {
	tikvStore, err := store.NewStore(ctx, pdAddrs, config)
	if err != nil {
		return nil, err
	}
	return &Client{
		tikvStore: tikvStore,
	}, nil
}

// Close stop the client.
func (c *Client) Close() error {
	return c.tikvStore.Close()
}

// Begin creates a transaction for read/write.
func (c *Client) Begin(ctx context.Context) (*Transaction, error) {
	ts, err := c.GetTS(ctx)
	if err != nil {
		return nil, err
	}
	return c.BeginWithTS(ctx, ts), nil
}

// BeginWithTS creates a transaction which is normally readonly.
func (c *Client) BeginWithTS(ctx context.Context, ts uint64) *Transaction {
	return newTransaction(c.tikvStore, ts)
}

// GetTS returns a latest timestamp.
func (c *Client) GetTS(ctx context.Context) (uint64, error) {
	return c.tikvStore.GetTimestampWithRetry(retry.NewBackoffer(ctx, retry.TsoMaxBackoff))
}
