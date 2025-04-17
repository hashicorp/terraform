// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oci

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
)

const errStateUnlock = `
Error unlocking oci state. Lock ID: %s

Error: %s

You may have to force-unlock this state in order to use it again.
`

func (b *Backend) StateMgr(name string) (statemgr.Full, error) {
	b.client.path = b.path(name)
	b.client.lockFilePath = b.getLockFilePath(name)
	stateMgr := &remote.State{Client: &RemoteClient{
		objectStorageClient: b.client.objectStorageClient,
		bucketName:          b.bucket,
		path:                b.path(name),
		lockFilePath:        b.getLockFilePath(name),
		namespace:           b.namespace,
		kmsKeyID:            b.kmsKeyID,

		SSECustomerKey:       b.SSECustomerKey,
		SSECustomerKeySHA256: b.SSECustomerKeySHA256,
		SSECustomerAlgorithm: b.SSECustomerAlgorithm,
	}}
	// Check to see if this state already exists.
	// If we're trying to force-unlock a state, we can't take the lock before
	// fetching the state. If the state doesn't exist, we have to assume this
	// is a normal create operation, and take the lock at that point.
	//
	// If we need to force-unlock, but for some reason the state no longer
	// exists, the user will have to use aws tools to manually fix the
	// situation.
	existing, err := b.Workspaces()
	if err != nil {
		return nil, err
	}

	exists := false
	for _, s := range existing {
		if s == name {
			exists = true
			break
		}
	}

	// We need to create the object so it's listed by States.
	if !exists {
		// take a lock on this state while we write it
		lockInfo := statemgr.NewLockInfo()
		lockInfo.Operation = "init"
		lockId, err := b.client.Lock(lockInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to lock oci state: %s", err)
		}

		// Local helper function so we can call it multiple places
		lockUnlock := func(parent error) error {
			if err := stateMgr.Unlock(lockId); err != nil {
				return fmt.Errorf(strings.TrimSpace(errStateUnlock), lockId, err)
			}
			return parent
		}

		// Grab the value
		// This is to ensure that no one beat us to writing a state between
		// the `exists` check and taking the lock.
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

func (b *Backend) configureRemoteClient() error {

	configProvider, err := b.configProvider.getSdkConfigProvider()
	if err != nil {
		return err
	}

	client, err := buildConfigureClient(configProvider, buildHttpClient())
	if err != nil {
		return err
	}

	b.client = &RemoteClient{
		objectStorageClient: client,
		bucketName:          b.bucket,
		namespace:           b.namespace,
		kmsKeyID:            b.kmsKeyID,

		SSECustomerKey:       b.SSECustomerKey,
		SSECustomerKeySHA256: b.SSECustomerKeySHA256,
		SSECustomerAlgorithm: b.SSECustomerAlgorithm,
	}
	return nil
}

func (b *Backend) Workspaces() ([]string, error) {
	logger := logWithOperation("listWorkspaces")
	const maxKeys = 1000

	ctx := context.TODO()
	wss := []string{backend.DefaultStateName}
	start := common.String("")
	if b.client == nil {
		err := b.configureRemoteClient()
		if err != nil {
			return nil, err
		}
	}
	for {
		listObjectReq := objectstorage.ListObjectsRequest{
			BucketName:    common.String(b.bucket),
			NamespaceName: common.String(b.namespace),
			Prefix:        common.String(b.workspaceKeyPrefix),
			Start:         start,
			Limit:         common.Int(maxKeys),
		}
		listObjectResponse, err := b.client.objectStorageClient.ListObjects(ctx, listObjectReq)
		if err != nil {
			logger.Error("Failed to list workspaces in Object Storage backend: %v", err)
			return nil, err
		}

		for _, object := range listObjectResponse.Objects {
			key := *object.Name
			if strings.HasPrefix(key, b.workspaceKeyPrefix) && strings.HasSuffix(key, b.key) {
				name := strings.TrimPrefix(key, b.workspaceKeyPrefix+"/")
				name = strings.TrimSuffix(name, b.key)
				name = strings.TrimSuffix(name, "/")

				if name != "" {
					wss = append(wss, name)
				}
			}
		}
		if len(listObjectResponse.Objects) < maxKeys {
			break
		}
		start = listObjectResponse.NextStartWith

	}

	return uniqueStrings(wss), nil
}

func (b *Backend) DeleteWorkspace(name string, force bool) error {

	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}
	if b.client == nil {
		err := b.configureRemoteClient()
		if err != nil {
			return err
		}
	}

	b.client.path = b.path(name)
	b.client.lockFilePath = b.getLockFilePath(name)
	return b.client.Delete()

}
