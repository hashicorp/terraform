package s3

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

// run through a multi-step upgrade in a single long script
func TestOldEnvUpgrade(t *testing.T) {
	testACC(t)
	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"bucket":     bucketName,
		"key":        keyName,
		"encrypt":    true,
		"lock_table": bucketName,
	}).(*Backend)

	createS3Bucket(t, b.s3Client, bucketName)
	defer deleteS3Bucket(t, b.s3Client, bucketName)
	createDynamoDBTable(t, b.dynClient, bucketName)
	defer deleteDynamoDBTable(t, b.dynClient, bucketName)

	// put multiple states in old env paths.
	s1 := terraform.NewState()
	s1.Modules[0] = &terraform.ModuleState{
		Path: []string{"root"},
		Resources: map[string]*terraform.ResourceState{
			"aws_instance.a": &terraform.ResourceState{
				Type: "aws_instance",
				Primary: &terraform.InstanceState{
					ID: "bar",
				},
			},
		},
	}

	s2 := terraform.NewState()
	s2.Modules[0] = &terraform.ModuleState{
		Path: []string{"root"},
		Resources: map[string]*terraform.ResourceState{
			"aws_instance.b": &terraform.ResourceState{
				Type: "aws_instance",
				Primary: &terraform.InstanceState{
					ID: "baz",
				},
			},
		},
	}

	// RemoteClient to Put things in the old paths
	client := &RemoteClient{
		s3Client:             b.s3Client,
		dynClient:            b.dynClient,
		bucketName:           b.bucketName,
		path:                 b.oldPath("s1"),
		serverSideEncryption: b.serverSideEncryption,
		acl:                  b.acl,
		kmsKeyID:             b.kmsKeyID,
		lockTable:            b.lockTable,
	}

	stateMgr := &remote.State{Client: client}
	stateMgr.WriteState(s1)
	if err := stateMgr.PersistState(); err != nil {
		t.Fatal(err)
	}

	client.path = b.oldPath("s2")
	stateMgr.WriteState(s2)
	if err := stateMgr.PersistState(); err != nil {
		t.Fatal(err)
	}

	if err := checkStateList(b, []string{"default", "s1", "s2"}); err != nil {
		t.Fatal(err)
	}

	// add a new state
	s3Mgr, err := b.State("s3")
	if err != nil {
		t.Fatal(err)
	}
	// make sure we didn't get an upgrader
	_, ok := s3Mgr.(*remote.State).Client.(*envUpgrader)
	if ok {
		t.Fatal("s3 should not be upgraded")
	}
	if err := checkStateList(b, []string{"default", "s1", "s2", "s3"}); err != nil {
		t.Fatal(err)
	}

	testUpgradeAndCompare(t, "s1", b, s1)
	if err := checkStateList(b, []string{"default", "s1", "s2", "s3"}); err != nil {
		t.Fatal(err)
	}

	testUpgradeAndCompare(t, "s2", b, s2)
	if err := checkStateList(b, []string{"default", "s1", "s2", "s3"}); err != nil {
		t.Fatal(err)
	}

	// Check that the old state paths don't exist.
	// Refreshing swallows the error if the key doesn't exist, so just check
	// for empty state.
	client.path = b.oldPath("s1")
	stateMgr = &remote.State{Client: client}
	if err := stateMgr.RefreshState(); err != nil {
		t.Fatal("error refreshing old s1 state path:", err)
	}
	if stateMgr.State() != nil {
		t.Fatal("expected empty state at old s1 path, got", stateMgr.State())
	}

	client.path = b.oldPath("s2")
	stateMgr = &remote.State{Client: client}
	if err := stateMgr.RefreshState(); err != nil {
		t.Fatal("error refreshing old s2 state path:", err)
	}
	if stateMgr.State() != nil {
		t.Fatal("expected empty state at old s2 path, got", stateMgr.State())
	}
}

// Pre-load an old named state and test that it can be correctly locked and
// unlocked around the upgrade.
func TestOldEnvUpgradeLocked(t *testing.T) {
	testACC(t)
	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b1 := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"bucket":     bucketName,
		"key":        keyName,
		"encrypt":    true,
		"lock_table": bucketName,
	}).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"bucket":     bucketName,
		"key":        keyName,
		"encrypt":    true,
		"lock_table": bucketName,
	}).(*Backend)

	createS3Bucket(t, b1.s3Client, bucketName)
	defer deleteS3Bucket(t, b1.s3Client, bucketName)
	createDynamoDBTable(t, b1.dynClient, bucketName)
	defer deleteDynamoDBTable(t, b1.dynClient, bucketName)

	// create a legacy named state
	s1 := terraform.NewState()

	// RemoteClient to Put things in the old paths
	client := &RemoteClient{
		s3Client:             b1.s3Client,
		dynClient:            b1.dynClient,
		bucketName:           b1.bucketName,
		path:                 b1.oldPath("s1"),
		serverSideEncryption: b1.serverSideEncryption,
		acl:                  b1.acl,
		kmsKeyID:             b1.kmsKeyID,
		lockTable:            b1.lockTable,
	}

	stateMgr := &remote.State{Client: client}
	stateMgr.WriteState(s1)
	if err := stateMgr.PersistState(); err != nil {
		t.Fatal(err)
	}

	s1Mgr, err := b1.State("s1")
	if err != nil {
		t.Fatal(err)
	}

	s1LockInfo := state.NewLockInfo()
	s1LockInfo.Operation = "test"
	s1LockInfo.Who = "s1"
	s1LockID, err := s1Mgr.Lock(s1LockInfo)
	if err != nil {
		t.Fatal(err)
	}

	if s1LockID == "" {
		t.Fatal("s1 not locked")
	}

	{
		// check that we can't get a lock on s1 from b2
		s1Mgr, err := b2.State("s1")
		if err != nil {
			t.Fatal(err)
		}

		s1LockInfo := state.NewLockInfo()
		s1LockInfo.Operation = "test-fail"
		s1LockInfo.Who = "b2"
		_, err = s1Mgr.Lock(s1LockInfo)
		if err == nil {
			t.Fatal("expected error getting second lock on s1")
		}
	}

	// upgrade the state on write
	s1.Serial++
	s1Mgr.WriteState(s1)
	if err := s1Mgr.PersistState(); err != nil {
		t.Fatal("error persisting s1", err)
	}

	// check that we have a different lockInfo now stored from the upgrade
	newLockInfo := s1Mgr.(*remote.State).Client.(*envUpgrader).lockInfo
	if newLockInfo.ID == s1LockID {
		t.Fatal("upgraded state should have a new lock ID")
	}

	// make sure we still can't get a second lock
	{
		// this time it will fail when trying fetch the new state because we
		// may need to create one
		_, err = b2.State("s1")
		if err == nil {
			t.Fatal("expected lock failure fetching s1")
		}
	}

	if err := s1Mgr.Unlock(newLockInfo.ID); err == nil {
		t.Fatal("unlocking with the new lock should have failed")
	}

	if err := s1Mgr.Unlock(s1LockID); err != nil {
		t.Fatal("unlock failed:", err)
	}

	// now we should be able to get a second lock
	{
		s1Mgr, err := b2.State("s1")
		if err != nil {
			t.Fatal(err)
		}

		s1LockInfo := state.NewLockInfo()
		s1LockInfo.Operation = "test-fail"
		s1LockInfo.Who = "b2"
		id, err := s1Mgr.Lock(s1LockInfo)
		if err != nil {
			t.Fatal("expected error getting second lock on s1")
		}
		defer func() {
			if err := s1Mgr.Unlock(id); err != nil {
				t.Fatal(err)
			}
		}()
	}
}

func checkStateList(b backend.Backend, expected []string) error {
	states, err := b.States()
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(states, expected) {
		return fmt.Errorf("incorrect states listed: %q", states)
	}
	return nil
}

func testUpgradeAndCompare(t *testing.T, name string, b backend.Backend, s *terraform.State) {
	sMgr, err := b.State(name)
	if err != nil {
		t.Fatal("error fetching", name, err)
	}

	// make sure we got our envUpgrader
	upgrader, ok := sMgr.(*remote.State).Client.(*envUpgrader)
	if !ok {
		t.Fatalf("expected envUpgrader for %q, got %T", name, sMgr.(*remote.State).Client)
	}

	if err := sMgr.RefreshState(); err != nil {
		t.Fatal(err)
	}

	// upgrade the state on write
	s.Serial++
	sMgr.WriteState(s)
	if err := sMgr.PersistState(); err != nil {
		t.Fatal("error persisting", name, err)
	}

	// the upgrader should be done
	if upgrader.needsUpgrade {
		t.Fatalf("%q not marked as upgraded", name)
	}

	if upgrader.RemoteClient.path != b.(*Backend).path(name) {
		t.Fatalf("incorrect path for %q: %s", name, upgrader.RemoteClient.path)
	}

	// fetch the state again, and compare
	sMgr, err = b.State(name)
	if err != nil {
		t.Fatal("error fetching", name, err)
	}

	if err := sMgr.RefreshState(); err != nil {
		t.Fatal("error refreshing", name, err)
	}

	upgraded := sMgr.State()
	if !(upgraded.Equal(s) && upgraded.Lineage == s.Lineage) {
		t.Fatalf("incorrectly upgraded %q: got: %s\nexpected: %s", name, upgraded, s)
	}
}
