## Blob Storage Blobs SDK for API version 2018-11-09

This package allows you to interact with the Blobs Blob Storage API

### Supported Authorizers

* Azure Active Directory (for the Resource Endpoint `https://storage.azure.com`)
* SharedKeyLite (Blob, File & Queue)

### Example Usage

```go
package main

import (
	"context"
	"fmt"
	"time"
	
	"github.com/Azure/go-autorest/autorest"
	"github.com/tombuildsstuff/giovanni/storage/2018-11-09/blob/blobs"
)

func Example() error {
	accountName := "storageaccount1"
    storageAccountKey := "ABC123...."
    containerName := "mycontainer"
    fileName := "example-large-file.iso"
    
    storageAuth := autorest.NewSharedKeyLiteAuthorizer(accountName, storageAccountKey)
    blobClient := blobs.New()
    blobClient.Client.Authorizer = storageAuth
    
    ctx := context.TODO()
    copyInput := blobs.CopyInput{
        CopySource: "http://releases.ubuntu.com/18.04.2/ubuntu-18.04.2-desktop-amd64.iso",
    }
    refreshInterval := 5 * time.Second
    if err := blobClient.CopyAndWait(ctx, accountName, containerName, fileName, copyInput, refreshInterval); err != nil {
        return fmt.Errorf("Error copying: %s", err)
    }
    
    return nil 
}

```