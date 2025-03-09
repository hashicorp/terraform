// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-uuid"
	"github.com/jackofallops/giovanni/storage/2023-11-03/blob/blobs"

	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

const (
	leaseHeader = "x-ms-lease-id"
	// Must be lower case
	lockInfoMetaKey = "terraformlockid"
)

const veryLongTimeout = 9999 * time.Hour

// newCtx creates a context with a (meaningless) deadline.
// This is only to make the go-azure-sdk/sdk/client Client happy.
func newCtx() context.Context {
	ctx, _ := context.WithTimeout(context.TODO(), veryLongTimeout)
	return ctx
}

type RemoteClient struct {
	giovanniBlobClient blobs.Client
	accountName        string
	containerName      string
	keyName            string
	leaseID            string
	snapshot           bool
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	options := blobs.GetInput{}
	if c.leaseID != "" {
		options.LeaseID = &c.leaseID
	}

	ctx := newCtx()
	blob, err := c.giovanniBlobClient.Get(ctx, c.containerName, c.keyName, options)
	if err != nil {
		if response.WasNotFound(blob.HttpResponse) {
			return nil, nil
		}
		return nil, err
	}

	if blob.Contents == nil {
		return nil, nil
	}

	payload := &remote.Payload{
		Data: *blob.Contents,
	}

	// If there was no data, then return nil
	if len(payload.Data) == 0 {
		return nil, nil
	}

	return payload, nil
}

func (c *RemoteClient) Put(data []byte) error {
	getOptions := blobs.GetPropertiesInput{}
	setOptions := blobs.SetPropertiesInput{}
	putOptions := blobs.PutBlockBlobInput{}

	options := blobs.GetInput{}
	if c.leaseID != "" {
		options.LeaseID = &c.leaseID
		getOptions.LeaseID = &c.leaseID
		setOptions.LeaseID = &c.leaseID
		putOptions.LeaseID = &c.leaseID
	}

	ctx := newCtx()

	if c.snapshot {
		snapshotInput := blobs.SnapshotInput{LeaseID: options.LeaseID}

		log.Printf("[DEBUG] Snapshotting existing Blob %q (Container %q / Account %q)", c.keyName, c.containerName, c.accountName)
		if _, err := c.giovanniBlobClient.Snapshot(ctx, c.containerName, c.keyName, snapshotInput); err != nil {
			return fmt.Errorf("error snapshotting Blob %q (Container %q / Account %q): %+v", c.keyName, c.containerName, c.accountName, err)
		}

		log.Print("[DEBUG] Created blob snapshot")
	}

	blob, err := c.giovanniBlobClient.GetProperties(ctx, c.containerName, c.keyName, getOptions)
	if err != nil {
		if !response.WasNotFound(blob.HttpResponse) {
			return err
		}
	}

	contentType := "application/json"
	putOptions.Content = &data
	putOptions.ContentType = &contentType
	putOptions.MetaData = blob.MetaData
	_, err = c.giovanniBlobClient.PutBlockBlob(ctx, c.containerName, c.keyName, putOptions)

	return err
}

func (c *RemoteClient) Delete() error {
	options := blobs.DeleteInput{}

	if c.leaseID != "" {
		options.LeaseID = &c.leaseID
	}

	ctx := newCtx()
	resp, err := c.giovanniBlobClient.Delete(ctx, c.containerName, c.keyName, options)
	if err != nil {
		if !response.WasNotFound(resp.HttpResponse) {
			return err
		}
	}
	return nil
}

func (c *RemoteClient) Lock(info *statemgr.LockInfo) (string, error) {
	stateName := fmt.Sprintf("%s/%s", c.containerName, c.keyName)
	info.Path = stateName

	if info.ID == "" {
		lockID, err := uuid.GenerateUUID()
		if err != nil {
			return "", err
		}

		info.ID = lockID
	}

	getLockInfoErr := func(err error) error {
		lockInfo, infoErr := c.getLockInfo()
		if infoErr != nil {
			err = errors.Join(err, infoErr)
		}

		return &statemgr.LockError{
			Err:  err,
			Info: lockInfo,
		}
	}

	leaseOptions := blobs.AcquireLeaseInput{
		ProposedLeaseID: &info.ID,
		LeaseDuration:   -1,
	}
	ctx := newCtx()

	// obtain properties to see if the blob lease is already in use. If the blob doesn't exist, create it
	properties, err := c.giovanniBlobClient.GetProperties(ctx, c.containerName, c.keyName, blobs.GetPropertiesInput{})
	if err != nil {
		// error if we had issues getting the blob
		if !response.WasNotFound(properties.HttpResponse) {
			return "", getLockInfoErr(err)
		}
		// if we don't find the blob, we need to build it

		contentType := "application/json"
		putGOptions := blobs.PutBlockBlobInput{
			ContentType: &contentType,
		}

		_, err = c.giovanniBlobClient.PutBlockBlob(ctx, c.containerName, c.keyName, putGOptions)
		if err != nil {
			return "", getLockInfoErr(err)
		}
	}

	// if the blob is already locked then error
	if properties.LeaseStatus == blobs.Locked {
		return "", getLockInfoErr(fmt.Errorf("state blob is already locked"))
	}

	leaseID, err := c.giovanniBlobClient.AcquireLease(ctx, c.containerName, c.keyName, leaseOptions)
	if err != nil {
		return "", getLockInfoErr(err)
	}

	info.ID = leaseID.LeaseID
	c.leaseID = leaseID.LeaseID

	if err := c.writeLockInfo(info); err != nil {
		return "", err
	}

	return info.ID, nil
}

func (c *RemoteClient) getLockInfo() (*statemgr.LockInfo, error) {
	options := blobs.GetPropertiesInput{}
	if c.leaseID != "" {
		options.LeaseID = &c.leaseID
	}

	ctx := newCtx()
	blob, err := c.giovanniBlobClient.GetProperties(ctx, c.containerName, c.keyName, options)
	if err != nil {
		return nil, err
	}

	raw := blob.MetaData[lockInfoMetaKey]
	if raw == "" {
		return nil, fmt.Errorf("blob metadata %q was empty", lockInfoMetaKey)
	}

	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}

	lockInfo := &statemgr.LockInfo{}
	err = json.Unmarshal(data, lockInfo)
	if err != nil {
		return nil, err
	}

	return lockInfo, nil
}

// writes info to blob meta data, deletes metadata entry if info is nil
func (c *RemoteClient) writeLockInfo(info *statemgr.LockInfo) error {
	ctx := newCtx()
	blob, err := c.giovanniBlobClient.GetProperties(ctx, c.containerName, c.keyName, blobs.GetPropertiesInput{LeaseID: &c.leaseID})
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}

	if info == nil {
		delete(blob.MetaData, lockInfoMetaKey)
	} else {
		value := base64.StdEncoding.EncodeToString(info.Marshal())
		blob.MetaData[lockInfoMetaKey] = value
	}

	opts := blobs.SetMetaDataInput{
		LeaseID:  &c.leaseID,
		MetaData: blob.MetaData,
	}

	_, err = c.giovanniBlobClient.SetMetaData(ctx, c.containerName, c.keyName, opts)
	return err
}

func (c *RemoteClient) Unlock(id string) error {
	lockErr := &statemgr.LockError{}

	lockInfo, err := c.getLockInfo()
	if err != nil {
		lockErr.Err = fmt.Errorf("failed to retrieve lock info: %s", err)
		return lockErr
	}
	lockErr.Info = lockInfo

	if lockInfo.ID != id {
		lockErr.Err = fmt.Errorf("lock id %q does not match existing lock", id)
		return lockErr
	}

	c.leaseID = lockInfo.ID
	if err := c.writeLockInfo(nil); err != nil {
		lockErr.Err = fmt.Errorf("failed to delete lock info from metadata: %s", err)
		return lockErr
	}

	ctx := newCtx()
	_, err = c.giovanniBlobClient.ReleaseLease(ctx, c.containerName, c.keyName, blobs.ReleaseLeaseInput{LeaseID: id})
	if err != nil {
		lockErr.Err = err
		return lockErr
	}

	c.leaseID = ""

	return nil
}
