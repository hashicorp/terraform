package azure

import (
	"encoding/xml"
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
	"github.com/hashicorp/terraform/helper/pathorcontents"
)

// AsmClient contains the handles to all the specific Azure Service Manager
// resource classes'respective clients.
type AsmClient struct {
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

const settingsPathWarnMsg = `
settings_file is not valid XML, so we are assuming it is a file path. This
support will be removed in the future. Please update your configuration to use
${file("filename.publishsettings")} instead.`

func validateAsmSettingsFile(v interface{}, _ string) ([]string, []error) {
	value := v.(string)
	if value == "" {
		return nil, nil
	}

	_, warnings, errors := readAsmSettingsFile(value)
	return warnings, errors
}

func validateAsmPublishSettings(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if value == "" {
		return nil, nil
	}

	var settings settingsData
	if err := xml.Unmarshal([]byte(value), &settings); err != nil {
		es = append(es, fmt.Errorf("error parsing publish_settings as XML: %s", err))

	}
	return
}

// settingsData is a private struct used to test the unmarshalling of the
// settingsFile contents, to determine if the contents are valid XML
type settingsData struct {
	XMLName xml.Name `xml:"PublishData"`
}

func readAsmSettingsFile(pathOrContents string) (s string, ws []string, es []error) {
	s, wasPath, err := pathorcontents.Read(pathOrContents)
	if err != nil {
		es = append(es, fmt.Errorf("error reading settings_file: %s", err))
	}

	if wasPath {
		ws = append(ws, settingsPathWarnMsg)
	}

	var settings settingsData
	if err := xml.Unmarshal([]byte(s), &settings); err != nil {
		es = append(es, fmt.Errorf("error parsing settings_file as XML: %s", err))

	}

	return
}

// asmCredentialsProvided is a helper method which indicates whether all the
// credentials required for authenticating against the ASM APIs were provided.
func (c *Config) asmCredentialsProvided() bool {
	if c.Settings != "" || (c.SubscriptionID != "" && c.Certificate != "") {
		return true
	}

	return false
}

// getAsmClient is a helper method which returns a fully instantiated
// *AsmClient based on the Config's current settings.
func (c *Config) getAsmClient() (client *AsmClient, err error) {
	var mc management.Client

	// first; check whether an ASM publishsettings file was provided:
	if c.Settings != "" {
		// then, configure using the settings provided in the file:
		mc, err = management.ClientFromPublishSettingsData([]byte(c.Settings), c.SubscriptionID)
		if err != nil {
			return
		}
	} else {
		// else, create the client manually:
		mc, err = management.NewClient(c.SubscriptionID, []byte(c.Certificate))
		if err != nil {
			return
		}
	}

	return &AsmClient{
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

// getStorageClientForStorageService is helper method which returns the
// storage.Client associated to the given storage service name.
func (c *AsmClient) getStorageClientForStorageService(serviceName string) (storage.Client, error) {
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
func (c *AsmClient) getStorageServiceBlobClient(serviceName string) (storage.BlobStorageClient, error) {
	storageClient, err := c.getStorageClientForStorageService(serviceName)
	if err != nil {
		return storage.BlobStorageClient{}, err
	}

	return storageClient.GetBlobService(), nil
}

// getStorageServiceQueueClient is a helper method which returns the
// storage.QueueServiceClient associated to the given storage service name.
func (c *AsmClient) getStorageServiceQueueClient(serviceName string) (storage.QueueServiceClient, error) {
	storageClient, err := c.getStorageClientForStorageService(serviceName)
	if err != nil {
		return storage.QueueServiceClient{}, err
	}

	return storageClient.GetQueueService(), err
}
