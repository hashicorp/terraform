package azure

import (
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/affinitygroup"
	"github.com/Azure/azure-sdk-for-go/management/hostedservice"
	"github.com/Azure/azure-sdk-for-go/management/networksecuritygroup"
	"github.com/Azure/azure-sdk-for-go/management/osimage"
	"github.com/Azure/azure-sdk-for-go/management/sql"
	"github.com/Azure/azure-sdk-for-go/management/storageservice"
	"github.com/Azure/azure-sdk-for-go/management/virtualmachine"
	"github.com/Azure/azure-sdk-for-go/management/virtualmachinedisk"
	"github.com/Azure/azure-sdk-for-go/management/virtualmachineimage"
	"github.com/Azure/azure-sdk-for-go/management/virtualnetwork"
	"github.com/Azure/azure-sdk-for-go/storage"
)

// Config is the configuration structure used to instantiate a
// new Azure management client.
type Config struct {
	Settings       []byte
	SubscriptionID string
	Certificate    []byte
	ManagementURL  string
}

// Client contains all the handles required for managing Azure services.
type Client struct {
	mgmtClient management.Client

	affinityGroupClient affinitygroup.AffinityGroupClient

	hostedServiceClient hostedservice.HostedServiceClient

	osImageClient osimage.OSImageClient

	sqlClient sql.SQLDatabaseClient

	storageServiceClient storageservice.StorageServiceClient

	vmClient virtualmachine.VirtualMachineClient

	vmDiskClient virtualmachinedisk.DiskClient

	vmImageClient virtualmachineimage.Client

	// unfortunately; because of how Azure's network API works; doing networking operations
	// concurrently is very hazardous, and we need a mutex to guard the VirtualNetworkClient.
	vnetClient virtualnetwork.VirtualNetworkClient
	vnetMutex  *sync.Mutex

	// same as the above for security group rule operations:
	secGroupClient networksecuritygroup.SecurityGroupClient
	secGroupMutex  *sync.Mutex
}

// getStorageClientForStorageService is helper method which returns the
// storage.Client associated to the given storage service name.
func (c Client) getStorageClientForStorageService(serviceName string) (storage.Client, error) {
	var storageClient storage.Client

	keys, err := c.storageServiceClient.GetStorageServiceKeys(serviceName)
	if err != nil {
		return storageClient, fmt.Errorf("Failed getting Storage Service keys for %s: %s", serviceName, err)
	}

	storageClient, err = storage.NewBasicClient(serviceName, keys.PrimaryKey)
	if err != nil {
		return storageClient, fmt.Errorf("Failed creating Storage Service client for %s: %s", serviceName, err)
	}

	return storageClient, err
}

// getStorageServiceBlobClient is a helper method which returns the
// storage.BlobStorageClient associated to the given storage service name.
func (c Client) getStorageServiceBlobClient(serviceName string) (storage.BlobStorageClient, error) {
	storageClient, err := c.getStorageClientForStorageService(serviceName)
	if err != nil {
		return storage.BlobStorageClient{}, err
	}

	return storageClient.GetBlobService(), nil
}

// getStorageServiceQueueClient is a helper method which returns the
// storage.QueueServiceClient associated to the given storage service name.
func (c Client) getStorageServiceQueueClient(serviceName string) (storage.QueueServiceClient, error) {
	storageClient, err := c.getStorageClientForStorageService(serviceName)
	if err != nil {
		return storage.QueueServiceClient{}, err
	}

	return storageClient.GetQueueService(), err
}

func (c *Config) NewClientFromSettingsData() (*Client, error) {
	mc, err := management.ClientFromPublishSettingsData(c.Settings, c.SubscriptionID)
	if err != nil {
		return nil, err
	}

	return &Client{
		mgmtClient:           mc,
		affinityGroupClient:  affinitygroup.NewClient(mc),
		hostedServiceClient:  hostedservice.NewClient(mc),
		secGroupClient:       networksecuritygroup.NewClient(mc),
		secGroupMutex:        &sync.Mutex{},
		osImageClient:        osimage.NewClient(mc),
		sqlClient:            sql.NewClient(mc),
		storageServiceClient: storageservice.NewClient(mc),
		vmClient:             virtualmachine.NewClient(mc),
		vmDiskClient:         virtualmachinedisk.NewClient(mc),
		vmImageClient:        virtualmachineimage.NewClient(mc),
		vnetClient:           virtualnetwork.NewClient(mc),
		vnetMutex:            &sync.Mutex{},
	}, nil
}

// NewClient returns a new Azure management client created
// using a subscription ID and certificate.
func (c *Config) NewClient() (*Client, error) {
	mc, err := management.NewClient(c.SubscriptionID, c.Certificate)
	if err != nil {
		return nil, nil
	}

	return &Client{
		mgmtClient:           mc,
		affinityGroupClient:  affinitygroup.NewClient(mc),
		hostedServiceClient:  hostedservice.NewClient(mc),
		secGroupClient:       networksecuritygroup.NewClient(mc),
		secGroupMutex:        &sync.Mutex{},
		osImageClient:        osimage.NewClient(mc),
		sqlClient:            sql.NewClient(mc),
		storageServiceClient: storageservice.NewClient(mc),
		vmClient:             virtualmachine.NewClient(mc),
		vmDiskClient:         virtualmachinedisk.NewClient(mc),
		vmImageClient:        virtualmachineimage.NewClient(mc),
		vnetClient:           virtualnetwork.NewClient(mc),
		vnetMutex:            &sync.Mutex{},
	}, nil
}
