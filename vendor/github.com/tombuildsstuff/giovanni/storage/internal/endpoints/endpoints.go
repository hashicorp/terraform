package endpoints

import (
	"fmt"
	"strings"
)

func GetAccountNameFromEndpoint(endpoint string) (*string, error) {
	segments := strings.Split(endpoint, ".")
	if len(segments) == 0 {
		return nil, fmt.Errorf("The Endpoint contained no segments")
	}
	return &segments[0], nil
}

// GetBlobEndpoint returns the endpoint for Blob API Operations on this storage account
func GetBlobEndpoint(baseUri string, accountName string) string {
	return fmt.Sprintf("https://%s.blob.%s", accountName, baseUri)
}

// GetDataLakeStoreEndpoint returns the endpoint for Data Lake Store API Operations on this storage account
func GetDataLakeStoreEndpoint(baseUri string, accountName string) string {
	return fmt.Sprintf("https://%s.dfs.%s", accountName, baseUri)
}

// GetFileEndpoint returns the endpoint for File Share API Operations on this storage account
func GetFileEndpoint(baseUri string, accountName string) string {
	return fmt.Sprintf("https://%s.file.%s", accountName, baseUri)
}

// GetQueueEndpoint returns the endpoint for Queue API Operations on this storage account
func GetQueueEndpoint(baseUri string, accountName string) string {
	return fmt.Sprintf("https://%s.queue.%s", accountName, baseUri)
}

// GetTableEndpoint returns the endpoint for Table API Operations on this storage account
func GetTableEndpoint(baseUri string, accountName string) string {
	return fmt.Sprintf("https://%s.table.%s", accountName, baseUri)
}
