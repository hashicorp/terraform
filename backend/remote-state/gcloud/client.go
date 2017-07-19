package gcloud

import (
	"cloud.google.com/go/storage"
	"encoding/json"
	"fmt"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"golang.org/x/net/context"
	"io/ioutil"
)

type RemoteClient struct {
	storageContext context.Context
	storageClient  *storage.Client
	bucketName     string
	stateFilePath  string
	lockFilePath   string
}

func (c *RemoteClient) Get() (payload *remote.Payload, err error) {
	bucket := c.storageClient.Bucket(c.bucketName)
	stateFile := bucket.Object(c.stateFilePath)
	stateFileUrl := c.stateFileUrl()

	stateFileReader, err := stateFile.NewReader(c.storageContext)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, nil
		} else {
			return nil, fmt.Errorf("Failed to open state file at %v: %v", stateFileUrl, err)
		}
	}
	defer stateFileReader.Close()

	stateFileContents, err := ioutil.ReadAll(stateFileReader)
	if err != nil {
		return nil, fmt.Errorf("Failed to read state file from %v: %v", stateFileUrl, err)
	}

	stateFileAttrs, err := stateFile.Attrs(c.storageContext)
	if err != nil {
		return nil, fmt.Errorf("Failed to read state file attrs from %v: %v", stateFileUrl, err)
	}

	result := &remote.Payload{
		Data: stateFileContents,
		MD5:  stateFileAttrs.MD5,
	}

	return result, nil
}

func (c *RemoteClient) Put(data []byte) error {
	bucket := c.storageClient.Bucket(c.bucketName)
	stateFile := bucket.Object(c.stateFilePath)

	stateFileWriter := stateFile.NewWriter(c.storageContext)

	stateFileWriter.Write(data)
	err := stateFileWriter.Close()

	if err != nil {
		return fmt.Errorf("Failed to upload state to %v: %v", c.stateFileUrl(), err)
	}

	return nil
}

func (c *RemoteClient) Delete() error {
	bucket := c.storageClient.Bucket(c.bucketName)
	stateFile := bucket.Object(c.stateFilePath)

	err := stateFile.Delete(c.storageContext)

	if err != nil {
		return fmt.Errorf("Failed to delete state file %v: %v", c.stateFileUrl(), err)
	}

	return nil
}

func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	if info.ID == "" {
		lockID, err := uuid.GenerateUUID()
		if err != nil {
			return "", err
		}

		info.ID = lockID
	}

	info.Path = c.lockFileUrl()

	infoJson, err := json.Marshal(info)
	if err != nil {
		return "", err
	}

	bucket := c.storageClient.Bucket(c.bucketName)
	lockFile := bucket.Object(c.lockFilePath)

	writer := lockFile.If(storage.Conditions{DoesNotExist: true}).NewWriter(c.storageContext)
	writer.Write(infoJson)
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("Error while saving lock file (%v): %v", info.Path, err)
	}

	return info.ID, nil
}

func (c *RemoteClient) Unlock(id string) error {
	lockErr := &state.LockError{}

	bucket := c.storageClient.Bucket(c.bucketName)
	lockFile := bucket.Object(c.lockFilePath)
	lockFileUrl := c.lockFileUrl()

	lockFileReader, err := lockFile.NewReader(c.storageContext)
	if err != nil {
		lockErr.Err = fmt.Errorf("Failed to retrieve lock info (%v): %v", lockFileUrl, err)
		return lockErr
	}
	defer lockFileReader.Close()

	lockFileContents, err := ioutil.ReadAll(lockFileReader)
	if err != nil {
		lockErr.Err = fmt.Errorf("Failed to retrieve lock info (%v): %v", lockFileUrl, err)
		return lockErr
	}

	lockInfo := &state.LockInfo{}
	err = json.Unmarshal(lockFileContents, lockInfo)
	if err != nil {
		lockErr.Err = fmt.Errorf("Failed to unmarshal lock info (%v): %v", lockFileUrl, err)
		return lockErr
	}

	lockErr.Info = lockInfo

	if lockInfo.ID != id {
		lockErr.Err = fmt.Errorf("Lock id %q does not match existing lock", id)
		return lockErr
	}

	lockFileAttrs, err := lockFile.Attrs(c.storageContext)
	if err != nil {
		lockErr.Err = fmt.Errorf("Failed to fetch lock file attrs (%v): %v", lockFileUrl, err)
		return lockErr
	}

	err = lockFile.If(storage.Conditions{GenerationMatch: lockFileAttrs.Generation}).Delete(c.storageContext)
	if err != nil {
		lockErr.Err = fmt.Errorf("Failed to delete lock file (%v): %v", lockFileUrl, err)
		return lockErr
	}

	return nil
}

func (c *RemoteClient) stateFileUrl() string {
	return fmt.Sprintf("gs://%v/%v", c.bucketName, c.stateFilePath)
}

func (c *RemoteClient) lockFileUrl() string {
	return fmt.Sprintf("gs://%v/%v", c.bucketName, c.lockFilePath)
}
