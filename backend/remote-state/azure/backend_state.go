package azure

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/states"
)

const (
	// This will be used as directory name, the odd looking colon is simply to
	// reduce the chance of name conflicts with existing objects.
	keyEnvPrefix = "env:"
)

func (b *Backend) Workspaces() ([]string, error) {
	prefix := b.keyName + keyEnvPrefix
	params := storage.ListBlobsParameters{
		Prefix: prefix,
	}

	ctx := context.TODO()
	client, err := b.armClient.getBlobClient(ctx)
	if err != nil {
		return nil, err
	}
	container := client.GetContainerReference(b.containerName)
	resp, err := container.ListBlobs(params)
	if err != nil {
		return nil, err
	}

	envs := map[string]struct{}{}
	for _, obj := range resp.Blobs {
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

func (b *Backend) DeleteWorkspace(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	ctx := context.TODO()
	client, err := b.armClient.getBlobClient(ctx)
	if err != nil {
		return err
	}

	containerReference := client.GetContainerReference(b.containerName)
	blobReference := containerReference.GetBlobReference(b.path(name))
	options := &storage.DeleteBlobOptions{}

	return blobReference.Delete(options)
}

func (b *Backend) StateMgr(name string) (state.State, error) {
	ctx := context.TODO()
	blobClient, err := b.armClient.getBlobClient(ctx)
	if err != nil {
		return nil, err
	}

	client := &RemoteClient{
		blobClient:    *blobClient,
		containerName: b.containerName,
		keyName:       b.path(name),
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
