package s3

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/states"
)

func (b *Backend) Workspaces() ([]string, error) {
	const maxKeys = 1000

	prefix := ""

	if b.workspaceKeyPrefix != "" {
		prefix = b.workspaceKeyPrefix + "/"
	}

	params := &s3.ListObjectsInput{
		Bucket:  &b.bucketName,
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int64(maxKeys),
	}

	wss := []string{backend.DefaultStateName}
	err := b.s3Client.ListObjectsPages(params, func(page *s3.ListObjectsOutput, lastPage bool) bool {
		for _, obj := range page.Contents {
			ws := b.keyEnv(*obj.Key)
			if ws != "" {
				wss = append(wss, ws)
			}
		}
		return !lastPage
	})

	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == s3.ErrCodeNoSuchBucket {
		return nil, fmt.Errorf(errS3NoSuchBucket, err)
	}

	sort.Strings(wss[1:])
	return wss, nil
}

func (b *Backend) keyEnv(key string) string {
	prefix := b.workspaceKeyPrefix

	if prefix == "" {
		parts := strings.SplitN(key, "/", 2)
		if len(parts) > 1 && parts[1] == b.keyName {
			return parts[0]
		} else {
			return ""
		}
	}

	// add a slash to treat this as a directory
	prefix += "/"

	parts := strings.SplitAfterN(key, prefix, 2)
	if len(parts) < 2 {
		return ""
	}

	// shouldn't happen since we listed by prefix
	if parts[0] != prefix {
		return ""
	}

	parts = strings.SplitN(parts[1], "/", 2)

	if len(parts) < 2 {
		return ""
	}

	// not our key, so don't include it in our listing
	if parts[1] != b.keyName {
		return ""
	}

	return parts[0]
}

func (b *Backend) DeleteWorkspace(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	client, err := b.remoteClient(name)
	if err != nil {
		return err
	}

	return client.Delete()
}

// get a remote client configured for this state
func (b *Backend) remoteClient(name string) (*RemoteClient, error) {
	if name == "" {
		return nil, errors.New("missing state name")
	}

	client := &RemoteClient{
		s3Client:              b.s3Client,
		dynClient:             b.dynClient,
		bucketName:            b.bucketName,
		path:                  b.path(name),
		serverSideEncryption:  b.serverSideEncryption,
		customerEncryptionKey: b.customerEncryptionKey,
		acl:                   b.acl,
		kmsKeyID:              b.kmsKeyID,
		ddbTable:              b.ddbTable,
	}

	return client, nil
}

func (b *Backend) StateMgr(name string) (state.State, error) {
	client, err := b.remoteClient(name)
	if err != nil {
		return nil, err
	}

	stateMgr := &remote.State{Client: client}
	// Check to see if this state already exists.
	// If we're trying to force-unlock a state, we can't take the lock before
	// fetching the state. If the state doesn't exist, we have to assume this
	// is a normal create operation, and take the lock at that point.
	//
	// If we need to force-unlock, but for some reason the state no longer
	// exists, the user will have to use aws tools to manually fix the
	// situation.
	existing, err := b.Workspaces()
	if err != nil {
		return nil, err
	}

	exists := false
	for _, s := range existing {
		if s == name {
			exists = true
			break
		}
	}

	// We need to create the object so it's listed by States.
	if !exists {
		// take a lock on this state while we write it
		lockInfo := state.NewLockInfo()
		lockInfo.Operation = "init"
		lockId, err := client.Lock(lockInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to lock s3 state: %s", err)
		}

		// Local helper function so we can call it multiple places
		lockUnlock := func(parent error) error {
			if err := stateMgr.Unlock(lockId); err != nil {
				return fmt.Errorf(strings.TrimSpace(errStateUnlock), lockId, err)
			}
			return parent
		}

		// Grab the value
		// This is to ensure that no one beat us to writing a state between
		// the `exists` check and taking the lock.
		if err := stateMgr.RefreshState(); err != nil {
			err = lockUnlock(err)
			return nil, err
		}

		// If we have no state, we have to create an empty state
		if v := stateMgr.State(); v == nil {
			if err := stateMgr.WriteState(states.NewState()); err != nil {
				err = lockUnlock(err)
				return nil, err
			}
			if err := stateMgr.PersistState(); err != nil {
				err = lockUnlock(err)
				return nil, err
			}
		}

		// Unlock, the state should now be initialized
		if err := lockUnlock(nil); err != nil {
			return nil, err
		}

	}

	return stateMgr, nil
}

func (b *Backend) client() *RemoteClient {
	return &RemoteClient{}
}

func (b *Backend) path(name string) string {
	if name == backend.DefaultStateName {
		return b.keyName
	}

	return path.Join(b.workspaceKeyPrefix, name, b.keyName)
}

const errStateUnlock = `
Error unlocking S3 state. Lock ID: %s

Error: %s

You may have to force-unlock this state in order to use it again.
`
