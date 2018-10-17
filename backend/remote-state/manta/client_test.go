package manta

import (
	"testing"

	"fmt"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(RemoteClient)
	var _ remote.ClientLocker = new(RemoteClient)
}

func TestRemoteClient(t *testing.T) {
	testACC(t)
	directory := fmt.Sprintf("terraform-remote-manta-test-%x", time.Now().Unix())
	keyName := "testState"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"path":       directory,
		"objectName": keyName,
	})).(*Backend)

	createMantaFolder(t, b.storageClient, directory)
	defer deleteMantaFolder(t, b.storageClient, directory)

	state, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	remote.TestClient(t, state.(*remote.State).Client)
}

func TestRemoteClientLocks(t *testing.T) {
	testACC(t)
	directory := fmt.Sprintf("terraform-remote-manta-test-%x", time.Now().Unix())
	keyName := "testState"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"path":       directory,
		"objectName": keyName,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"path":       directory,
		"objectName": keyName,
	})).(*Backend)

	createMantaFolder(t, b1.storageClient, directory)
	defer deleteMantaFolder(t, b1.storageClient, directory)

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
