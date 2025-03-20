// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oci

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
	"strings"
)

func (b *Backend) StateMgr(name string) (statemgr.Full, error) {

	b.client.path = b.path(name)
	b.client.lockFilePath = b.getLockFilePath(name)
	return &remote.State{Client: b.client}, nil
}

func (b *Backend) configureRemoteClient() error {

	configProvider, err := b.configProvider.getSdkConfigProvider()
	if err != nil {
		return err
	}
	common.SetSDKLogger(logger)

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
	logger.Info("Deleting workspace")
	b.client.path = b.path(name)
	b.client.lockFilePath = b.getLockFilePath(name)
	return b.client.Delete()

}
