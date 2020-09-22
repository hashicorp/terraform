package containers

import (
	"context"

	"github.com/Azure/go-autorest/autorest"
)

type StorageContainer interface {
	Create(ctx context.Context, accountName, containerName string, input CreateInput) (result CreateResponse, err error)
	Delete(ctx context.Context, accountName, containerName string) (result autorest.Response, err error)
	GetProperties(ctx context.Context, accountName, containerName string) (ContainerProperties, error)
	GetPropertiesWithLeaseID(ctx context.Context, accountName, containerName, leaseID string) (result ContainerProperties, err error)
	AcquireLease(ctx context.Context, accountName, containerName string, input AcquireLeaseInput) (result AcquireLeaseResponse, err error)
	BreakLease(ctx context.Context, accountName, containerName string, input BreakLeaseInput) (result BreakLeaseResponse, err error)
	ChangeLease(ctx context.Context, accountName, containerName string, input ChangeLeaseInput) (result ChangeLeaseResponse, err error)
	ReleaseLease(ctx context.Context, accountName, containerName, leaseID string) (result autorest.Response, err error)
	RenewLease(ctx context.Context, accountName, containerName, leaseID string) (result autorest.Response, err error)
	ListBlobs(ctx context.Context, accountName, containerName string, input ListBlobsInput) (result ListBlobsResult, err error)
	GetResourceManagerResourceID(subscriptionID, resourceGroup, accountName, containerName string) string
	SetAccessControl(ctx context.Context, accountName, containerName string, level AccessLevel) (autorest.Response, error)
	SetAccessControlWithLeaseID(ctx context.Context, accountName, containerName, leaseID string, level AccessLevel) (result autorest.Response, err error)
	SetMetaData(ctx context.Context, accountName, containerName string, metaData map[string]string) (autorest.Response, error)
	SetMetaDataWithLeaseID(ctx context.Context, accountName, containerName, leaseID string, metaData map[string]string) (result autorest.Response, err error)
}
