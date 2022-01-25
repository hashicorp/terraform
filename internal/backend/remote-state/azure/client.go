package azure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-uuid"
	"github.com/tombuildsstuff/giovanni/storage/2018-11-09/blob/blobs"

	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
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
	snapshot           bool
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	options := blobs.GetInput{}
	if c.leaseID != "" {
		options.LeaseID = &c.leaseID
	}

	ctx := context.TODO()
	blob, err := c.giovanniBlobClient.Get(ctx, c.accountName, c.containerName, c.keyName, options)
	if err != nil {
		if blob.Response.IsHTTPStatus(http.StatusNotFound) {
			return nil, nil
		}
		return nil, err
	}

	payload := &remote.Payload{
		Data: blob.Contents,
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

	ctx := context.TODO()

	if c.snapshot {
		snapshotInput := blobs.SnapshotInput{LeaseID: options.LeaseID}

		log.Printf("[DEBUG] Snapshotting existing Blob %q (Container %q / Account %q)", c.keyName, c.containerName, c.accountName)
		if _, err := c.giovanniBlobClient.Snapshot(ctx, c.accountName, c.containerName, c.keyName, snapshotInput); err != nil {
			return fmt.Errorf("error snapshotting Blob %q (Container %q / Account %q): %+v", c.keyName, c.containerName, c.accountName, err)
		}

		log.Print("[DEBUG] Created blob snapshot")
	}

	blob, err := c.giovanniBlobClient.GetProperties(ctx, c.accountName, c.containerName, c.keyName, getOptions)
	if err != nil {
		if blob.StatusCode != 404 {
			return err
		}
	}

	contentType := "application/json"
	putOptions.Content = &data
	putOptions.ContentType = &contentType
	putOptions.MetaData = blob.MetaData
	_, err = c.giovanniBlobClient.PutBlockBlob(ctx, c.accountName, c.containerName, c.keyName, putOptions)

	return err
}

func (c *RemoteClient) Delete() error {
	options := blobs.DeleteInput{}

	if c.leaseID != "" {
		options.LeaseID = &c.leaseID
	}

	ctx := context.TODO()
	resp, err := c.giovanniBlobClient.Delete(ctx, c.accountName, c.containerName, c.keyName, options)
	if err != nil {
		if !resp.IsHTTPStatus(http.StatusNotFound) {
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
			err = multierror.Append(err, infoErr)
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
	ctx := context.TODO()

	// obtain properties to see if the blob lease is already in use. If the blob doesn't exist, create it
	properties, err := c.giovanniBlobClient.GetProperties(ctx, c.accountName, c.containerName, c.keyName, blobs.GetPropertiesInput{})
	if err != nil {
		// error if we had issues getting the blob
		if !properties.Response.IsHTTPStatus(http.StatusNotFound) {
			return "", getLockInfoErr(err)
		}
		// if we don't find the blob, we need to build it

		contentType := "application/json"
		putGOptions := blobs.PutBlockBlobInput{
			ContentType: &contentType,
		}

		_, err = c.giovanniBlobClient.PutBlockBlob(ctx, c.accountName, c.containerName, c.keyName, putGOptions)
		if err != nil {
			return "", getLockInfoErr(err)
		}
	}

	// if the blob is already locked then error
	if properties.LeaseStatus == blobs.Locked {
		return "", getLockInfoErr(fmt.Errorf("state blob is already locked"))
	}

	leaseID, err := c.giovanniBlobClient.AcquireLease(ctx, c.accountName, c.containerName, c.keyName, leaseOptions)
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

	lockInfo := &statemgr.LockInfo{}
	err = json.Unmarshal(data, lockInfo)
	if err != nil {
		return nil, err
	}

	return lockInfo, nil
}

// writes info to blob meta data, deletes metadata entry if info is nil
func (c *RemoteClient) writeLockInfo(info *statemgr.LockInfo) error {
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

	ctx := context.TODO()
	_, err = c.giovanniBlobClient.ReleaseLease(ctx, c.accountName, c.containerName, c.keyName, id)
	if err != nil {
		lockErr.Err = err
		return lockErr
	}

	c.leaseID = ""

	return nil
}
