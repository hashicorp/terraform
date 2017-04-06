package s3

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

const (
	keyEnvPrefix = "-env:"
)

func (b *Backend) States() ([]string, error) {
	// fetch deprecated envs
	old, err := b.oldStates()
	if err != nil {
		return nil, err
	}

	prefix := b.keyName + keyEnvPrefix
	params := &s3.ListObjectsInput{
		Bucket: &b.bucketName,
		Prefix: aws.String(prefix),
	}

	resp, err := b.s3Client.ListObjects(params)
	if err != nil {
		return nil, err
	}

	envs := map[string]struct{}{}
	for _, obj := range resp.Contents {
		key := *obj.Key
		if strings.HasPrefix(key, prefix) {
			name := strings.TrimPrefix(key, prefix)
			// we store the state in a key, not a directory
			if strings.Contains(name, "/") {
				continue
			}

			envs[name] = struct{}{}
		}
	}

	result := []string{backend.DefaultStateName}
	for name := range envs {
		result = append(result, name)
	}

	// add old envs
	for _, name := range old {
		result = append(result, name)
	}

	sort.Strings(result[1:])
	return result, nil
}

func (b *Backend) DeleteState(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	params := &s3.DeleteObjectInput{
		Bucket: &b.bucketName,
		Key:    aws.String(b.path(name)),
	}

	_, err := b.s3Client.DeleteObject(params)
	if err != nil {
		return err
	}

	return nil
}

func (b *Backend) State(name string) (state.State, error) {
	oldStates, err := b.oldStates()
	if err != nil {
		return nil, err
	}

	for _, n := range oldStates {
		if n == name {
			return b.oldState(name)
		}
	}
	return b.state(name)
}

// TODO: recombine State after deprecated env code is removed
func (b *Backend) state(name string) (state.State, error) {

	client := &RemoteClient{
		s3Client:             b.s3Client,
		dynClient:            b.dynClient,
		bucketName:           b.bucketName,
		path:                 b.path(name),
		serverSideEncryption: b.serverSideEncryption,
		acl:                  b.acl,
		kmsKeyID:             b.kmsKeyID,
		lockTable:            b.lockTable,
	}

	stateMgr := &remote.State{Client: client}

	//if this isn't the default state name, we need to create the object so
	//it's listed by States.
	if name != backend.DefaultStateName {
		// take a lock on this state while we write it
		lockInfo := state.NewLockInfo()
		lockInfo.Operation = "init"
		lockID, err := client.Lock(lockInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to lock s3 state: %s", err)
		}

		unlock := lockUnlock(stateMgr, lockID)

		// Grab the value
		if err := stateMgr.RefreshState(); err != nil {
			return nil, unlock(err)
		}

		// If we have no state, we have to create an empty state
		if v := stateMgr.State(); v == nil {
			if err := stateMgr.WriteState(terraform.NewState()); err != nil {
				return nil, unlock(err)
			}
			if err := stateMgr.PersistState(); err != nil {
				return nil, unlock(err)
			}
		}

		// Unlock, the state should now be initialized
		if err := unlock(nil); err != nil {
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

	return b.keyName + keyEnvPrefix + name
}

// helper function so we can call it multiple places and combine errors
func lockUnlock(stateMgr state.State, id string) func(error) error {
	return func(parent error) error {
		if err := stateMgr.Unlock(id); err != nil {
			return fmt.Errorf(strings.TrimSpace(errStateUnlock), id, err)
		}
		return parent
	}
}

const errStateUnlock = `
Error unlocking S3 state. Lock ID: %s

Error: %s

You may have to force-unlock this state in order to use it again.
`
