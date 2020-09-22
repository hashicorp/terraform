package blobs

import (
	"context"
	"fmt"
	"time"
)

// CopyAndWait copies a blob to a destination within the storage account and waits for it to finish copying.
func (client Client) CopyAndWait(ctx context.Context, accountName, containerName, blobName string, input CopyInput, pollingInterval time.Duration) error {
	if _, err := client.Copy(ctx, accountName, containerName, blobName, input); err != nil {
		return fmt.Errorf("Error copying: %s", err)
	}

	for true {
		getInput := GetPropertiesInput{
			LeaseID: input.LeaseID,
		}
		getResult, err := client.GetProperties(ctx, accountName, containerName, blobName, getInput)
		if err != nil {
			return fmt.Errorf("")
		}

		switch getResult.CopyStatus {
		case Aborted:
			return fmt.Errorf("Copy was aborted: %s", getResult.CopyStatusDescription)

		case Failed:
			return fmt.Errorf("Copy failed: %s", getResult.CopyStatusDescription)

		case Success:
			return nil

		case Pending:
			time.Sleep(pollingInterval)
			continue
		}
	}

	return fmt.Errorf("Unexpected error waiting for the copy to complete")
}
