package azure

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/acctest"
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

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"storage_account_name": res.storageAccountName,
		"container_name":       res.containerName,
		"key":                  keyName,
		"access_key":           res.accessKey,
	})).(*Backend)

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

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"storage_account_name": res.storageAccountName,
		"container_name":       res.containerName,
		"key":                  keyName,
		"access_key":           res.accessKey,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"storage_account_name": res.storageAccountName,
		"container_name":       res.containerName,
		"key":                  keyName,
		"access_key":           res.accessKey,
	})).(*Backend)

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

func TestPutMaintainsMetaData(t *testing.T) {
	testACC(t)

	keyName := "testState"
	headerName := "acceptancetest"
	expectedValue := "f3b56bad-33ad-4b93-a600-7a66e9cbd1eb"
	res := setupResources(t, keyName)
	defer destroyResources(t, res.resourceGroupName)

	config := getBackendConfig(t, res)
	blobClient, err := getBlobClient(config)
	if err != nil {
		t.Fatalf("Error getting Blob Client: %+v", err)
	}

	containerReference := blobClient.GetContainerReference(res.containerName)
	blobReference := containerReference.GetBlobReference(keyName)

	err = blobReference.CreateBlockBlob(&storage.PutBlobOptions{})
	if err != nil {
		t.Fatalf("Error Creating Block Blob: %+v", err)
	}

	err = blobReference.GetMetadata(&storage.GetBlobMetadataOptions{})
	if err != nil {
		t.Fatalf("Error loading MetaData: %+v", err)
	}

	blobReference.Metadata[headerName] = expectedValue
	err = blobReference.SetMetadata(&storage.SetBlobMetadataOptions{})
	if err != nil {
		t.Fatalf("Error setting MetaData: %+v", err)
	}

	// update the metadata using the Backend
	remoteClient := RemoteClient{
		keyName:       res.keyName,
		containerName: res.containerName,
		blobClient:    blobClient,
	}

	bytes := []byte(acctest.RandString(20))
	err = remoteClient.Put(bytes)
	if err != nil {
		t.Fatalf("Error putting data: %+v", err)
	}

	// Verify it still exists
	err = blobReference.GetMetadata(&storage.GetBlobMetadataOptions{})
	if err != nil {
		t.Fatalf("Error loading MetaData: %+v", err)
	}

	if blobReference.Metadata[headerName] != expectedValue {
		t.Fatalf("%q was not set to %q in the MetaData: %+v", headerName, expectedValue, blobReference.Metadata)
	}
}

func getBackendConfig(t *testing.T, res testResources) BackendConfig {
	clients := getTestClient(t)
	return BackendConfig{
		ClientID:       clients.clientID,
		ClientSecret:   clients.clientSecret,
		Environment:    clients.environment.Name,
		SubscriptionID: clients.subscriptionID,
		TenantID:       clients.tenantID,

		ResourceGroupName:  res.resourceGroupName,
		StorageAccountName: res.storageAccountName,
	}
}
