package azure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"encoding/base64"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	multierror "github.com/hashicorp/go-multierror"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

const (
	leaseHeader = "x-ms-lease-id"
	// Must be lower case
	lockInfoMetaKey = "terraformlockid"
)

type RemoteClient struct {
	blobClient    storage.BlobStorageClient
	containerName string
	keyName       string
	leaseID       string
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	blob, err := c.blobClient.GetBlob(c.containerName, c.keyName)
	if err != nil {
		if storErr, ok := err.(storage.AzureStorageServiceError); ok {
			if storErr.Code == "BlobNotFound" {
				return nil, nil
			}
		}
		return nil, err
	}

	defer blob.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, blob); err != nil {
		return nil, fmt.Errorf("Failed to read remote state: %s", err)
	}

	payload := &remote.Payload{
		Data: buf.Bytes(),
	}

	// If there was no data, then return nil
	if len(payload.Data) == 0 {
		return nil, nil
	}

	return payload, nil
}

func (c *RemoteClient) Put(data []byte) error {
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	if c.leaseID != "" {
		headers[leaseHeader] = c.leaseID
	}

	log.Print("[DEBUG] Uploading remote state to Azure")

	err := c.blobClient.CreateBlockBlobFromReader(
		c.containerName,
		c.keyName,
		uint64(len(data)),
		bytes.NewReader(data),
		headers,
	)

	if err != nil {
		return fmt.Errorf("Failed to upload state: %v", err)
	}

	return nil
}

func (c *RemoteClient) Delete() error {
	headers := map[string]string{}
	if c.leaseID != "" {
		headers[leaseHeader] = c.leaseID
	}

	return c.blobClient.DeleteBlob(c.containerName, c.keyName, headers)
}

func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
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
			err = multierror.Append(err, infoErr)
		}

		return &state.LockError{
			Err:  err,
			Info: lockInfo,
		}
	}

	leaseID, err := c.blobClient.AcquireLease(c.containerName, c.keyName, -1, info.ID)
	if err != nil {
		if storErr, ok := err.(storage.AzureStorageServiceError); ok && storErr.Code != "BlobNotFound" {
			return "", getLockInfoErr(err)
		}

		// failed to lock as there was no state blob, write empty state
		stateMgr := &remote.State{Client: c}

		// ensure state is actually empty
		if err := stateMgr.RefreshState(); err != nil {
			return "", fmt.Errorf("Failed to refresh state before writing empty state for locking: %s", err)
		}

		log.Print("[DEBUG] Could not lock as state blob did not exist, creating with empty state")

		if v := stateMgr.State(); v == nil {
			if err := stateMgr.WriteState(terraform.NewState()); err != nil {
				return "", fmt.Errorf("Failed to write empty state for locking: %s", err)
			}
			if err := stateMgr.PersistState(); err != nil {
				return "", fmt.Errorf("Failed to persist empty state for locking: %s", err)
			}
		}

		leaseID, err = c.blobClient.AcquireLease(c.containerName, c.keyName, -1, info.ID)
		if err != nil {
			return "", getLockInfoErr(err)
		}
	}

	info.ID = leaseID
	c.leaseID = leaseID

	if err := c.writeLockInfo(info); err != nil {
		return "", err
	}

	return info.ID, nil
}

func (c *RemoteClient) getLockInfo() (*state.LockInfo, error) {
	meta, err := c.blobClient.GetBlobMetadata(c.containerName, c.keyName)
	if err != nil {
		return nil, err
	}

	raw := meta[lockInfoMetaKey]
	if raw == "" {
		return nil, fmt.Errorf("blob metadata %s was empty", lockInfoMetaKey)
	}

	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}

	lockInfo := &state.LockInfo{}
	err = json.Unmarshal(data, lockInfo)
	if err != nil {
		return nil, err
	}

	return lockInfo, nil
}

// writes info to blob meta data, deletes metadata entry if info is nil
func (c *RemoteClient) writeLockInfo(info *state.LockInfo) error {
	meta, err := c.blobClient.GetBlobMetadata(c.containerName, c.keyName)
	if err != nil {
		return err
	}

	if info == nil {
		delete(meta, lockInfoMetaKey)
	} else {
		value := base64.StdEncoding.EncodeToString(info.Marshal())
		meta[lockInfoMetaKey] = value
	}

	headers := map[string]string{
		leaseHeader: c.leaseID,
	}
	return c.blobClient.SetBlobMetadata(c.containerName, c.keyName, meta, headers)

}

func (c *RemoteClient) Unlock(id string) error {
	lockErr := &state.LockError{}

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

	if err := c.writeLockInfo(nil); err != nil {
		lockErr.Err = fmt.Errorf("failed to delete lock info from metadata: %s", err)
		return lockErr
	}

	err = c.blobClient.ReleaseLease(c.containerName, c.keyName, id)
	if err != nil {
		lockErr.Err = err
		return lockErr
	}

	c.leaseID = ""

	return nil
}
