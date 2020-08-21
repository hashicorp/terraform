package blobs

import (
	"context"
	"os"
	"time"

	"github.com/Azure/go-autorest/autorest"
)

type StorageBlob interface {
	AppendBlock(ctx context.Context, accountName, containerName, blobName string, input AppendBlockInput) (result AppendBlockResult, err error)
	Copy(ctx context.Context, accountName, containerName, blobName string, input CopyInput) (result CopyResult, err error)
	AbortCopy(ctx context.Context, accountName, containerName, blobName string, input AbortCopyInput) (result autorest.Response, err error)
	CopyAndWait(ctx context.Context, accountName, containerName, blobName string, input CopyInput, pollingInterval time.Duration) error
	Delete(ctx context.Context, accountName, containerName, blobName string, input DeleteInput) (result autorest.Response, err error)
	DeleteSnapshot(ctx context.Context, accountName, containerName, blobName string, input DeleteSnapshotInput) (result autorest.Response, err error)
	DeleteSnapshots(ctx context.Context, accountName, containerName, blobName string, input DeleteSnapshotsInput) (result autorest.Response, err error)
	Get(ctx context.Context, accountName, containerName, blobName string, input GetInput) (result GetResult, err error)
	GetBlockList(ctx context.Context, accountName, containerName, blobName string, input GetBlockListInput) (result GetBlockListResult, err error)
	GetPageRanges(ctx context.Context, accountName, containerName, blobName string, input GetPageRangesInput) (result GetPageRangesResult, err error)
	IncrementalCopyBlob(ctx context.Context, accountName, containerName, blobName string, input IncrementalCopyBlobInput) (result autorest.Response, err error)
	AcquireLease(ctx context.Context, accountName, containerName, blobName string, input AcquireLeaseInput) (result AcquireLeaseResult, err error)
	BreakLease(ctx context.Context, accountName, containerName, blobName string, input BreakLeaseInput) (result autorest.Response, err error)
	ChangeLease(ctx context.Context, accountName, containerName, blobName string, input ChangeLeaseInput) (result ChangeLeaseResponse, err error)
	ReleaseLease(ctx context.Context, accountName, containerName, blobName, leaseID string) (result autorest.Response, err error)
	RenewLease(ctx context.Context, accountName, containerName, blobName, leaseID string) (result autorest.Response, err error)
	SetMetaData(ctx context.Context, accountName, containerName, blobName string, input SetMetaDataInput) (result autorest.Response, err error)
	GetProperties(ctx context.Context, accountName, containerName, blobName string, input GetPropertiesInput) (result GetPropertiesResult, err error)
	SetProperties(ctx context.Context, accountName, containerName, blobName string, input SetPropertiesInput) (result SetPropertiesResult, err error)
	PutAppendBlob(ctx context.Context, accountName, containerName, blobName string, input PutAppendBlobInput) (result autorest.Response, err error)
	PutBlock(ctx context.Context, accountName, containerName, blobName string, input PutBlockInput) (result PutBlockResult, err error)
	PutBlockBlob(ctx context.Context, accountName, containerName, blobName string, input PutBlockBlobInput) (result autorest.Response, err error)
	PutBlockBlobFromFile(ctx context.Context, accountName, containerName, blobName string, file *os.File, input PutBlockBlobInput) error
	PutBlockList(ctx context.Context, accountName, containerName, blobName string, input PutBlockListInput) (result PutBlockListResult, err error)
	PutBlockFromURL(ctx context.Context, accountName, containerName, blobName string, input PutBlockFromURLInput) (result PutBlockFromURLResult, err error)
	PutPageBlob(ctx context.Context, accountName, containerName, blobName string, input PutPageBlobInput) (result autorest.Response, err error)
	PutPageClear(ctx context.Context, accountName, containerName, blobName string, input PutPageClearInput) (result autorest.Response, err error)
	PutPageUpdate(ctx context.Context, accountName, containerName, blobName string, input PutPageUpdateInput) (result PutPageUpdateResult, err error)
	GetResourceID(accountName, containerName, blobName string) string
	SetTier(ctx context.Context, accountName, containerName, blobName string, tier AccessTier) (result autorest.Response, err error)
	Snapshot(ctx context.Context, accountName, containerName, blobName string, input SnapshotInput) (result SnapshotResult, err error)
	GetSnapshotProperties(ctx context.Context, accountName, containerName, blobName string, input GetSnapshotPropertiesInput) (result GetPropertiesResult, err error)
	Undelete(ctx context.Context, accountName, containerName, blobName string) (result autorest.Response, err error)
}
