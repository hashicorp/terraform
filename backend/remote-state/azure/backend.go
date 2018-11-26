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

			"client_secret": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Client Secret.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_SECRET", ""),
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
}

type BackendConfig struct {
	// Required
	StorageAccountName string

	// Optional
	AccessKey                     string
	ClientID                      string
	ClientSecret                  string
	CustomResourceManagerEndpoint string
	Environment                   string
	MsiEndpoint                   string
	ResourceGroupName             string
	SasToken                      string
	SubscriptionID                string
	TenantID                      string
	UseMsi                        bool
}

func (b *Backend) configure(ctx context.Context) error {
	if b.containerName != "" {
		return nil
	}

	// Grab the resource data
	data := schema.FromContextBackendConfig(ctx)
	b.containerName = data.Get("container_name").(string)
	b.keyName = data.Get("key").(string)

	// support for previously deprecated fields
	clientId := valueFromDeprecatedField(data, "client_id", "arm_client_id")
	clientSecret := valueFromDeprecatedField(data, "client_secret", "arm_client_secret")
	subscriptionId := valueFromDeprecatedField(data, "subscription_id", "arm_subscription_id")
	tenantId := valueFromDeprecatedField(data, "tenant_id", "arm_tenant_id")

	config := BackendConfig{
		AccessKey:                     data.Get("access_key").(string),
		ClientID:                      clientId,
		ClientSecret:                  clientSecret,
		CustomResourceManagerEndpoint: data.Get("endpoint").(string),
		Environment:                   data.Get("environment").(string),
		MsiEndpoint:                   data.Get("msi_endpoint").(string),
		ResourceGroupName:             data.Get("resource_group_name").(string),
		SasToken:                      data.Get("sas_token").(string),
		StorageAccountName:            data.Get("storage_account_name").(string),
		SubscriptionID:                subscriptionId,
		TenantID:                      tenantId,
		UseMsi:                        data.Get("use_msi").(bool),
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
