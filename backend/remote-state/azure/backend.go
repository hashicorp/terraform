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

			// TODO: rename these fields
			// TODO: support for custom resource manager endpoints
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
	AccessKey         string
	ClientID          string
	ClientSecret      string
	Environment       string
	MsiEndpoint       string
	ResourceGroupName string
	SubscriptionID    string
	TenantID          string
	UseMsi            bool
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
		MsiEndpoint:        data.Get("msi_endpoint").(string),
		ResourceGroupName:  data.Get("resource_group_name").(string),
		StorageAccountName: data.Get("storage_account_name").(string),
		SubscriptionID:     data.Get("arm_subscription_id").(string),
		TenantID:           data.Get("arm_tenant_id").(string),
		UseMsi:             data.Get("use_msi").(bool),
	}

	armClient, err := buildArmClient(config)
	if err != nil {
		return err
	}

	if config.AccessKey == "" && config.ResourceGroupName == "" {
		return fmt.Errorf("Either an Access Key or the Resource Group for the Storage Account must be specified")
	}

	b.armClient = armClient
	return nil
}
