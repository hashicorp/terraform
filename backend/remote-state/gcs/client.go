package gcs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	"cloud.google.com/go/storage"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"golang.org/x/net/context"
)

// remoteClient is used by "state/remote".State to read and write
// blobs representing state.
// Implements "state/remote".ClientLocker
type remoteClient struct {
	storageContext context.Context
	storageClient  *storage.Client
	bucketName     string
	stateFilePath  string
	lockFilePath   string
	encryptionKey  []byte
}

func (c *remoteClient) Get() (payload *remote.Payload, err error) {
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

func (c *remoteClient) Put(data []byte) error {
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

func (c *remoteClient) Delete() error {
	if err := c.stateFile().Delete(c.storageContext); err != nil {
		return fmt.Errorf("Failed to delete state file %v: %v", c.stateFileURL(), err)
	}

	return nil
}

// Lock writes to a lock file, ensuring file creation. Returns the generation
// number, which must be passed to Unlock().
func (c *remoteClient) Lock(info *state.LockInfo) (string, error) {
	// update the path we're using
	// we can't set the ID until the info is written
	info.Path = c.lockFileURL()

	infoJson, err := json.Marshal(info)
	if err != nil {
		return "", err
	}

	lockFile := c.lockFile()
	w := lockFile.If(storage.Conditions{DoesNotExist: true}).NewWriter(c.storageContext)
	err = func() error {
		if _, err := w.Write(infoJson); err != nil {
			return err
		}
		return w.Close()
	}()

	if err != nil {
		return "", c.lockError(fmt.Errorf("writing %q failed: %v", c.lockFileURL(), err))
	}

	info.ID = strconv.FormatInt(w.Attrs().Generation, 10)

	return info.ID, nil
}

func (c *remoteClient) Unlock(id string) error {
	gen, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}

	if err := c.lockFile().If(storage.Conditions{GenerationMatch: gen}).Delete(c.storageContext); err != nil {
		return c.lockError(err)
	}

	return nil
}

func (c *remoteClient) lockError(err error) *state.LockError {
	lockErr := &state.LockError{
		Err: err,
	}

	info, infoErr := c.lockInfo()
	if infoErr != nil {
		lockErr.Err = multierror.Append(lockErr.Err, infoErr)
	} else {
		lockErr.Info = info
	}
	return lockErr
}

// lockInfo reads the lock file, parses its contents and returns the parsed
// LockInfo struct.
func (c *remoteClient) lockInfo() (*state.LockInfo, error) {
	r, err := c.lockFile().NewReader(c.storageContext)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	rawData, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	info := &state.LockInfo{}
	if err := json.Unmarshal(rawData, info); err != nil {
		return nil, err
	}

	// We use the Generation as the ID, so overwrite the ID in the json.
	// This can't be written into the Info, since the generation isn't known
	// until it's written.
	attrs, err := c.lockFile().Attrs(c.storageContext)
	if err != nil {
		return nil, err
	}
	info.ID = strconv.FormatInt(attrs.Generation, 10)

	return info, nil
}

func (c *remoteClient) stateFile() *storage.ObjectHandle {
	h := c.storageClient.Bucket(c.bucketName).Object(c.stateFilePath)
	if len(c.encryptionKey) > 0 {
		return h.Key(c.encryptionKey)
	}
	return h
}

func (c *remoteClient) stateFileURL() string {
	return fmt.Sprintf("gs://%v/%v", c.bucketName, c.stateFilePath)
}

func (c *remoteClient) lockFile() *storage.ObjectHandle {
	return c.storageClient.Bucket(c.bucketName).Object(c.lockFilePath)
}

func (c *remoteClient) lockFileURL() string {
	return fmt.Sprintf("gs://%v/%v", c.bucketName, c.lockFilePath)
}
