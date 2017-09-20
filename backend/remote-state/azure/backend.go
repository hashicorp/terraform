package azure

import (
	"context"
	"fmt"

	armStorage "github.com/Azure/azure-sdk-for-go/arm/storage"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
)

// New creates a new backend for S3 remote state.
func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"storage_account_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the storage account.",
			},

			"container_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The container name.",
			},

			"key": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The blob key.",
			},

			"environment": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Azure cloud environment.",
				Default:     "",
			},

			"access_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The access key.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_ACCESS_KEY", ""),
			},

			"resource_group_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The resource group name.",
			},

			"arm_subscription_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Subscription ID.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_SUBSCRIPTION_ID", ""),
			},

			"arm_client_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Client ID.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_ID", ""),
			},

			"arm_client_secret": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Client Secret.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_SECRET", ""),
			},

			"arm_tenant_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Tenant ID.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_TENANT_ID", ""),
			},
		},
	}

	result := &Backend{Backend: s}
	result.Backend.ConfigureFunc = result.configure
	return result
}

type Backend struct {
	*schema.Backend

	// The fields below are set from configure
	blobClient storage.BlobStorageClient

	containerName string
	keyName       string
	leaseID       string
}

type BackendConfig struct {
	AccessKey          string
	Environment        string
	ClientID           string
	ClientSecret       string
	ResourceGroupName  string
	StorageAccountName string
	SubscriptionID     string
	TenantID           string
}

func (b *Backend) configure(ctx context.Context) error {
	if b.containerName != "" {
		return nil
	}

	// Grab the resource data
	data := schema.FromContextBackendConfig(ctx)

	b.containerName = data.Get("container_name").(string)
	b.keyName = data.Get("key").(string)

	config := BackendConfig{
		AccessKey:          data.Get("access_key").(string),
		ClientID:           data.Get("arm_client_id").(string),
		ClientSecret:       data.Get("arm_client_secret").(string),
		Environment:        data.Get("environment").(string),
		ResourceGroupName:  data.Get("resource_group_name").(string),
		StorageAccountName: data.Get("storage_account_name").(string),
		SubscriptionID:     data.Get("arm_subscription_id").(string),
		TenantID:           data.Get("arm_tenant_id").(string),
	}

	blobClient, err := getBlobClient(config)
	if err != nil {
		return err
	}
	b.blobClient = blobClient

	return nil
}

func getBlobClient(config BackendConfig) (storage.BlobStorageClient, error) {
	var client storage.BlobStorageClient

	env, err := getAzureEnvironment(config.Environment)
	if err != nil {
		return client, err
	}

	accessKey, err := getAccessKey(config, env)
	if err != nil {
		return client, err
	}

	storageClient, err := storage.NewClient(config.StorageAccountName, accessKey, env.StorageEndpointSuffix,
		storage.DefaultAPIVersion, true)
	if err != nil {
		return client, fmt.Errorf("Error creating storage client for storage account %q: %s", config.StorageAccountName, err)
	}

	client = storageClient.GetBlobService()
	return client, nil
}

func getAccessKey(config BackendConfig, env azure.Environment) (string, error) {
	if config.AccessKey != "" {
		return config.AccessKey, nil
	}

	rgOk := config.ResourceGroupName != ""
	subOk := config.SubscriptionID != ""
	clientIDOk := config.ClientID != ""
	clientSecretOK := config.ClientSecret != ""
	tenantIDOk := config.TenantID != ""
	if !rgOk || !subOk || !clientIDOk || !clientSecretOK || !tenantIDOk {
		return "", fmt.Errorf("resource_group_name and credentials must be provided when access_key is absent")
	}

	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, config.TenantID)
	if err != nil {
		return "", err
	}

	spt, err := adal.NewServicePrincipalToken(*oauthConfig, config.ClientID, config.ClientSecret, env.ResourceManagerEndpoint)
	if err != nil {
		return "", err
	}

	accountsClient := armStorage.NewAccountsClientWithBaseURI(env.ResourceManagerEndpoint, config.SubscriptionID)
	accountsClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	keys, err := accountsClient.ListKeys(config.ResourceGroupName, config.StorageAccountName)
	if err != nil {
		return "", fmt.Errorf("Error retrieving keys for storage account %q: %s", config.StorageAccountName, err)
	}

	if keys.Keys == nil {
		return "", fmt.Errorf("Nil key returned for storage account %q", config.StorageAccountName)
	}

	accessKeys := *keys.Keys
	return *accessKeys[0].Value, nil
}

func getAzureEnvironment(environment string) (azure.Environment, error) {
	if environment == "" {
		return azure.PublicCloud, nil
	}

	env, err := azure.EnvironmentFromName(environment)
	if err != nil {
		// try again with wrapped value to support readable values like german instead of AZUREGERMANCLOUD
		var innerErr error
		env, innerErr = azure.EnvironmentFromName(fmt.Sprintf("AZURE%sCLOUD", environment))
		if innerErr != nil {
			return env, fmt.Errorf("invalid 'environment' configuration: %s", err)
		}
	}

	return env, nil
}
