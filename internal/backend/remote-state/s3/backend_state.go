// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package s3

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	baselogging "github.com/hashicorp/aws-sdk-go-base/v2/logging"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

const (
	// defaultWorkspaceKeyPrefix is the default prefix for workspace storage.
	// The colon is used to reduce the chance of name conflicts with existing objects.
	defaultWorkspaceKeyPrefix = "env:"
	// lockFileSuffix defines the suffix for Terraform state lock files.
	lockFileSuffix = ".tflock"
)

func (b *Backend) Workspaces() ([]string, error) {
	const maxKeys = 1000

	ctx := context.TODO()
	log := logger()
	log = logWithOperation(log, operationBackendWorkspaces)
	log = log.With(
		logKeyBucket, b.bucketName,
	)

	prefix := ""

	if b.workspaceKeyPrefix != "" {
		prefix = b.workspaceKeyPrefix + "/"
	}

	log = log.With(
		logKeyBackendWorkspacePrefix, prefix,
	)

	params := &s3.ListObjectsV2Input{
		Bucket:  aws.String(b.bucketName),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(maxKeys),
	}

	wss := []string{backend.DefaultStateName}

	ctx, baselog := baselogging.NewHcLogger(ctx, log)
	ctx = baselogging.RegisterLogger(ctx, baselog)

	pages := s3.NewListObjectsV2Paginator(b.s3Client, params)
	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)
		if err != nil {
			if IsA[*s3types.NoSuchBucket](err) {
				return nil, fmt.Errorf(errS3NoSuchBucket, b.bucketName, err)
			}
			if foo, ok := As[smithy.APIError](err); b.workspaceKeyPrefix == defaultWorkspaceKeyPrefix && ok && foo.ErrorCode() == "AccessDenied" {
				log.Warn("Unable to list non-default workspaces", "err", err.Error())
				return wss[:1], nil
			}
			return nil, fmt.Errorf("Unable to list objects in S3 bucket %q with prefix %q: %w", b.bucketName, prefix, err)
		}

		for _, obj := range page.Contents {
			ws := b.keyEnv(aws.ToString(obj.Key))
			if ws != "" {
				wss = append(wss, ws)
			}
		}
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

func (b *Backend) DeleteWorkspace(name string, _ bool) error {
	log := logger()
	log = logWithOperation(log, operationBackendDeleteWorkspace)
	log = log.With(
		logKeyBackendWorkspace, name,
	)

	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	log.Info("Deleting workspace")

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
		skipS3Checksum:        b.skipS3Checksum,
		lockFilePath:          b.getLockFilePath(name),
		useLockFile:           b.useLockFile,
	}

	return client, nil
}

func (b *Backend) StateMgr(name string) (statemgr.Full, error) {
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
		lockInfo := statemgr.NewLockInfo()
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
			if err := stateMgr.PersistState(nil); err != nil {
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

var _ error = bucketRegionError{}

type bucketRegionError struct {
	requestRegion, bucketRegion string
}

func newBucketRegionError(requestRegion, bucketRegion string) bucketRegionError {
	return bucketRegionError{
		requestRegion: requestRegion,
		bucketRegion:  bucketRegion,
	}
}

func (err bucketRegionError) Error() string {
	return fmt.Sprintf("requested bucket from %q, actual location %q", err.requestRegion, err.bucketRegion)
}

// getLockFilePath returns the path to the lock file for the given Terraform state.
// For `default.tfstate`, the lock file is stored at `default.tfstate.tflock`.
func (b *Backend) getLockFilePath(name string) string {
	return b.path(name) + lockFileSuffix
}
