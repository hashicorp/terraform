// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package elasticsearch

import (
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func (b *Backend) Workspaces() ([]string, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	client := &RemoteClient{
		Client: b.client,
		Index:  b.index,
	}

	workspaces, err := client.Workspaces()
	if err != nil {
		return nil, diags.Append(fmt.Errorf("failed to list workspaces: %w", err))
	}

	// Ensure default workspace is always first
	result := []string{backend.DefaultStateName}
	for _, name := range workspaces {
		if name != backend.DefaultStateName {
			result = append(result, name)
		}
	}

	// Sort non-default workspaces
	if len(result) > 1 {
		sort.Strings(result[1:])
	}

	return result, diags
}

func (b *Backend) DeleteWorkspace(name string, _ bool) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if name == backend.DefaultStateName || name == "" {
		return diags.Append(fmt.Errorf("can't delete default state"))
	}

	client := &RemoteClient{
		Client:    b.client,
		Index:     b.index,
		Workspace: name,
	}

	if err := client.DeleteWorkspace(); err != nil {
		return diags.Append(err)
	}

	return diags
}

func (b *Backend) StateMgr(name string) (statemgr.Full, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Build the state client
	var stateMgr statemgr.Full = &remote.State{
		Client: &RemoteClient{
			Client:      b.client,
			Index:       b.index,
			Workspace:   name,
			LockEnabled: true,
		},
	}

	// Check to see if this state already exists.
	// If the state doesn't exist, we have to assume this
	// is a normal create operation, and take the lock at that point.
	existing, wDiags := b.Workspaces()
	diags = diags.Append(wDiags)
	if wDiags.HasErrors() {
		return nil, diags
	}

	exists := false
	for _, s := range existing {
		if s == name {
			exists = true
			break
		}
	}

	// Grab a lock, we use this to write an empty state if one doesn't
	// exist already. We have to write an empty state as a sentinel value
	// so Workspaces() knows it exists.
	if !exists {
		lockInfo := statemgr.NewLockInfo()
		lockInfo.Operation = "init"
		lockId, err := stateMgr.Lock(lockInfo)
		if err != nil {
			return nil, diags.Append(fmt.Errorf("failed to lock Elasticsearch state: %s", err))
		}

		// Local helper function so we can call it multiple places
		lockUnlock := func(parent error) error {
			if err := stateMgr.Unlock(lockId); err != nil {
				return fmt.Errorf("error unlocking Elasticsearch state: %s", err)
			}
			return parent
		}

		if v := stateMgr.State(); v == nil {
			if err := stateMgr.WriteState(states.NewState()); err != nil {
				err = lockUnlock(err)
				return nil, diags.Append(err)
			}
			if err := stateMgr.PersistState(nil); err != nil {
				err = lockUnlock(err)
				return nil, diags.Append(err)
			}
		}

		// Unlock, the state should now be initialized
		if err := lockUnlock(nil); err != nil {
			return nil, diags.Append(err)
		}
	}

	return stateMgr, diags
}
