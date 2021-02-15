package azure

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/remote"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/tombuildsstuff/giovanni/storage/2018-11-09/blob/blobs"
	"github.com/tombuildsstuff/giovanni/storage/2018-11-09/blob/containers"
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

	ctx := context.TODO()
	client, err := b.armClient.getContainersClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.ListBlobs(ctx, b.armClient.storageAccountName, b.containerName, params)
	if err != nil {
		return nil, err
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

func (b *Backend) DeleteWorkspace(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	ctx := context.TODO()
	client, err := b.armClient.getBlobClient(ctx)
	if err != nil {
		return err
	}

	if resp, err := client.Delete(ctx, b.armClient.storageAccountName, b.containerName, b.path(name), blobs.DeleteInput{}); err != nil {
		if resp.Response.StatusCode != 404 {
			return err
		}
	}

	return nil
}

func (b *Backend) StateMgr(name string) (statemgr.Full, error) {
	ctx := context.TODO()
	blobClient, err := b.armClient.getBlobClient(ctx)
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

	if err := createNewStateIfNotExists(stateMgr); err != nil {
		return nil, err

	}
	return stateMgr, nil
}

func createNewStateIfNotExists(stateMgr *remote.State) error {
	// In Azure implementation lock info is stored as a lease on a state blob itself
	// this means that we don't need to deal with locks when the state blob does not exist yet
	if err := stateMgr.RefreshState(); err != nil {
		return err
	}

	// If we have no state, we have to create an empty state
	if v := stateMgr.State(); v == nil {
		if err := stateMgr.WriteState(states.NewState()); err != nil {
			return err
		}
		if err := stateMgr.PersistState(); err != nil {
			return err
		}
	}
	return nil
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
