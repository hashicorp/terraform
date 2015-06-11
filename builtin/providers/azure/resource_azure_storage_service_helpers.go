package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/storageservice"
	"github.com/Azure/azure-sdk-for-go/storage"
)

// getStorageClientForStorageService is helper function which returns the
// storage.Client associated to the given storage service name.
func getStorageClientForStorageService(mgmtClient management.Client, serviceName string) (storage.Client, error) {
	var storageClient storage.Client
	storageServiceClient := storageservice.NewClient(mgmtClient)

	keys, err := storageServiceClient.GetStorageServiceKeys(serviceName)
	if err != nil {
		return storageClient, fmt.Errorf("Failed getting Storage Service keys for %s: %s", serviceName, err)
	}

	storageClient, err = storage.NewBasicClient(serviceName, keys.PrimaryKey)
	if err != nil {
		return storageClient, fmt.Errorf("Failed creating Storage Service client for %s: %s", serviceName, err)
	}

	return storageClient, err
}

// getStorageServiceBlobClient is a helper function which returns the
// storage.BlobStorageClient associated to the given storage service name.
func getStorageServiceBlobClient(mgmtClient management.Client, serviceName string) (storage.BlobStorageClient, error) {
	storageClient, err := getStorageClientForStorageService(mgmtClient, serviceName)
	if err != nil {
		return storage.BlobStorageClient{}, err
	}

	return storageClient.GetBlobService(), nil
}

// getStorageServiceQueueClient is a helper function which returns the
// storage.QueueServiceClient associated to the given storage service name.
func getStorageServiceQueueClient(mgmtClient management.Client, serviceName string) (storage.QueueServiceClient, error) {
	storageClient, err := getStorageClientForStorageService(mgmtClient, serviceName)
	if err != nil {
		return storage.QueueServiceClient{}, err
	}

	return storageClient.GetQueueService(), err
}
