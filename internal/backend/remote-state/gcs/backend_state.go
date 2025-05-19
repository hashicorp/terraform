// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gcs

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

const (
	stateFileSuffix = ".tfstate"
	lockFileSuffix  = ".tflock"
)

// Workspaces returns a list of names for the workspaces found on GCS. The default
// state is always returned as the first element in the slice.
func (b *Backend) Workspaces() ([]string, error) {
	ctx := context.TODO()

	states := []string{backend.DefaultStateName}

	bucket := b.storageClient.Bucket(b.bucketName)
	objs := bucket.Objects(ctx, &storage.Query{
		Delimiter: "/",
		Prefix:    b.prefix,
	})
	for {
		attrs, err := objs.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("querying Cloud Storage failed: %v", err)
		}

		name := path.Base(attrs.Name)
		if !strings.HasSuffix(name, stateFileSuffix) {
			continue
		}
		st := strings.TrimSuffix(name, stateFileSuffix)

		if st != backend.DefaultStateName {
			states = append(states, st)
		}
	}

	sort.Strings(states[1:])
	return states, nil
}

// DeleteWorkspace deletes the named workspaces. The "default" state cannot be deleted.
func (b *Backend) DeleteWorkspace(name string, _ bool) error {
	if name == backend.DefaultStateName {
		return fmt.Errorf("cowardly refusing to delete the %q state", name)
	}

	c, err := b.client(name)
	if err != nil {
		return err
	}

	return c.Delete()
}

// client returns a remoteClient for the named state.
func (b *Backend) client(name string) (*remoteClient, error) {
	if name == "" {
		return nil, fmt.Errorf("%q is not a valid state name", name)
	}

	return &remoteClient{
		storageClient: b.storageClient,
		bucketName:    b.bucketName,
		stateFilePath: b.stateFile(name),
		lockFilePath:  b.lockFile(name),
		encryptionKey: b.encryptionKey,
		kmsKeyName:    b.kmsKeyName,
	}, nil
}

// StateMgr reads and returns the named state from GCS. If the named state does
// not yet exist, a new state file is created.
func (b *Backend) StateMgr(name string) (statemgr.Full, error) {
	c, err := b.client(name)
	if err != nil {
		return nil, err
	}

	st := &remote.State{Client: c}

	// Grab the value
	if err := st.RefreshState(); err != nil {
		return nil, err
	}

	// If we have no state, we have to create an empty state
	if v := st.State(); v == nil {

		lockInfo := statemgr.NewLockInfo()
		lockInfo.Operation = "init"
		lockID, err := st.Lock(lockInfo)
		if err != nil {
			return nil, err
		}

		// Local helper function so we can call it multiple places
		unlock := func(baseErr error) error {
			if err := st.Unlock(lockID); err != nil {
				const unlockErrMsg = `%v
				Additionally, unlocking the state file on Google Cloud Storage failed:

				Error message: %q
				Lock ID (gen): %v
				Lock file URL: %v

				You may have to force-unlock this state in order to use it again.
				The GCloud backend acquires a lock during initialization to ensure
				the initial state file is created.`
				return fmt.Errorf(unlockErrMsg, baseErr, err.Error(), lockID, c.lockFileURL())
			}

			return baseErr
		}

		if err := st.WriteState(states.NewState()); err != nil {
			return nil, unlock(err)
		}
		if err := st.PersistState(nil); err != nil {
			return nil, unlock(err)
		}

		// Unlock, the state should now be initialized
		if err := unlock(nil); err != nil {
			return nil, err
		}

	}

	return st, nil
}

func (b *Backend) stateFile(name string) string {
	return path.Join(b.prefix, name+stateFileSuffix)
}

func (b *Backend) lockFile(name string) string {
	return path.Join(b.prefix, name+lockFileSuffix)
}
