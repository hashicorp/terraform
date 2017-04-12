package s3

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

/*
DEPRECATED FILE
TODO: remove after 0.10
these methods are only used to update the env format in S3
*/

const (
	oldEnvPrefix = "env:"
)

func (b *Backend) oldStates() ([]string, error) {
	params := &s3.ListObjectsInput{
		Bucket: &b.bucketName,
		Prefix: aws.String(oldEnvPrefix + "/"),
	}

	resp, err := b.s3Client.ListObjects(params)
	if err != nil {
		return nil, err
	}

	var envs []string
	for _, obj := range resp.Contents {
		env := oldKeyEnv(*obj.Key)
		if env != "" {
			envs = append(envs, env)
		}
	}

	// don't add "default" here
	sort.Strings(envs)
	return envs, nil
}

// extract the env name from the S3 key
func oldKeyEnv(key string) string {
	parts := strings.Split(key, "/")
	if len(parts) < 3 {
		// no env here
		return ""
	}

	if parts[0] != oldEnvPrefix {
		// not our key, so ignore
		return ""
	}

	return parts[1]
}

func (b *Backend) oldDeleteState(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	params := &s3.DeleteObjectInput{
		Bucket: &b.bucketName,
		Key:    aws.String(b.oldPath(name)),
	}

	_, err := b.s3Client.DeleteObject(params)
	if err != nil {
		return err
	}

	return nil
}

func (b *Backend) oldState(name string) (state.State, error) {
	client := &envUpgrader{
		RemoteClient: &RemoteClient{
			s3Client:             b.s3Client,
			dynClient:            b.dynClient,
			bucketName:           b.bucketName,
			path:                 b.oldPath(name),
			serverSideEncryption: b.serverSideEncryption,
			acl:                  b.acl,
			kmsKeyID:             b.kmsKeyID,
			lockTable:            b.lockTable,
		},
		backend:      b,
		name:         name,
		needsUpgrade: true,
	}

	// No need for the initialization code, since we know this must already exist.
	return &remote.State{Client: client}, nil
}

func (b *Backend) oldPath(name string) string {
	if name == backend.DefaultStateName {
		return b.keyName
	}
	return strings.Join([]string{oldEnvPrefix, name, b.keyName}, "/")
}

// envUpgrader wraps the Backend and RemoteClient to update a named state's
// location when there's a write.
type envUpgrader struct {
	*RemoteClient
	// we need the backend to help with the upgrade
	backend *Backend

	// only run upgrade onnce
	needsUpgrade bool

	// the name of this state
	name string

	// store the original lockInfo and original lock ID to translate the lock
	// to the new path.
	oldLockID string
	lockInfo  *state.LockInfo
}

// we upgrade on write, so that we know the user is expected to have write
// permissions, and we have a lock
func (c *envUpgrader) Put(data []byte) error {
	if c.needsUpgrade {
		if err := c.putUpgrade(data); err != nil {
			return err
		}
		c.needsUpgrade = false
		return nil
	}

	return c.RemoteClient.Put(data)
}

// move the state to the new location, put the datam and remove the old state
func (c *envUpgrader) putUpgrade(data []byte) error {
	// make a copy of the client with the new path
	newClient := &RemoteClient{
		s3Client:             c.RemoteClient.s3Client,
		dynClient:            c.RemoteClient.dynClient,
		bucketName:           c.RemoteClient.bucketName,
		path:                 c.backend.path(c.name),
		serverSideEncryption: c.RemoteClient.serverSideEncryption,
		acl:                  c.RemoteClient.acl,
		kmsKeyID:             c.RemoteClient.kmsKeyID,
		lockTable:            c.RemoteClient.lockTable,
	}

	// we lock the new state if we have a lock on the old
	if c.lockInfo != nil {
		// If we are using locks, we already have one since this happens in WriteState/Put
		lockInfo := state.NewLockInfo()
		lockInfo.Operation = c.lockInfo.Operation
		lockInfo.Info = "env path update"
		// we don't need the lock id, bc we store the LockInfo, and know our
		// impl uses that ID
		_, err := newClient.Lock(lockInfo)
		if err != nil {
			return fmt.Errorf("failed to lock s3 state: %s", err)
		}

		// replace the lock info for the next Unlock call
		c.lockInfo = lockInfo
	}

	// write the state to the new location
	if err := newClient.Put(data); err != nil {
		return err
	}

	// we've moved the state, now we can delete the old one
	err := c.backend.oldDeleteState(c.name)

	// if we're locking, make sure to remove the old lock
	if c.lockInfo != nil {
		if unlockErr := c.RemoteClient.Unlock(c.oldLockID); unlockErr != nil {
			err = multierror.Append(err, unlockErr)
			return fmt.Errorf("error removing old state during env upgrade: %s", err)
		}
	}

	// replace the client with the new path
	c.RemoteClient = newClient

	return err
}

func (c *envUpgrader) Lock(info *state.LockInfo) (string, error) {
	id, err := c.RemoteClient.Lock(info)
	if c.needsUpgrade {
		// store the lock ID for when the path gets updated
		c.oldLockID = id
		c.lockInfo = info
	}
	return id, err
}

func (c *envUpgrader) Unlock(id string) error {
	// If terraform had a lock during the upgrade, we need to translate the old
	// lock id to the new lock.
	if c.lockInfo != nil {
		if id != c.oldLockID {
			// The lock id doesn't match, so something went wrong.
			// Don't show the id to the user, since that id doesn't really
			// exist any longer, and the user may need the new LockInfo to
			// recover.
			return &state.LockError{
				Err:  errors.New("incorrect lock id"),
				Info: c.lockInfo,
			}
		}

		if err := c.RemoteClient.Unlock(c.lockInfo.ID); err != nil {
			return err
		}

		c.oldLockID = ""
		c.lockInfo = nil
		return nil
	}

	return c.RemoteClient.Unlock(id)
}
