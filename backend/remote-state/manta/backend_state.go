package manta

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	tritonErrors "github.com/joyent/triton-go/errors"
	"github.com/joyent/triton-go/storage"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/states"
)

func (b *Backend) Workspaces() ([]string, error) {
	result := []string{backend.DefaultStateName}

	objs, err := b.storageClient.Dir().List(context.Background(), &storage.ListDirectoryInput{
		DirectoryName: path.Join(mantaDefaultRootStore, b.path),
	})
	if err != nil {
		if tritonErrors.IsResourceNotFound(err) {
			return result, nil
		}
		return nil, err
	}

	for _, obj := range objs.Entries {
		if obj.Type == "directory" && obj.Name != "" {
			result = append(result, obj.Name)
		}
	}

	sort.Strings(result[1:])
	return result, nil
}

func (b *Backend) DeleteWorkspace(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	//firstly we need to delete the state file
	err := b.storageClient.Objects().Delete(context.Background(), &storage.DeleteObjectInput{
		ObjectPath: path.Join(mantaDefaultRootStore, b.statePath(name), b.objectName),
	})
	if err != nil {
		return err
	}

	//then we need to delete the state folder
	err = b.storageClient.Objects().Delete(context.Background(), &storage.DeleteObjectInput{
		ObjectPath: path.Join(mantaDefaultRootStore, b.statePath(name)),
	})
	if err != nil {
		return err
	}

	return nil
}

func (b *Backend) StateMgr(name string) (state.State, error) {
	if name == "" {
		return nil, errors.New("missing state name")
	}

	client := &RemoteClient{
		storageClient: b.storageClient,
		directoryName: b.statePath(name),
		keyName:       b.objectName,
	}

	stateMgr := &remote.State{Client: client}

	//if this isn't the default state name, we need to create the object so
	//it's listed by States.
	if name != backend.DefaultStateName {
		// take a lock on this state while we write it
		lockInfo := state.NewLockInfo()
		lockInfo.Operation = "init"
		lockId, err := client.Lock(lockInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to lock manta state: %s", err)
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

		// If we have no state, we have to create an empty state
		if v := stateMgr.State(); v == nil {
			if err := stateMgr.WriteState(states.NewState()); err != nil {
				err = lockUnlock(err)
				return nil, err
			}
			if err := stateMgr.PersistState(); err != nil {
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

func (b *Backend) client() *RemoteClient {
	return &RemoteClient{}
}

func (b *Backend) statePath(name string) string {
	if name == backend.DefaultStateName {
		return b.path
	}

	return path.Join(b.path, name)
}

const errStateUnlock = `
Error unlocking Manta state. Lock ID: %s

Error: %s

You may have to force-unlock this state in order to use it again.
`
