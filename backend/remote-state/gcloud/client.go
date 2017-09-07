package gcloud

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"cloud.google.com/go/storage"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"golang.org/x/net/context"
)

type RemoteClient struct {
	storageContext context.Context
	storageClient  *storage.Client
	bucketName     string
	stateFilePath  string
	lockFilePath   string
}

func (c *RemoteClient) Get() (payload *remote.Payload, err error) {
	stateFileReader, err := c.stateFile().NewReader(c.storageContext)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, nil
		} else {
			return nil, fmt.Errorf("Failed to open state file at %v: %v", c.stateFileURL(), err)
		}
	}
	defer stateFileReader.Close()

	stateFileContents, err := ioutil.ReadAll(stateFileReader)
	if err != nil {
		return nil, fmt.Errorf("Failed to read state file from %v: %v", c.stateFileURL(), err)
	}

	stateFileAttrs, err := c.stateFile().Attrs(c.storageContext)
	if err != nil {
		return nil, fmt.Errorf("Failed to read state file attrs from %v: %v", c.stateFileURL(), err)
	}

	result := &remote.Payload{
		Data: stateFileContents,
		MD5:  stateFileAttrs.MD5,
	}

	return result, nil
}

func (c *RemoteClient) Put(data []byte) error {
	err := func() error {
		stateFileWriter := c.stateFile().NewWriter(c.storageContext)
		if _, err := stateFileWriter.Write(data); err != nil {
			return err
		}
		return stateFileWriter.Close()
	}()
	if err != nil {
		return fmt.Errorf("Failed to upload state to %v: %v", c.stateFileURL(), err)
	}

	return nil
}

func (c *RemoteClient) Delete() error {
	if err := c.stateFile().Delete(c.storageContext); err != nil {
		return fmt.Errorf("Failed to delete state file %v: %v", c.stateFileURL(), err)
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

	info.Path = c.lockFileURL()

	infoJson, err := json.Marshal(info)
	if err != nil {
		return "", err
	}

	writer := c.lockFile().If(storage.Conditions{DoesNotExist: true}).NewWriter(c.storageContext)
	writer.Write(infoJson)
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("Error while saving lock file (%v): %v", info.Path, err)
	}

	return info.ID, nil
}

func (c *RemoteClient) Unlock(id string) error {
	lockErr := &state.LockError{}

	lockFileReader, err := c.lockFile().NewReader(c.storageContext)
	if err != nil {
		lockErr.Err = fmt.Errorf("Failed to retrieve lock info (%v): %v", c.lockFileURL(), err)
		return lockErr
	}
	defer lockFileReader.Close()

	lockFileContents, err := ioutil.ReadAll(lockFileReader)
	if err != nil {
		lockErr.Err = fmt.Errorf("Failed to retrieve lock info (%v): %v", c.lockFileURL(), err)
		return lockErr
	}

	lockInfo := &state.LockInfo{}
	err = json.Unmarshal(lockFileContents, lockInfo)
	if err != nil {
		lockErr.Err = fmt.Errorf("Failed to unmarshal lock info (%v): %v", c.lockFileURL(), err)
		return lockErr
	}

	lockErr.Info = lockInfo

	if lockInfo.ID != id {
		lockErr.Err = fmt.Errorf("Lock id %q does not match existing lock", id)
		return lockErr
	}

	lockFileAttrs, err := lockFile.Attrs(c.storageContext)
	if err != nil {
		lockErr.Err = fmt.Errorf("Failed to fetch lock file attrs (%v): %v", c.lockFileURL(), err)
		return lockErr
	}

	err = lockFile.If(storage.Conditions{GenerationMatch: lockFileAttrs.Generation}).Delete(c.storageContext)
	if err != nil {
		lockErr.Err = fmt.Errorf("Failed to delete lock file (%v): %v", c.lockFileURL(), err)
		return lockErr
	}

	return nil
}

func (c *RemoteClient) stateFile() *storage.ObjectHandle {
	return c.storageClient.Bucket(c.bucketName).Object(c.stateFilePath)
}

func (c *RemoteClient) stateFileURL() string {
	return fmt.Sprintf("gs://%v/%v", c.bucketName, c.stateFilePath)
}

func (c *RemoteClient) lockFile() *storage.ObjectHandle {
	return c.storageClient.Bucket(c.bucketName).Object(c.lockFilePath)
}

func (c *RemoteClient) lockFileURL() string {
	return fmt.Sprintf("gs://%v/%v", c.bucketName, c.lockFilePath)
}
