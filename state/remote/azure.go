package remote

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Azure/azure-sdk-for-go/arm/storage"
	mainStorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	riviera "github.com/jen20/riviera/azure"
)

func azureFactory(conf map[string]string) (Client, error) {
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

	env, err := getAzureEnvironmentFromConf(conf)
	if err != nil {
		return nil, err
	}

	accessKey, ok := confOrEnv(conf, "access_key", "ARM_ACCESS_KEY")
	if !ok {
		resourceGroupName, ok := conf["resource_group_name"]
		if !ok {
			return nil, fmt.Errorf("missing 'resource_group_name' configuration")
		}

		var err error
		accessKey, err = getStorageAccountAccessKey(conf, resourceGroupName, storageAccountName, env)
		if err != nil {
			return nil, fmt.Errorf("Couldn't read access key from storage account: %s.", err)
		}
	}

	storageClient, err := mainStorage.NewClient(storageAccountName, accessKey, env.StorageEndpointSuffix,
		mainStorage.DefaultAPIVersion, true)
	if err != nil {
		return nil, fmt.Errorf("Error creating storage client for storage account %q: %s", storageAccountName, err)
	}

	blobClient := storageClient.GetBlobService()
	leaseID, _ := confOrEnv(conf, "lease_id", "ARM_LEASE_ID")

	return &AzureClient{
		blobClient:    &blobClient,
		containerName: containerName,
		keyName:       keyName,
		leaseID:       leaseID,
	}, nil
}

func getStorageAccountAccessKey(conf map[string]string, resourceGroupName, storageAccountName string, env azure.Environment) (string, error) {
	creds, err := getCredentialsFromConf(conf, env)
	if err != nil {
		return "", err
	}

	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, creds.TenantID)
	if err != nil {
		return "", err
	}
	if oauthConfig == nil {
		return "", fmt.Errorf("Unable to configure OAuthConfig for tenant %s", creds.TenantID)
	}

	spt, err := adal.NewServicePrincipalToken(*oauthConfig, creds.ClientID, creds.ClientSecret, env.ResourceManagerEndpoint)
	if err != nil {
		return "", err
	}

	accountsClient := storage.NewAccountsClientWithBaseURI(env.ResourceManagerEndpoint, creds.SubscriptionID)
	accountsClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	keys, err := accountsClient.ListKeys(resourceGroupName, storageAccountName)
	if err != nil {
		return "", fmt.Errorf("Error retrieving keys for storage account %q: %s", storageAccountName, err)
	}

	if keys.Keys == nil {
		return "", fmt.Errorf("Nil key returned for storage account %q", storageAccountName)
	}

	accessKeys := *keys.Keys
	return *accessKeys[0].Value, nil
}

func getCredentialsFromConf(conf map[string]string, env azure.Environment) (*riviera.AzureResourceManagerCredentials, error) {
	subscriptionID, ok := confOrEnv(conf, "arm_subscription_id", "ARM_SUBSCRIPTION_ID")
	if !ok {
		return nil, fmt.Errorf("missing 'arm_subscription_id' configuration")
	}
	clientID, ok := confOrEnv(conf, "arm_client_id", "ARM_CLIENT_ID")
	if !ok {
		return nil, fmt.Errorf("missing 'arm_client_id' configuration")
	}
	clientSecret, ok := confOrEnv(conf, "arm_client_secret", "ARM_CLIENT_SECRET")
	if !ok {
		return nil, fmt.Errorf("missing 'arm_client_secret' configuration")
	}
	tenantID, ok := confOrEnv(conf, "arm_tenant_id", "ARM_TENANT_ID")
	if !ok {
		return nil, fmt.Errorf("missing 'arm_tenant_id' configuration")
	}

	return &riviera.AzureResourceManagerCredentials{
		SubscriptionID:          subscriptionID,
		ClientID:                clientID,
		ClientSecret:            clientSecret,
		TenantID:                tenantID,
		ActiveDirectoryEndpoint: env.ActiveDirectoryEndpoint,
		ResourceManagerEndpoint: env.ResourceManagerEndpoint,
	}, nil
}

func getAzureEnvironmentFromConf(conf map[string]string) (azure.Environment, error) {
	envName, ok := confOrEnv(conf, "environment", "ARM_ENVIRONMENT")
	if !ok {
		return azure.PublicCloud, nil
	}

	env, err := azure.EnvironmentFromName(envName)
	if err != nil {
		// try again with wrapped value to support readable values like german instead of AZUREGERMANCLOUD
		var innerErr error
		env, innerErr = azure.EnvironmentFromName(fmt.Sprintf("AZURE%sCLOUD", envName))
		if innerErr != nil {
			return env, fmt.Errorf("invalid 'environment' configuration: %s", err)
		}
	}

	return env, nil
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
	leaseID       string
}

func (c *AzureClient) Get() (*Payload, error) {
	containerReference := c.blobClient.GetContainerReference(c.containerName)
	blobReference := containerReference.GetBlobReference(c.keyName)
	options := &mainStorage.GetBlobOptions{}
	blob, err := blobReference.Get(options)
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
	setOptions := &mainStorage.SetBlobPropertiesOptions{}
	putOptions := &mainStorage.PutBlobOptions{}

	containerReference := c.blobClient.GetContainerReference(c.containerName)
	blobReference := containerReference.GetBlobReference(c.keyName)

	blobReference.Properties.ContentType = "application/json"
	blobReference.Properties.ContentLength = int64(len(data))

	if c.leaseID != "" {
		setOptions.LeaseID = c.leaseID
		putOptions.LeaseID = c.leaseID
	}

	reader := bytes.NewReader(data)

	err := blobReference.CreateBlockBlobFromReader(reader, putOptions)
	if err != nil {
		return err
	}

	return blobReference.SetProperties(setOptions)
}

func (c *AzureClient) Delete() error {
	containerReference := c.blobClient.GetContainerReference(c.containerName)
	blobReference := containerReference.GetBlobReference(c.keyName)
	options := &mainStorage.DeleteBlobOptions{}

	if c.leaseID != "" {
		options.LeaseID = c.leaseID
	}

	return blobReference.Delete(options)
}
