package azure

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
)

// New creates a new backend for Azure remote state.
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

			"metadata_host": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_METADATA_HOST", ""),
				Description: "The Metadata URL which will be used to obtain the Cloud Environment.",
			},

			"environment": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Azure cloud environment.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_ENVIRONMENT", "public"),
			},

			"access_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The access key.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_ACCESS_KEY", ""),
			},

			"sas_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A SAS Token used to interact with the Blob Storage Account.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_SAS_TOKEN", ""),
			},

			"snapshot": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Enable/Disable automatic blob snapshotting",
				DefaultFunc: schema.EnvDefaultFunc("ARM_SNAPSHOT", false),
			},

			"resource_group_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The resource group name.",
			},

			"client_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Client ID.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_ID", ""),
			},

			"endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A custom Endpoint used to access the Azure Resource Manager API's.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_ENDPOINT", ""),
			},

			"subscription_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Subscription ID.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_SUBSCRIPTION_ID", ""),
			},

			"tenant_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Tenant ID.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_TENANT_ID", ""),
			},

			// Service Principal (Client Certificate) specific
			"client_certificate_password": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The password associated with the Client Certificate specified in `client_certificate_path`",
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_CERTIFICATE_PASSWORD", ""),
			},
			"client_certificate_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path to the PFX file used as the Client Certificate when authenticating as a Service Principal",
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_CERTIFICATE_PATH", ""),
			},

			// Service Principal (Client Secret) specific
			"client_secret": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Client Secret.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_SECRET", ""),
			},

			// Managed Service Identity specific
			"use_msi": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Should Managed Service Identity be used?.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_USE_MSI", false),
			},
			"msi_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Managed Service Identity Endpoint.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_MSI_ENDPOINT", ""),
			},

			"endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A custom Endpoint used to access the Azure Resource Manager API's.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_ENDPOINT", ""),
			},

			"key_vault_key_identifier": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Azure Key Vault Key Identifier.",
				DefaultFunc: schema.EnvDefaultFunc("KEY_VAULT_KEY_IDENTIFIER", ""),
			},

			// Deprecated fields
			"arm_client_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Client ID.",
				Deprecated:  "`arm_client_id` has been replaced by `client_id`",
			},

			"arm_client_secret": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Client Secret.",
				Deprecated:  "`arm_client_secret` has been replaced by `client_secret`",
			},

			"arm_subscription_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Subscription ID.",
				Deprecated:  "`arm_subscription_id` has been replaced by `subscription_id`",
			},

			"arm_tenant_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Tenant ID.",
				Deprecated:  "`arm_tenant_id` has been replaced by `tenant_id`",
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
	armClient     *ArmClient
	containerName string
	keyName       string
	accountName   string
	snapshot      bool
}

type BackendConfig struct {
	// Required
	StorageAccountName string

	// Optional
	AccessKey                     string
	ClientID                      string
	ClientCertificatePassword     string
	ClientCertificatePath         string
	ClientSecret                  string
	CustomResourceManagerEndpoint string
	MetadataHost                  string
	Environment                   string
	MsiEndpoint                   string
	ResourceGroupName             string
	SasToken                      string
	SubscriptionID                string
	TenantID                      string
	UseMsi                        bool
	KeyVaultKeyIdentifier         string
}

func (b *Backend) configure(ctx context.Context) error {
	if b.containerName != "" {
		return nil
	}

	// Grab the resource data
	data := schema.FromContextBackendConfig(ctx)
	b.containerName = data.Get("container_name").(string)
	b.accountName = data.Get("storage_account_name").(string)
	b.keyName = data.Get("key").(string)
	b.snapshot = data.Get("snapshot").(bool)

	// support for previously deprecated fields
	clientId := valueFromDeprecatedField(data, "client_id", "arm_client_id")
	clientSecret := valueFromDeprecatedField(data, "client_secret", "arm_client_secret")
	subscriptionId := valueFromDeprecatedField(data, "subscription_id", "arm_subscription_id")
	tenantId := valueFromDeprecatedField(data, "tenant_id", "arm_tenant_id")

	config := BackendConfig{
		AccessKey:                     data.Get("access_key").(string),
		ClientID:                      clientId,
		ClientCertificatePassword:     data.Get("client_certificate_password").(string),
		ClientCertificatePath:         data.Get("client_certificate_path").(string),
		ClientSecret:                  clientSecret,
		CustomResourceManagerEndpoint: data.Get("endpoint").(string),
		MetadataHost:                  data.Get("metadata_host").(string),
		Environment:                   data.Get("environment").(string),
		MsiEndpoint:                   data.Get("msi_endpoint").(string),
		ResourceGroupName:             data.Get("resource_group_name").(string),
		SasToken:                      data.Get("sas_token").(string),
		StorageAccountName:            data.Get("storage_account_name").(string),
		SubscriptionID:                subscriptionId,
		TenantID:                      tenantId,
		UseMsi:                        data.Get("use_msi").(bool),
		KeyVaultKeyIdentifier:         data.Get("key_vault_key_identifier").(string),
	}

	armClient, err := buildArmClient(config)
	if err != nil {
		return err
	}

	if config.AccessKey == "" && config.SasToken == "" && config.ResourceGroupName == "" {
		return fmt.Errorf("Either an Access Key / SAS Token or the Resource Group for the Storage Account must be specified")
	}

	b.armClient = armClient
	return nil
}

func valueFromDeprecatedField(d *schema.ResourceData, key, deprecatedFieldKey string) string {
	v := d.Get(key).(string)

	if v == "" {
		v = d.Get(deprecatedFieldKey).(string)
	}

	return v
}
