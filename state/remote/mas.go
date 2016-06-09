package remote

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Azure/azure-sdk-for-go/arm/storage"
	mainStorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	riviera "github.com/jen20/riviera/azure"
)

func masFactory(conf map[string]string) (Client, error) {
	storageAccountName, ok := conf["storage_account_name"]
	if !ok {
		return nil, fmt.Errorf("missing 'storage_account_name' configuration")
	}
	containerName, ok := conf["container_name"]
	if !ok {
		return nil, fmt.Errorf("missing 'container_name' configuration")
	}
	keyName, ok := conf["key"]
	if !ok {
		return nil, fmt.Errorf("missing 'key' configuration")
	}

	accessKey, ok := confOrEnv(conf, "access_key", "ARM_ACCESS_KEY")
	if !ok {
		resourceGroupName, ok := conf["resource_group_name"]
		if !ok {
			return nil, fmt.Errorf("missing 'resource_group' configuration")
		}

		var err error
		accessKey, err = getStorageAccountAccessKey(conf, resourceGroupName, storageAccountName)
		if err != nil {
			return nil, fmt.Errorf("Couldn't instantiate blob storage client: %s.", err)
		}
	}

	storageClient, err := mainStorage.NewBasicClient(storageAccountName, accessKey)
	if err != nil {
		return nil, fmt.Errorf("Error creating storage client for storage account %q: %s", storageAccountName, err)
	}

	blobClient := storageClient.GetBlobService()

	return &AzureClient{
		blobClient:    &blobClient,
		containerName: containerName,
		keyName:       keyName,
	}, nil
}

func getStorageAccountAccessKey(conf map[string]string, resourceGroupName, storageAccountName string) (string, error) {
	subscriptionID, ok := confOrEnv(conf, "arm_subscription_id", "ARM_SUBSCRIPTION_ID")
	if !ok {
		return "", fmt.Errorf("missing 'arm_subscription_id' configuration")
	}
	clientID, ok := confOrEnv(conf, "arm_client_id", "ARM_CLIENT_ID")
	if !ok {
		return "", fmt.Errorf("missing 'arm_client_id' configuration")
	}
	clientSecret, ok := confOrEnv(conf, "arm_client_secret", "ARM_CLIENT_SECRET")
	if !ok {
		return "", fmt.Errorf("missing 'arm_client_secret' configuration")
	}
	tenantID, ok := confOrEnv(conf, "arm_tenant_id", "ARM_TENANT_ID")
	if !ok {
		return "", fmt.Errorf("missing 'arm_tenant_id' configuration")
	}

	rivieraClient, err := riviera.NewClient(&riviera.AzureResourceManagerCredentials{
		SubscriptionID: subscriptionID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		TenantID:       tenantID,
	})
	if err != nil {
		return "", fmt.Errorf("Error creating Riviera client: %s", err)
	}

	request := rivieraClient.NewRequest()
	request.Command = riviera.RegisterResourceProvider{
		Namespace: "Microsoft.Storage",
	}

	response, err := request.Execute()
	if err != nil {
		return "", fmt.Errorf("Cannot request provider registration for Azure Resource Manager: %s.", err)
	}

	if !response.IsSuccessful() {
		return "", fmt.Errorf("Credentials for acessing the Azure Resource Manager API are likely " +
			"to be incorrect, or\n  the service principal does not have permission to use " +
			"the Azure Service Management\n  API.")
	}

	oauthConfig, err := azure.PublicCloud.OAuthConfigForTenant(tenantID)
	if err != nil {
		return "", err
	}

	if oauthConfig == nil {
		return "", fmt.Errorf("Unable to configure OAuthConfig for tenant %s", tenantID)
	}

	spt, err := azure.NewServicePrincipalToken(*oauthConfig, clientID, clientSecret, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return "", err
	}

	accountsClient := storage.NewAccountsClient(subscriptionID)
	accountsClient.Authorizer = spt

	keys, err := accountsClient.ListKeys(resourceGroupName, storageAccountName)
	if err != nil {
		return "", fmt.Errorf("Error retrieving keys for storage account %q: %s", storageAccountName, err)
	}

	if keys.Key1 == nil {
		return "", fmt.Errorf("Nil key returned for storage account %q", storageAccountName)
	}

	return *keys.Key1, nil
}

func confOrEnv(conf map[string]string, confKey, envVar string) (string, bool) {
	value, ok := conf[confKey]
	if ok {
		return value, true
	}

	value = os.Getenv(envVar)

	return value, value != ""
}

type AzureClient struct {
	blobClient    *mainStorage.BlobStorageClient
	containerName string
	keyName       string
}

func (c *AzureClient) Get() (*Payload, error) {
	blob, err := c.blobClient.GetBlob(c.containerName, c.keyName)
	if err != nil {
		if storErr, ok := err.(mainStorage.AzureStorageServiceError); ok {
			if storErr.Code == "BlobNotFound" {
				return nil, nil
			}
		}
		return nil, err
	}

	defer blob.Close()

	data, err := ioutil.ReadAll(blob)
	if err != nil {
		return nil, err
	}

	payload := &Payload{
		Data: data,
	}

	// If there was no data, then return nil
	if len(payload.Data) == 0 {
		return nil, nil
	}

	return payload, nil
}

func (c *AzureClient) Put(data []byte) error {
	return c.blobClient.CreateBlockBlobFromReader(
		c.containerName,
		c.keyName,
		uint64(len(data)),
		bytes.NewReader(data),
		map[string]string{
			"Content-Type": "application/json",
		},
	)
}

func (c *AzureClient) Delete() error {
	return c.blobClient.DeleteBlob(c.containerName, c.keyName, nil)
}
