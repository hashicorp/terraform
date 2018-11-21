package azure

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/states"
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
	containerReference := c.blobClient.GetContainerReference(c.containerName)
	blobReference := containerReference.GetBlobReference(c.keyName)
	options := &storage.GetBlobOptions{}

	if c.leaseID != "" {
		options.LeaseID = c.leaseID
	}

	blob, err := blobReference.Get(options)
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
	getOptions := &storage.GetBlobMetadataOptions{}
	setOptions := &storage.SetBlobPropertiesOptions{}
	putOptions := &storage.PutBlobOptions{}

	containerReference := c.blobClient.GetContainerReference(c.containerName)
	blobReference := containerReference.GetBlobReference(c.keyName)

	blobReference.Properties.ContentType = "application/json"
	blobReference.Properties.ContentLength = int64(len(data))

	if c.leaseID != "" {
		getOptions.LeaseID = c.leaseID
		setOptions.LeaseID = c.leaseID
		putOptions.LeaseID = c.leaseID
	}

	exists, err := blobReference.Exists()
	if err != nil {
		return err
	}

	if exists {
		err = blobReference.GetMetadata(getOptions)
		if err != nil {
			return err
		}
	}

	reader := bytes.NewReader(data)

	err = blobReference.CreateBlockBlobFromReader(reader, putOptions)
	if err != nil {
		return err
	}

	return blobReference.SetProperties(setOptions)
}

func (c *RemoteClient) Delete() error {
	containerReference := c.blobClient.GetContainerReference(c.containerName)
	blobReference := containerReference.GetBlobReference(c.keyName)
	options := &storage.DeleteBlobOptions{}

	if c.leaseID != "" {
		options.LeaseID = c.leaseID
	}

	return blobReference.Delete(options)
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

	containerReference := c.blobClient.GetContainerReference(c.containerName)
	blobReference := containerReference.GetBlobReference(c.keyName)
	leaseID, err := blobReference.AcquireLease(-1, info.ID, &storage.LeaseOptions{})
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
			if err := stateMgr.WriteState(states.NewState()); err != nil {
				return "", fmt.Errorf("Failed to write empty state for locking: %s", err)
			}
			if err := stateMgr.PersistState(); err != nil {
				return "", fmt.Errorf("Failed to persist empty state for locking: %s", err)
			}
		}

		leaseID, err = blobReference.AcquireLease(-1, info.ID, &storage.LeaseOptions{})
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
	containerReference := c.blobClient.GetContainerReference(c.containerName)
	blobReference := containerReference.GetBlobReference(c.keyName)
	err := blobReference.GetMetadata(&storage.GetBlobMetadataOptions{})
	if err != nil {
		return nil, err
	}

	raw := blobReference.Metadata[lockInfoMetaKey]
	if raw == "" {
		return nil, fmt.Errorf("blob metadata %q was empty", lockInfoMetaKey)
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
	containerReference := c.blobClient.GetContainerReference(c.containerName)
	blobReference := containerReference.GetBlobReference(c.keyName)
	err := blobReference.GetMetadata(&storage.GetBlobMetadataOptions{
		LeaseID: c.leaseID,
	})
	if err != nil {
		return err
	}

	if info == nil {
		delete(blobReference.Metadata, lockInfoMetaKey)
	} else {
		value := base64.StdEncoding.EncodeToString(info.Marshal())
		blobReference.Metadata[lockInfoMetaKey] = value
	}

	opts := &storage.SetBlobMetadataOptions{
		LeaseID: c.leaseID,
	}
	return blobReference.SetMetadata(opts)
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

	containerReference := c.blobClient.GetContainerReference(c.containerName)
	blobReference := containerReference.GetBlobReference(c.keyName)
	err = blobReference.ReleaseLease(id, &storage.LeaseOptions{})
	if err != nil {
		lockErr.Err = err
		return lockErr
	}

	c.leaseID = ""

	return nil
}
