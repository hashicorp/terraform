package azure

import (
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(RemoteClient)
	var _ remote.ClientLocker = new(RemoteClient)
}

func TestRemoteClient(t *testing.T) {
	testACC(t)

	keyName := "testState"
	res := setupResources(t, keyName)
	defer destroyResources(t, res.resourceGroupName)

	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"storage_account_name": res.storageAccountName,
		"container_name":       res.containerName,
		"key":                  keyName,
		"access_key":           res.accessKey,
	}).(*Backend)

	state, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	remote.TestClient(t, state.(*remote.State).Client)
}

func TestRemoteClientLocks(t *testing.T) {
	testACC(t)

	keyName := "testState"
	res := setupResources(t, keyName)
	defer destroyResources(t, res.resourceGroupName)

	b1 := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"storage_account_name": res.storageAccountName,
		"container_name":       res.containerName,
		"key":                  keyName,
		"access_key":           res.accessKey,
	}).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"storage_account_name": res.storageAccountName,
		"container_name":       res.containerName,
		"key":                  keyName,
		"access_key":           res.accessKey,
	}).(*Backend)

	s1, err := b1.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := b2.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	remote.TestRemoteLocks(t, s1.(*remote.State).Client, s2.(*remote.State).Client)
}
