// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package inmem

import (
	"crypto/sha256"

	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// RemoteClient is a remote client that stores data in memory for testing.
type RemoteClient struct {
	Data []byte
	MD5  []byte
	Name string
}

func (c *RemoteClient) Get() (*remote.Payload, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if c.Data == nil {
		return nil, nil
	}

	return &remote.Payload{
		Data: c.Data,
		MD5:  c.MD5,
	}, diags
}

func (c *RemoteClient) Put(data []byte) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	hash := sha256.Sum256(data)

	c.Data = data
	c.MD5 = hash[:]
	return diags
}

func (c *RemoteClient) Delete() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	c.Data = nil
	c.MD5 = nil
	return diags
}

func (c *RemoteClient) Lock(info *statemgr.LockInfo) (string, error) {
	return locks.lock(c.Name, info)
}
func (c *RemoteClient) Unlock(id string) error {
	return locks.unlock(c.Name, id)
}
