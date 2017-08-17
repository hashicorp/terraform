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
			"storage_account_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the storage account.",
			},

			"container_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The container name.",
			},

			"key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The blob key.",
			},

			"environment": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Azure cloud environment.",
				Default:     "",
			},

			"access_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The access key.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_ACCESS_KEY", ""),
			},

			"resource_group_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The resource group name.",
			},

			"arm_subscription_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Subscription ID.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_SUBSCRIPTION_ID", ""),
			},

			"arm_client_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Client ID.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_ID", ""),
			},

			"arm_client_secret": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Client Secret.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_SECRET", ""),
			},

			"arm_tenant_id": &schema.Schema{
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

func (b *Backend) configure(ctx context.Context) error {
	if b.containerName != "" {
		return nil
	}

	// Grab the resource data
	data := schema.FromContextBackendConfig(ctx)

	b.containerName = data.Get("container_name").(string)
	b.keyName = data.Get("key").(string)

	blobClient, err := getBlobClient(data)
	if err != nil {
		return err
	}
	b.blobClient = blobClient

	return nil
}

func getBlobClient(d *schema.ResourceData) (storage.BlobStorageClient, error) {
	var client storage.BlobStorageClient

	env, err := getAzureEnvironment(d.Get("environment").(string))
	if err != nil {
		return client, err
	}

	storageAccountName := d.Get("storage_account_name").(string)

	accessKey, err := getAccessKey(d, storageAccountName, env)
	if err != nil {
		return client, err
	}

	storageClient, err := storage.NewClient(storageAccountName, accessKey, env.StorageEndpointSuffix,
		storage.DefaultAPIVersion, true)
	if err != nil {
		return client, fmt.Errorf("Error creating storage client for storage account %q: %s", storageAccountName, err)
	}

	client = storageClient.GetBlobService()
	return client, nil
}

func getAccessKey(d *schema.ResourceData, storageAccountName string, env azure.Environment) (string, error) {
	if key, ok := d.GetOk("access_key"); ok {
		return key.(string), nil
	}

	resourceGroupName, rgOk := d.GetOk("resource_group_name")
	subscriptionID, subOk := d.GetOk("arm_subscription_id")
	clientID, clientIDOk := d.GetOk("arm_client_id")
	clientSecret, clientSecretOK := d.GetOk("arm_client_secret")
	tenantID, tenantIDOk := d.GetOk("arm_tenant_id")
	if !rgOk || !subOk || !clientIDOk || !clientSecretOK || !tenantIDOk {
		return "", fmt.Errorf("resource_group_name and credentials must be provided when access_key is absent")
	}

	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, tenantID.(string))
	if err != nil {
		return "", err
	}

	spt, err := adal.NewServicePrincipalToken(*oauthConfig, clientID.(string), clientSecret.(string), env.ResourceManagerEndpoint)
	if err != nil {
		return "", err
	}

	accountsClient := armStorage.NewAccountsClientWithBaseURI(env.ResourceManagerEndpoint, subscriptionID.(string))
	accountsClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	keys, err := accountsClient.ListKeys(resourceGroupName.(string), storageAccountName)
	if err != nil {
		return "", fmt.Errorf("Error retrieving keys for storage account %q: %s", storageAccountName, err)
	}

	if keys.Keys == nil {
		return "", fmt.Errorf("Nil key returned for storage account %q", storageAccountName)
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
