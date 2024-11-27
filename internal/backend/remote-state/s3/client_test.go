// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package s3

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"maps"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(RemoteClient)
	var _ remote.ClientLocker = new(RemoteClient)
}

func TestRemoteClientBasic(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":  bucketName,
		"key":     keyName,
		"encrypt": true,
	})).(*Backend)

	createS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)
	defer deleteS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)

	state, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	remote.TestClient(t, state.(*remote.State).Client)
}

func TestRemoteClientLocks(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":         bucketName,
		"key":            keyName,
		"encrypt":        true,
		"dynamodb_table": bucketName,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":         bucketName,
		"key":            keyName,
		"encrypt":        true,
		"dynamodb_table": bucketName,
	})).(*Backend)

	createS3Bucket(ctx, t, b1.s3Client, bucketName, b1.awsConfig.Region)
	defer deleteS3Bucket(ctx, t, b1.s3Client, bucketName, b1.awsConfig.Region)
	createDynamoDBTable(ctx, t, b1.dynClient, bucketName)
	defer deleteDynamoDBTable(ctx, t, b1.dynClient, bucketName)

	s1, err := b1.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := b2.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	remote.TestRemoteLocks(t, s1.(*remote.State).Client, s2.(*remote.State).Client)
}

// verify that we can unlock a state with an existing lock
func TestForceUnlock(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-force-%x", time.Now().Unix())
	keyName := "testState"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":         bucketName,
		"key":            keyName,
		"encrypt":        true,
		"dynamodb_table": bucketName,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":         bucketName,
		"key":            keyName,
		"encrypt":        true,
		"dynamodb_table": bucketName,
	})).(*Backend)

	createS3Bucket(ctx, t, b1.s3Client, bucketName, b1.awsConfig.Region)
	defer deleteS3Bucket(ctx, t, b1.s3Client, bucketName, b1.awsConfig.Region)
	createDynamoDBTable(ctx, t, b1.dynClient, bucketName)
	defer deleteDynamoDBTable(ctx, t, b1.dynClient, bucketName)

	// first test with default
	s1, err := b1.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	info := statemgr.NewLockInfo()
	info.Operation = "test"
	info.Who = "clientA"

	lockID, err := s1.Lock(info)
	if err != nil {
		t.Fatal("unable to get initial lock:", err)
	}

	// s1 is now locked, get the same state through s2 and unlock it
	s2, err := b2.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal("failed to get default state to force unlock:", err)
	}

	if err := s2.Unlock(lockID); err != nil {
		t.Fatal("failed to force-unlock default state")
	}

	// now try the same thing with a named state
	// first test with default
	s1, err = b1.StateMgr("test")
	if err != nil {
		t.Fatal(err)
	}

	info = statemgr.NewLockInfo()
	info.Operation = "test"
	info.Who = "clientA"

	lockID, err = s1.Lock(info)
	if err != nil {
		t.Fatal("unable to get initial lock:", err)
	}

	// s1 is now locked, get the same state through s2 and unlock it
	s2, err = b2.StateMgr("test")
	if err != nil {
		t.Fatal("failed to get named state to force unlock:", err)
	}

	if err = s2.Unlock(lockID); err != nil {
		t.Fatal("failed to force-unlock named state")
	}
}

func TestForceUnlock_withLockfile(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-force-%x", time.Now().Unix())
	keyName := "testState"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":       bucketName,
		"key":          keyName,
		"encrypt":      true,
		"use_lockfile": true,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":       bucketName,
		"key":          keyName,
		"encrypt":      true,
		"use_lockfile": true,
	})).(*Backend)

	createS3Bucket(ctx, t, b1.s3Client, bucketName, b1.awsConfig.Region)
	defer deleteS3Bucket(ctx, t, b1.s3Client, bucketName, b1.awsConfig.Region)

	// first test with default
	s1, err := b1.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	info := statemgr.NewLockInfo()
	info.Operation = "test"
	info.Who = "clientA"

	lockID, err := s1.Lock(info)
	if err != nil {
		t.Fatal("unable to get initial lock:", err)
	}

	// s1 is now locked, get the same state through s2 and unlock it
	s2, err := b2.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal("failed to get default state to force unlock:", err)
	}

	if err := s2.Unlock(lockID); err != nil {
		t.Fatal("failed to force-unlock default state")
	}

	// now try the same thing with a named state
	// first test with default
	s1, err = b1.StateMgr("test")
	if err != nil {
		t.Fatal(err)
	}

	info = statemgr.NewLockInfo()
	info.Operation = "test"
	info.Who = "clientA"

	lockID, err = s1.Lock(info)
	if err != nil {
		t.Fatal("unable to get initial lock:", err)
	}

	// s1 is now locked, get the same state through s2 and unlock it
	s2, err = b2.StateMgr("test")
	if err != nil {
		t.Fatal("failed to get named state to force unlock:", err)
	}

	if err = s2.Unlock(lockID); err != nil {
		t.Fatal("failed to force-unlock named state")
	}
}

func TestRemoteClient_clientMD5(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":         bucketName,
		"key":            keyName,
		"dynamodb_table": bucketName,
	})).(*Backend)

	createS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)
	defer deleteS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)
	createDynamoDBTable(ctx, t, b.dynClient, bucketName)
	defer deleteDynamoDBTable(ctx, t, b.dynClient, bucketName)

	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}
	client := s.(*remote.State).Client.(*RemoteClient)

	sum := md5.Sum([]byte("test"))

	if err := client.putMD5(ctx, sum[:]); err != nil {
		t.Fatal(err)
	}

	getSum, err := client.getMD5(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(getSum, sum[:]) {
		t.Fatalf("getMD5 returned the wrong checksum: expected %x, got %x", sum[:], getSum)
	}

	if err := client.deleteMD5(ctx); err != nil {
		t.Fatal(err)
	}

	if getSum, err := client.getMD5(ctx); err == nil {
		t.Fatalf("expected getMD5 error, got none. checksum: %x", getSum)
	}
}

// verify that a client won't return a state with an incorrect checksum.
func TestRemoteClient_stateChecksum(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":         bucketName,
		"key":            keyName,
		"dynamodb_table": bucketName,
	})).(*Backend)

	createS3Bucket(ctx, t, b1.s3Client, bucketName, b1.awsConfig.Region)
	defer deleteS3Bucket(ctx, t, b1.s3Client, bucketName, b1.awsConfig.Region)
	createDynamoDBTable(ctx, t, b1.dynClient, bucketName)
	defer deleteDynamoDBTable(ctx, t, b1.dynClient, bucketName)

	s1, err := b1.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}
	client1 := s1.(*remote.State).Client

	// create an old and new state version to persist
	s := statemgr.TestFullInitialState()
	sf := &statefile.File{State: s}
	var oldState bytes.Buffer
	if err := statefile.Write(sf, &oldState); err != nil {
		t.Fatal(err)
	}
	sf.Serial++
	var newState bytes.Buffer
	if err := statefile.Write(sf, &newState); err != nil {
		t.Fatal(err)
	}

	// Use b2 without a dynamodb_table to bypass the lock table to write the state directly.
	// client2 will write the "incorrect" state, simulating s3 eventually consistency delays
	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket": bucketName,
		"key":    keyName,
	})).(*Backend)
	s2, err := b2.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}
	client2 := s2.(*remote.State).Client

	// write the new state through client2 so that there is no checksum yet
	if err := client2.Put(newState.Bytes()); err != nil {
		t.Fatal(err)
	}

	// verify that we can pull a state without a checksum
	if _, err := client1.Get(); err != nil {
		t.Fatal(err)
	}

	// write the new state back with its checksum
	if err := client1.Put(newState.Bytes()); err != nil {
		t.Fatal(err)
	}

	// put an empty state in place to check for panics during get
	if err := client2.Put([]byte{}); err != nil {
		t.Fatal(err)
	}

	// remove the timeouts so we can fail immediately
	origTimeout := consistencyRetryTimeout
	origInterval := consistencyRetryPollInterval
	defer func() {
		consistencyRetryTimeout = origTimeout
		consistencyRetryPollInterval = origInterval
	}()
	consistencyRetryTimeout = 0
	consistencyRetryPollInterval = 0

	// fetching an empty state through client1 should now error out due to a
	// mismatched checksum.
	if _, err := client1.Get(); !IsA[badChecksumError](err) {
		t.Fatalf("expected state checksum error: got %s", err)
	} else if bse, ok := As[badChecksumError](err); ok && len(bse.digest) != 0 {
		t.Fatalf("expected empty checksum, got %x", bse.digest)
	}

	// put the old state in place of the new, without updating the checksum
	if err := client2.Put(oldState.Bytes()); err != nil {
		t.Fatal(err)
	}

	// fetching the wrong state through client1 should now error out due to a
	// mismatched checksum.
	if _, err := client1.Get(); !IsA[badChecksumError](err) {
		t.Fatalf("expected state checksum error: got %s", err)
	}

	// update the state with the correct one after we Get again
	testChecksumHook = func() {
		if err := client2.Put(newState.Bytes()); err != nil {
			t.Fatal(err)
		}
		testChecksumHook = nil
	}

	consistencyRetryTimeout = origTimeout

	// this final Get will fail to fail the checksum verification, the above
	// callback will update the state with the correct version, and Get should
	// retry automatically.
	if _, err := client1.Get(); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteClientPutLargeUploadWithObjectLock_Compliance(t *testing.T) {
	testACC(t)
	objectLockPreCheck(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket": bucketName,
		"key":    keyName,
	})).(*Backend)

	createS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region,
		s3BucketWithVersioning,
		s3BucketWithObjectLock(s3types.ObjectLockRetentionModeCompliance),
	)
	defer deleteS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)

	s1, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}
	client := s1.(*remote.State).Client

	var state bytes.Buffer
	dataW := io.LimitReader(neverEnding('x'), manager.DefaultUploadPartSize*2)
	_, err = state.ReadFrom(dataW)
	if err != nil {
		t.Fatalf("writing dummy data: %s", err)
	}

	err = client.Put(state.Bytes())
	if err != nil {
		t.Fatalf("putting data: %s", err)
	}
}

func TestRemoteClientLockFileWithObjectLock_Compliance(t *testing.T) {
	testACC(t)
	objectLockPreCheck(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":       bucketName,
		"key":          keyName,
		"use_lockfile": true,
	})).(*Backend)

	createS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region,
		s3BucketWithVersioning,
		s3BucketWithObjectLock(s3types.ObjectLockRetentionModeCompliance),
	)
	defer deleteS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)

	s1, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}
	client := s1.(*remote.State).Client

	var state bytes.Buffer
	dataW := io.LimitReader(neverEnding('x'), manager.DefaultUploadPartSize)
	_, err = state.ReadFrom(dataW)
	if err != nil {
		t.Fatalf("writing dummy data: %s", err)
	}

	err = client.Put(state.Bytes())
	if err != nil {
		t.Fatalf("putting data: %s", err)
	}
}

func TestRemoteClientLockFileWithObjectLock_Governance(t *testing.T) {
	testACC(t)
	objectLockPreCheck(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":       bucketName,
		"key":          keyName,
		"use_lockfile": true,
	})).(*Backend)

	createS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region,
		s3BucketWithVersioning,
		s3BucketWithObjectLock(s3types.ObjectLockRetentionModeGovernance),
	)
	defer deleteS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)

	s1, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}
	client := s1.(*remote.State).Client

	var state bytes.Buffer
	dataW := io.LimitReader(neverEnding('x'), manager.DefaultUploadPartSize)
	_, err = state.ReadFrom(dataW)
	if err != nil {
		t.Fatalf("writing dummy data: %s", err)
	}

	err = client.Put(state.Bytes())
	if err != nil {
		t.Fatalf("putting data: %s", err)
	}
}

type neverEnding byte

func (b neverEnding) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = byte(b)
	}
	return len(p), nil
}

func TestRemoteClientSkipS3Checksum(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	testcases := map[string]struct {
		config   map[string]any
		expected string
	}{
		"default": {
			config:   map[string]any{},
			expected: string(s3types.ChecksumAlgorithmSha256),
		},
		"true": {
			config: map[string]any{
				"skip_s3_checksum": true,
			},
			expected: "",
		},
		"false": {
			config: map[string]any{
				"skip_s3_checksum": false,
			},
			expected: string(s3types.ChecksumAlgorithmSha256),
		},
	}

	for name, testcase := range testcases {
		t.Run(name, func(t *testing.T) {
			config := map[string]interface{}{
				"bucket": bucketName,
				"key":    keyName,
			}
			maps.Copy(config, testcase.config)
			b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config)).(*Backend)

			createS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)
			defer deleteS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)

			state, err := b.StateMgr(backend.DefaultStateName)
			if err != nil {
				t.Fatal(err)
			}

			c := state.(*remote.State).Client
			client := c.(*RemoteClient)

			s := statemgr.TestFullInitialState()
			sf := &statefile.File{State: s}
			var stateBuf bytes.Buffer
			if err := statefile.Write(sf, &stateBuf); err != nil {
				t.Fatal(err)
			}

			var checksum string
			err = client.put(stateBuf.Bytes(), func(opts *s3.Options) {
				opts.APIOptions = append(opts.APIOptions,
					addRetrieveChecksumHeaderMiddleware(t, &checksum),
					addCancelRequestMiddleware(),
				)
			})
			if err == nil {
				t.Fatal("Expected an error, got none")
			} else if !errors.Is(err, errCancelOperation) {
				t.Fatalf("Unexpected error: %s", err)
			}

			if a, e := checksum, testcase.expected; a != e {
				t.Fatalf("expected %q, got %q", e, a)
			}
		})
	}
}

func addRetrieveChecksumHeaderMiddleware(t *testing.T, checksum *string) func(*middleware.Stack) error {
	return func(stack *middleware.Stack) error {
		return stack.Finalize.Add(
			retrieveChecksumHeaderMiddleware(t, checksum),
			middleware.After,
		)
	}
}

func retrieveChecksumHeaderMiddleware(t *testing.T, checksum *string) middleware.FinalizeMiddleware {
	return middleware.FinalizeMiddlewareFunc(
		"Test: Retrieve Stuff",
		func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			t.Helper()

			request, ok := in.Request.(*smithyhttp.Request)
			if !ok {
				t.Fatalf("Expected *github.com/aws/smithy-go/transport/http.Request, got %s", fullTypeName(in.Request))
			}

			*checksum = request.Header.Get("x-amz-sdk-checksum-algorithm")

			return next.HandleFinalize(ctx, in)
		})
}
