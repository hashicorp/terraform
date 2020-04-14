package azure

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/states"
	"github.com/tombuildsstuff/giovanni/storage/2018-11-09/blob/blobs"
)

const (
	leaseHeader = "x-ms-lease-id"
	// Must be lower case
	lockInfoMetaKey = "terraformlockid"
)

type RemoteClient struct {
	giovanniBlobClient blobs.Client
	accountName        string
	containerName      string
	keyName            string
	leaseID            string
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	options := blobs.GetInput{}
	if c.leaseID != "" {
		options.LeaseID = &c.leaseID
	}

	ctx := context.TODO()
	blob, err := c.giovanniBlobClient.Get(ctx, c.accountName, c.containerName, c.keyName, options)
	if err != nil {
		if blob.StatusCode == 404 {
			return nil, nil
		}
		return nil, err
	}

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, blob.Body); err != nil {
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
	getGOptions := blobs.GetPropertiesInput{}
	setGOptions := blobs.SetPropertiesInput{}
	putGOptions := blobs.PutBlockBlobInput{}

	options := blobs.GetInput{}
	if c.leaseID != "" {
		options.LeaseID = &c.leaseID
		getGOptions.LeaseID = &c.leaseID
		setGOptions.LeaseID = &c.leaseID
		putGOptions.LeaseID = &c.leaseID
	}

	ctx := context.TODO()
	_, err := c.giovanniBlobClient.GetProperties(ctx, c.accountName, c.containerName, c.keyName, getGOptions)
	if err != nil {
		return err
	}

	contentType := "application/json"
	putGOptions.Content = &data
	putGOptions.ContentType = &contentType
	_, err = c.giovanniBlobClient.PutBlockBlob(ctx, c.accountName, c.containerName, c.keyName, putGOptions)

	return err
}

func (c *RemoteClient) Delete() error {
	gOptions := blobs.DeleteInput{}

	if c.leaseID != "" {
		gOptions.LeaseID = &c.leaseID
	}

	ctx := context.TODO()
	_, err := c.giovanniBlobClient.Delete(ctx, c.accountName, c.containerName, c.keyName, gOptions)
	return err
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

	leaseOptions := blobs.AcquireLeaseInput{
		ProposedLeaseID: &info.ID,
	}
	ctx := context.TODO()
	leaseID, err := c.giovanniBlobClient.AcquireLease(ctx, c.accountName, c.containerName, c.keyName, leaseOptions)
	if err != nil {
		if leaseID.StatusCode != 404 {
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

		leaseID, err = c.giovanniBlobClient.AcquireLease(ctx, c.accountName, c.containerName, c.keyName, leaseOptions)
		if err != nil {
			return "", getLockInfoErr(err)
		}
	}

	info.ID = leaseID.LeaseID
	c.leaseID = leaseID.LeaseID

	if err := c.writeLockInfo(info); err != nil {
		return "", err
	}

	return info.ID, nil
}

func (c *RemoteClient) getLockInfo() (*state.LockInfo, error) {
	options := blobs.GetPropertiesInput{}
	if c.leaseID != "" {
		options.LeaseID = &c.leaseID
	}

	ctx := context.TODO()
	blob, err := c.giovanniBlobClient.GetProperties(ctx, c.accountName, c.containerName, c.keyName, options)
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

	lockInfo := &state.LockInfo{}
	err = json.Unmarshal(data, lockInfo)
	if err != nil {
		return nil, err
	}

	return lockInfo, nil
}

// writes info to blob meta data, deletes metadata entry if info is nil
func (c *RemoteClient) writeLockInfo(info *state.LockInfo) error {
	ctx := context.TODO()
	blob, err := c.giovanniBlobClient.GetProperties(ctx, c.accountName, c.containerName, c.keyName, blobs.GetPropertiesInput{LeaseID: &c.leaseID})
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

	_, err = c.giovanniBlobClient.SetMetaData(ctx, c.accountName, c.containerName, c.keyName, opts)
	return err
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

	c.leaseID = lockInfo.ID
	if err := c.writeLockInfo(nil); err != nil {
		lockErr.Err = fmt.Errorf("failed to delete lock info from metadata: %s", err)
		return lockErr
	}

	ctx := context.TODO()
	_, err = c.giovanniBlobClient.ReleaseLease(ctx, c.accountName, c.containerName, c.keyName, id)
	if err != nil {
		lockErr.Err = err
		return lockErr
	}

	c.leaseID = ""

	return nil
}
