// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cos

import (
	"fmt"
	"log"
	"path"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Define file suffix
const (
	stateFileSuffix = ".tfstate"
	lockFileSuffix  = ".tflock"
)

// Workspaces returns a list of names for the workspaces
func (b *Backend) Workspaces() ([]string, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	c, err := b.client("tencentcloud")
	if err != nil {
		return nil, diags.Append(err)
	}

	obs, err := c.getBucket(b.prefix)
	log.Printf("[DEBUG] list all workspaces, objects: %v, error: %v", obs, err)
	if err != nil {
		return nil, diags.Append(err)
	}

	ws := []string{backend.DefaultStateName}
	for _, vv := range obs {
		// <name>.tfstate
		if !strings.HasSuffix(vv.Key, stateFileSuffix) {
			continue
		}
		// default worksapce
		if path.Join(b.prefix, b.key) == vv.Key {
			continue
		}
		// <prefix>/<worksapce>/<key>
		prefix := strings.TrimRight(b.prefix, "/") + "/"
		parts := strings.Split(strings.TrimPrefix(vv.Key, prefix), "/")
		if len(parts) > 0 && parts[0] != "" {
			ws = append(ws, parts[0])
		}
	}

	sort.Strings(ws[1:])
	log.Printf("[DEBUG] list all workspaces, workspaces: %v", ws)

	return ws, diags
}

// DeleteWorkspace deletes the named workspaces. The "default" state cannot be deleted.
func (b *Backend) DeleteWorkspace(name string, _ bool) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	log.Printf("[DEBUG] delete workspace, workspace: %v", name)

	if name == backend.DefaultStateName || name == "" {
		return tfdiags.Diagnostics{}.Append(fmt.Errorf("default state is not allowed to be deleted"))
	}

	c, err := b.client(name)
	if err != nil {
		return diags.Append(err)
	}

	return diags.Append(c.Delete())
}

// StateMgr manage the state, if the named state not exists, a new file will created
func (b *Backend) StateMgr(name string) (statemgr.Full, error) {
	log.Printf("[DEBUG] state manager, current workspace: %v", name)

	c, err := b.client(name)
	if err != nil {
		return nil, err
	}
	stateMgr := &remote.State{Client: c}

	ws, diags := b.Workspaces()
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	exists := false
	for _, candidate := range ws {
		if candidate == name {
			exists = true
			break
		}
	}

	if !exists {
		log.Printf("[DEBUG] workspace %v not exists", name)

		// take a lock on this state while we write it
		lockInfo := statemgr.NewLockInfo()
		lockInfo.Operation = "init"
		lockId, err := c.Lock(lockInfo)
		if err != nil {
			return nil, fmt.Errorf("Failed to lock cos state: %s", err)
		}

		// Local helper function so we can call it multiple places
		lockUnlock := func(e error) error {
			if err := stateMgr.Unlock(lockId); err != nil {
				return fmt.Errorf(unlockErrMsg, err, lockId)
			}
			return e
		}

		// Grab the value
		if err := stateMgr.RefreshState(); err != nil {
			err = lockUnlock(err)
			return nil, err
		}

		// If we have no state, we have to create an empty state
		if v := stateMgr.State(); v == nil {
			if err := stateMgr.WriteState(states.NewState()); err != nil {
				err = lockUnlock(err)
				return nil, err
			}
			if err := stateMgr.PersistState(nil); err != nil {
				err = lockUnlock(err)
				return nil, err
			}
		}

		// Unlock, the state should now be initialized
		if err := lockUnlock(nil); err != nil {
			return nil, err
		}
	}

	return stateMgr, nil
}

// client returns a remoteClient for the named state.
func (b *Backend) client(name string) (*remoteClient, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("state name not allow to be empty")
	}

	return &remoteClient{
		cosContext: b.cosContext,
		cosClient:  b.cosClient,
		tagClient:  b.tagClient,
		bucket:     b.bucket,
		stateFile:  b.stateFile(name),
		lockFile:   b.lockFile(name),
		encrypt:    b.encrypt,
		acl:        b.acl,
	}, nil
}

// stateFile returns state file path by name
func (b *Backend) stateFile(name string) string {
	if name == backend.DefaultStateName {
		return path.Join(b.prefix, b.key)
	}
	return path.Join(b.prefix, name, b.key)
}

// lockFile returns lock file path by name
func (b *Backend) lockFile(name string) string {
	return b.stateFile(name) + lockFileSuffix
}

// unlockErrMsg is error msg for unlock failed
const unlockErrMsg = `
Unlocking the state file on TencentCloud cos backend failed:

Error message: %v
Lock ID (gen): %s

You may have to force-unlock this state in order to use it again.
The TencentCloud backend acquires a lock during initialization
to ensure the initial state file is created.
`
