// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/jackofallops/giovanni/storage/2023-11-03/blob/blobs"
	"github.com/jackofallops/giovanni/storage/2023-11-03/blob/containers"
)

const (
	// This will be used as directory name, the odd looking colon is simply to
	// reduce the chance of name conflicts with existing objects.
	keyEnvPrefix = "env:"
)

func (b *Backend) Workspaces() ([]string, error) {
	prefix := b.keyName + keyEnvPrefix
	params := containers.ListBlobsInput{
		Prefix: &prefix,
	}

	ctx := newCtx()
	client, err := b.apiClient.getContainersClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("retrieving container client: %v", err)
	}
	resp, err := client.ListBlobs(ctx, b.containerName, params)
	if err != nil {
		return nil, fmt.Errorf("listing blobs: %v", err)
	}

	envs := map[string]struct{}{}
	for _, obj := range resp.Blobs.Blobs {
		key := obj.Name
		if strings.HasPrefix(key, prefix) {
			name := strings.TrimPrefix(key, prefix)
			// we store the state in a key, not a directory
			if strings.Contains(name, "/") {
				continue
			}

			envs[name] = struct{}{}
		}
	}

	result := []string{backend.DefaultStateName}
	for name := range envs {
		result = append(result, name)
	}
	sort.Strings(result[1:])
	return result, nil
}

func (b *Backend) DeleteWorkspace(name string, _ bool) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	ctx := newCtx()
	client, err := b.apiClient.getBlobClient(ctx)
	if err != nil {
		return err
	}

	if resp, err := client.Delete(ctx, b.containerName, b.path(name), blobs.DeleteInput{}); err != nil {
		if !response.WasNotFound(resp.HttpResponse) {
			return err
		}
	}

	return nil
}

func (b *Backend) StateMgr(name string) (statemgr.Full, error) {
	ctx := newCtx()
	blobClient, err := b.apiClient.getBlobClient(ctx)
	if err != nil {
		return nil, err
	}

	client := &RemoteClient{
		giovanniBlobClient: *blobClient,
		containerName:      b.containerName,
		keyName:            b.path(name),
		accountName:        b.accountName,
		snapshot:           b.snapshot,
	}

	stateMgr := &remote.State{Client: client}

	// Grab the value
	if err := stateMgr.RefreshState(); err != nil {
		return nil, err
	}
	//if this isn't the default state name, we need to create the object so
	//it's listed by States.
	if v := stateMgr.State(); v == nil {
		// take a lock on this state while we write it
		lockInfo := statemgr.NewLockInfo()
		lockInfo.Operation = "init"
		lockId, err := client.Lock(lockInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to lock azure state: %s", err)
		}

		// Local helper function so we can call it multiple places
		lockUnlock := func(parent error) error {
			if err := stateMgr.Unlock(lockId); err != nil {
				return fmt.Errorf(strings.TrimSpace(errStateUnlock), lockId, err)
			}
			return parent
		}

		// Grab the value
		if err := stateMgr.RefreshState(); err != nil {
			err = lockUnlock(err)
			return nil, err
		}
		//if this isn't the default state name, we need to create the object so
		//it's listed by States.
		if v := stateMgr.State(); v == nil {
			// If we have no state, we have to create an empty state
			if err := stateMgr.WriteState(states.NewState()); err != nil {
				err = lockUnlock(err)
				return nil, err
			}
			if err := stateMgr.PersistState(nil); err != nil {
				err = lockUnlock(err)
				return nil, err
			}

			// Unlock, the state should now be initialized
			if err := lockUnlock(nil); err != nil {
				return nil, err
			}
		}
	}

	return stateMgr, nil
}

func (b *Backend) client() *RemoteClient {
	return &RemoteClient{}
}

func (b *Backend) path(name string) string {
	if name == backend.DefaultStateName {
		return b.keyName
	}

	return b.keyName + keyEnvPrefix + name
}

const errStateUnlock = `
Error unlocking Azure state. Lock ID: %s

Error: %s

You may have to force-unlock this state in order to use it again.
`
