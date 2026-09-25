// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-uuid"
	"github.com/jackofallops/giovanni/storage/2023-11-03/blob/blobs"

	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

const (
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
	snapshot           bool
	lockInfo           *statemgr.LockInfo
}

func (c *RemoteClient) Get() (*remote.Payload, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ctx := newCtx()
	blob, err := c.giovanniBlobClient.Get(ctx, c.containerName, c.keyName, blobs.GetInput{})
	if err != nil {
		if response.WasNotFound(blob.HttpResponse) {
			return nil, nil
		}
		return nil, diags.Append(err)
	}

	if blob.Contents == nil {
		return nil, diags
	}

	payload := &remote.Payload{
		Data: *blob.Contents,
	}

	// If there was no data, then return nil
	if len(payload.Data) == 0 {
		return nil, diags
	}

	return payload, diags
}

func (c *RemoteClient) Put(data []byte) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	ctx := newCtx()

	if c.snapshot {
		snapshotInput := blobs.SnapshotInput{}
		if c.lockInfo != nil {
			snapshotInput.LeaseID = &c.lockInfo.ID
		}

		log.Printf("[DEBUG] Snapshotting existing Blob %q (Container %q / Account %q)", c.keyName, c.containerName, c.accountName)
		if _, err := c.giovanniBlobClient.Snapshot(ctx, c.containerName, c.keyName, snapshotInput); err != nil {
			return diags.Append(fmt.Errorf("error snapshotting Blob %q (Container %q / Account %q): %+v", c.keyName, c.containerName, c.accountName, err))
		}

		log.Print("[DEBUG] Created blob snapshot")
	}

	contentType := "application/json"
	putOptions := blobs.PutBlockBlobInput{
		Content:     &data,
		ContentType: &contentType,
	}
	if c.lockInfo != nil {
		putOptions.LeaseID = &c.lockInfo.ID
		putOptions.MetaData = map[string]string{
			lockInfoMetaKey: base64.StdEncoding.EncodeToString(c.lockInfo.Marshal()),
		}
	}
	if _, err := c.giovanniBlobClient.PutBlockBlob(ctx, c.containerName, c.keyName, putOptions); err != nil {
		return diags.Append(err)
	}

	return nil
}

func (c *RemoteClient) Delete() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	options := blobs.DeleteInput{}

	if c.lockInfo != nil {
		options.LeaseID = &c.lockInfo.ID
	}

	ctx := newCtx()
	resp, err := c.giovanniBlobClient.Delete(ctx, c.containerName, c.keyName, options)
	if err != nil {
		if !response.WasNotFound(resp.HttpResponse) {
			return diags.Append(err)
		}
	}
	return diags
}

func (c *RemoteClient) Lock(info *statemgr.LockInfo) (string, error) {
	stateName := fmt.Sprintf("%s/%s", c.containerName, c.keyName)
	info.Path = stateName

	proposedLockID := info.ID
	if proposedLockID == "" {
		var err error
		proposedLockID, err = uuid.GenerateUUID()
		if err != nil {
			return "", err
		}
	}

	// This error wrap function is to return a statemgr.LockError in case the blob is locked by someone else.
	// The statemgr.LockError will then result into retry if -lock-timeout is specified.
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
		ProposedLeaseID: &proposedLockID,
		LeaseDuration:   -1,
	}
	ctx := newCtx()

	resp, err := c.giovanniBlobClient.AcquireLease(ctx, c.containerName, c.keyName, leaseOptions)
	if err != nil {
		if resp.HttpResponse == nil {
			return "", err
		}
		switch resp.HttpResponse.StatusCode {
		case http.StatusNotFound:
			// This indicates the state blob not exists yet, need to create it first.
			// Note that in this case, there is still a window that someone else create and lock the same blob,
			// hence we need to try to wrap the error to be statemgr.LockInfo.
			contentType := "application/json"
			putGOptions := blobs.PutBlockBlobInput{
				ContentType: &contentType,
			}

			_, err = c.giovanniBlobClient.PutBlockBlob(ctx, c.containerName, c.keyName, putGOptions)
			if err != nil {
				return "", getLockInfoErr(err)
			}

			resp, err = c.giovanniBlobClient.AcquireLease(ctx, c.containerName, c.keyName, leaseOptions)
			if err != nil {
				return "", getLockInfoErr(err)
			}
		case http.StatusConflict:
			// This indicates the state blob is already locked.
			return "", getLockInfoErr(err)
		default:
			return "", err
		}
	}

	// Cache the lockinfo with the actual lock id (i.e. lease id)
	info.ID = resp.LeaseID
	c.lockInfo = info

	// Update the lock info in the blob metadata
	opts := blobs.SetMetaDataInput{
		LeaseID: &info.ID,
		MetaData: map[string]string{
			lockInfoMetaKey: base64.StdEncoding.EncodeToString(info.Marshal()),
		},
	}
	if _, err = c.giovanniBlobClient.SetMetaData(ctx, c.containerName, c.keyName, opts); err != nil {
		err = fmt.Errorf("failed to set metadata: %v", err)
		// Try to release the lock before error out
		if _, rerr := c.giovanniBlobClient.ReleaseLease(ctx, c.containerName, c.keyName, blobs.ReleaseLeaseInput{LeaseID: info.ID}); rerr != nil {
			err = errors.Join(err, fmt.Errorf("failed to lease %q, may need manual release: %w", info.ID, err))
		}
		return "", err
	}

	return info.ID, nil
}

func (c *RemoteClient) getLockInfo() (*statemgr.LockInfo, error) {
	ctx := newCtx()
	blob, err := c.giovanniBlobClient.GetProperties(ctx, c.containerName, c.keyName, blobs.GetPropertiesInput{})
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

func (c *RemoteClient) Unlock(id string) error {
	ctx := newCtx()

	if c.lockInfo != nil && c.lockInfo.ID != id {
		return fmt.Errorf("lock id %q does not match the current lock %q", id, c.lockInfo.ID)
	}

	// Clear the lockinfo from the blob metadata prior to release the lease.
	opts := blobs.SetMetaDataInput{
		LeaseID:  &id,
		MetaData: map[string]string{},
	}

	if _, err := c.giovanniBlobClient.SetMetaData(ctx, c.containerName, c.keyName, opts); err != nil {
		return fmt.Errorf("failed to clear lock info from metadata: %s", err)
	}

	if _, err := c.giovanniBlobClient.ReleaseLease(ctx, c.containerName, c.keyName, blobs.ReleaseLeaseInput{LeaseID: id}); err != nil {
		return err
	}

	c.lockInfo = nil

	return nil
}
