package azure

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/storageservice"
	"github.com/Azure/azure-sdk-for-go/storage"
)

// getStorageServiceBlobClient is a helper function which returns the
// storage.BlobStorageClient associated to the given storage account name.
func getStorageServiceBlobClient(mgmtClient management.Client, serviceName string) (storage.BlobStorageClient, error) {
	log.Println("[INFO] Begun generating Azure Storage Service Blob client.")
	var blobClient storage.BlobStorageClient

	storageServiceClient := storageservice.NewClient(mgmtClient)

	keys, err := storageServiceClient.GetStorageServiceKeys(serviceName)
	if err != nil {
		return blobClient, fmt.Errorf("Error reading Storage Service %s's keys from Azure: %s", serviceName, err)
	}

	storageClient, err := storage.NewBasicClient(serviceName, keys.PrimaryKey)
	if err != nil {
		return blobClient, fmt.Errorf("Error creating Storage Service Client for %s: %s", serviceName, err)
	}

	return storageClient.GetBlobService(), nil
}
