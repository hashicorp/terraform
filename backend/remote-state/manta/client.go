package manta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"path"

	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	tritonErrors "github.com/joyent/triton-go/errors"
	"github.com/joyent/triton-go/storage"
)

const (
	mantaDefaultRootStore = "/stor"
	lockFileName          = "tflock"
)

type RemoteClient struct {
	storageClient *storage.StorageClient
	directoryName string
	keyName       string
	statePath     string
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	output, err := c.storageClient.Objects().Get(context.Background(), &storage.GetObjectInput{
		ObjectPath: path.Join(mantaDefaultRootStore, c.directoryName, c.keyName),
	})
	if err != nil {
		if tritonErrors.IsResourceNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	defer output.ObjectReader.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, output.ObjectReader); err != nil {
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
	contentType := "application/json"
	contentLength := int64(len(data))

	params := &storage.PutObjectInput{
		ContentType:   contentType,
		ContentLength: uint64(contentLength),
		ObjectPath:    path.Join(mantaDefaultRootStore, c.directoryName, c.keyName),
		ObjectReader:  bytes.NewReader(data),
	}

	log.Printf("[DEBUG] Uploading remote state to Manta: %#v", params)
	err := c.storageClient.Objects().Put(context.Background(), params)
	if err != nil {
		return err
	}

	return nil
}

func (c *RemoteClient) Delete() error {
	err := c.storageClient.Objects().Delete(context.Background(), &storage.DeleteObjectInput{
		ObjectPath: path.Join(mantaDefaultRootStore, c.directoryName, c.keyName),
	})

	return err
}

func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	//At Joyent, we want to make sure that the State directory exists before we interact with it
	//We don't expect users to have to create it in advance
	//The order of operations of Backend State as follows:
	// * Get - if this doesn't exist then we continue as though it's new
	// * Lock - we make sure that the state directory exists as it's the entrance to writing to Manta
	// * Put - put the state up there
	// * Unlock - unlock the directory
	//We can always guarantee that the user can put their state in the specified location because of this
	err := c.storageClient.Dir().Put(context.Background(), &storage.PutDirectoryInput{
		DirectoryName: path.Join(mantaDefaultRootStore, c.directoryName),
	})
	if err != nil {
		return "", err
	}

	//firstly we want to check that a lock doesn't already exist
	lockErr := &state.LockError{}
	lockInfo, err := c.getLockInfo()
	if err != nil {
		if tritonErrors.IsResourceNotFound(err) {
			lockErr.Err = fmt.Errorf("failed to retrieve lock info: %s", err)
			return "", lockErr
		}
	}

	if lockInfo != nil {
		lockErr := &state.LockError{
			Err:  fmt.Errorf("A lock is already acquired"),
			Info: lockInfo,
		}
		return "", lockErr
	}

	info.Path = path.Join(c.directoryName, lockFileName)

	if info.ID == "" {
		lockID, err := uuid.GenerateUUID()
		if err != nil {
			return "", err
		}

		info.ID = lockID
	}

	data := info.Marshal()

	contentType := "application/json"
	contentLength := int64(len(data))

	params := &storage.PutObjectInput{
		ContentType:   contentType,
		ContentLength: uint64(contentLength),
		ObjectPath:    path.Join(mantaDefaultRootStore, c.directoryName, lockFileName),
		ObjectReader:  bytes.NewReader(data),
	}

	log.Printf("[DEBUG] Creating manta state lock: %#v", params)
	err = c.storageClient.Objects().Put(context.Background(), params)
	if err != nil {
		return "", err
	}

	return info.ID, nil
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

	err = c.storageClient.Objects().Delete(context.Background(), &storage.DeleteObjectInput{
		ObjectPath: path.Join(mantaDefaultRootStore, c.directoryName, lockFileName),
	})

	return err
}

func (c *RemoteClient) getLockInfo() (*state.LockInfo, error) {
	output, err := c.storageClient.Objects().Get(context.Background(), &storage.GetObjectInput{
		ObjectPath: path.Join(mantaDefaultRootStore, c.directoryName, lockFileName),
	})
	if err != nil {
		return nil, err
	}

	defer output.ObjectReader.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, output.ObjectReader); err != nil {
		return nil, fmt.Errorf("Failed to read lock info: %s", err)
	}

	lockInfo := &state.LockInfo{}
	err = json.Unmarshal(buf.Bytes(), lockInfo)
	if err != nil {
		return nil, err
	}

	return lockInfo, nil
}
