package obs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/huaweicloud/golangsdk/openstack/obs"
)

// ErrCode
const (
	// Access OBS client denied
	ErrCodeAccessDenied = "AccessDenied"

	// The specified bucket does not exist.
	ErrCodeNoSuchBucket = "NoSuchBucket"

	// The specified key does not exist.
	ErrCodeNoSuchKey = "NoSuchKey"
)

// RemoteClient implements the client of remote state
type RemoteClient struct {
	obsClient  *obs.ObsClient
	bucketName string
	stateFile  string
	lockFile   string
	acl        string
	encryption bool
	kmsKeyID   string
}

// Get remote state file
func (c *RemoteClient) Get() (*remote.Payload, error) {
	output, err := c.getObject(c.stateFile)
	if err != nil || output == nil {
		return nil, err
	}

	defer output.Body.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, output.Body); err != nil {
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

// Put state file to remote
func (c *RemoteClient) Put(data []byte) error {
	return c.putObject(c.stateFile, data)
}

// Delete remote state file
func (c *RemoteClient) Delete() error {
	return c.deleteObject(c.stateFile)
}

// Lock lock remote state file for writing
func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	log.Printf("[DEBUG] lock remote state file %s", c.lockFile)

	//firstly we want to check that a lock doesn't already exist
	lockErr := &state.LockError{}
	lockInfo, err := c.getLockInfo()
	if err != nil {
		lockErr.Err = fmt.Errorf("failed to retrieve lock info: %s", err)
		return "", lockErr
	}

	if lockInfo != nil {
		lockErr := &state.LockError{
			Err:  fmt.Errorf("A lock is already acquired"),
			Info: lockInfo,
		}
		return "", lockErr
	}

	// get the lock
	info.Path = c.lockFile
	if info.ID == "" {
		lockID, err := uuid.GenerateUUID()
		if err != nil {
			return "", err
		}

		info.ID = lockID
	}

	data, err := json.Marshal(info)
	if err != nil {
		return "", c.lockError(err)
	}

	err = c.putObject(c.lockFile, data)
	if err != nil {
		return "", c.lockError(err)
	}

	return info.ID, nil
}

// Unlock unlock remote state file
func (c *RemoteClient) Unlock(check string) error {
	log.Printf("[DEBUG] unlock remote state file %s", c.lockFile)

	info, err := c.getLockInfo()
	if err != nil {
		return c.lockError(err)
	}

	if info.ID != check {
		return c.lockError(fmt.Errorf("lock id mismatch, %v != %v", info.ID, check))
	}

	err = c.deleteObject(c.lockFile)
	if err != nil {
		return c.lockError(err)
	}

	return nil
}

// lockError returns state.LockError
func (c *RemoteClient) lockError(err error) *state.LockError {
	log.Printf("[WARN] failed to lock or unlock %s: %v", c.lockFile, err)

	lockErr := &state.LockError{
		Err: err,
	}

	info, infoErr := c.getLockInfo()
	if infoErr != nil {
		lockErr.Err = multierror.Append(lockErr.Err, infoErr)
	} else {
		lockErr.Info = info
	}

	return lockErr
}

// getLockInfo returns LockInfo from lock file
func (c *RemoteClient) getLockInfo() (*state.LockInfo, error) {
	output, err := c.getObject(c.lockFile)
	if err != nil || output == nil {
		return nil, err
	}

	defer output.Body.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, output.Body); err != nil {
		return nil, fmt.Errorf("Failed to read remote lock info: %s", err)
	}

	lockInfo := &state.LockInfo{}
	err = json.Unmarshal(buf.Bytes(), lockInfo)
	if err != nil {
		return nil, err
	}

	return lockInfo, nil
}

// getObject get remote object
func (c *RemoteClient) getObject(object string) (*obs.GetObjectOutput, error) {
	var output *obs.GetObjectOutput
	var err error

	input := &obs.GetObjectInput{}
	input.Bucket = c.bucketName
	input.Key = object

	log.Printf("[DEBUG] get remote object: %s in bucket: %s", object, c.bucketName)
	output, err = c.obsClient.GetObject(input)
	if err != nil {
		if obserr, ok := err.(obs.ObsError); ok {
			switch obserr.Code {
			case ErrCodeNoSuchBucket:
				return nil, fmt.Errorf(errNoSuchBucket, obserr)
			case ErrCodeNoSuchKey:
				return nil, nil
			}
		}
		return nil, err
	}

	return output, nil
}

// Put object to remote bucket
func (c *RemoteClient) putObject(object string, data []byte) error {
	i := &obs.PutObjectInput{}
	i.ContentType = "application/json"
	i.ContentLength = int64(len(data))
	i.Body = bytes.NewReader(data)
	i.Bucket = c.bucketName
	i.Key = object

	if c.encryption {
		sseKmsHeader := obs.SseKmsHeader{
			Encryption: obs.DEFAULT_SSE_KMS_ENCRYPTION,
		}
		if c.kmsKeyID != "" {
			sseKmsHeader.Key = c.kmsKeyID
		}
		i.SseHeader = sseKmsHeader
	}

	if c.acl != "" {
		i.ACL = obs.AclType(c.acl)
	}

	log.Printf("[DEBUG] upload object: %s to bucket: %s", object, c.bucketName)
	_, err := c.obsClient.PutObject(i)
	if err != nil {
		return fmt.Errorf("failed to upload object: %s", err)
	}

	return nil
}

// delete remote object
func (c *RemoteClient) deleteObject(object string) error {
	log.Printf("[DEBUG] delete remote state:%s in bucket:%s", object, c.bucketName)

	_, err := c.obsClient.DeleteObject(&obs.DeleteObjectInput{
		Bucket: c.bucketName,
		Key:    object,
	})

	if err != nil {
		return err
	}

	return nil
}

const errNoSuchBucket = `bucket does not exist.

The referenced bucket must have been previously created. If the bucket was
created within the last minute, please wait for a minute or two and try again.

Error: %s
`
