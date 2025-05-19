// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	"math/rand"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(RemoteClient)
	var _ remote.ClientLocker = new(RemoteClient)
}

func TestRemoteClient(t *testing.T) {
	testACC(t)
	defer cleanupK8sResources(t)

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"secret_suffix": secretSuffix,
	}))

	state, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	remote.TestClient(t, state.(*remote.State).Client)
}

func TestRemoteClientLocks(t *testing.T) {
	testACC(t)
	defer cleanupK8sResources(t)

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"secret_suffix": secretSuffix,
	}))

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"secret_suffix": secretSuffix,
	}))

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

func TestLargeState(t *testing.T) {
	testACC(t)
	defer cleanupK8sResources(t)

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"secret_suffix": secretSuffix,
	}))

	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	// generate a very large state
	largeState := generateLargeState(20)
	err = s.WriteState(largeState)
	if err != nil {
		t.Fatal(err)
	}
	err = s.PersistState(nil)
	if err != nil {
		t.Fatal(err)
	}
	err = s.RefreshState()
	if err != nil {
		t.Fatal(err)
	}

	// shrink it down
	largeState = generateLargeState(3)
	err = s.WriteState(largeState)
	if err != nil {
		t.Fatal(err)
	}
	err = s.PersistState(nil)
	if err != nil {
		t.Fatal(err)
	}
	err = s.RefreshState()
	if err != nil {
		t.Fatal(err)
	}
}

func generateLargeState(size int) *states.State {
	bigState := states.NewState()
	dataSize := defaultChunkSize * size
	chars := "abcdefghijklmnopqrstuvwxyz"
	testData := make([]byte, dataSize)
	for i := 0; i < dataSize; i++ {
		testData[i] = chars[rand.Intn(len(chars))]
	}
	bigState.SyncWrapper().SetResourceInstanceCurrent(
		addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "large_resource",
				Name: "foo",
			},
		}.Absolute(addrs.RootModuleInstance),
		&states.ResourceInstanceObjectSrc{
			AttrsJSON:     []byte(`{"test_data":"` + string(testData) + `"}`),
			Status:        states.ObjectReady,
			SchemaVersion: 0,
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)
	return bigState
}

func TestForceUnlock(t *testing.T) {
	testACC(t)
	defer cleanupK8sResources(t)

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"secret_suffix": secretSuffix,
	}))

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"secret_suffix": secretSuffix,
	}))

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
