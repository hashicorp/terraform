// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/legacy/helper/schema"
)

// New creates a new backend for Azure remote state.
func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Type:        schema.TypeString,
				Optional:    true, // TODO: make this Required in a future version
				Description: "The Subscription ID where the Storage Account is located.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_SUBSCRIPTION_ID", ""),
			},

			"resource_group_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Resource Group where the Storage Account is located.",
			},

			"storage_account_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the storage account.",
			},

			"container_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The container name to use in the Storage Account.",
			},

			"key": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The blob key to use in the Storage Container.",
			},

			"snapshot": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether to enable automatic blob snapshotting.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_SNAPSHOT", false),
			},

			"environment": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Cloud Environment which should be used. Possible values are public, usgovernment, and china. Defaults to public. Not used and should not be specified when `metadata_host` is specified.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_ENVIRONMENT", "public"),
			},

			"metadata_host": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"ARM_METADATA_HOSTNAME", "ARM_METADATA_HOST"}, ""), // TODO: remove support for `METADATA_HOST` in a future version
				Description: "The Hostname which should be used for the Azure Metadata Service.",
			},

			"access_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The access key to use when authenticating using a Storage Access Key.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_ACCESS_KEY", ""),
			},

			"sas_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The SAS Token to use when authenticating using a SAS Token.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_SAS_TOKEN", ""),
			},

			"tenant_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Tenant ID to use when authenticating using Azure Active Directory.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_TENANT_ID", ""),
			},

			"client_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Client ID to use when authenticating using Azure Active Directory.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_ID", ""),
			},

			"client_id_file_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path to a file containing the Client ID which should be used.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_ID_FILE_PATH", nil),
			},

			"endpoint": {
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "`endpoint` is deprecated in favor of `msi_endpoint`, it will be removed in a future version of Terraform",
			},

			// Client Certificate specific fields
			"client_certificate": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_CERTIFICATE", ""),
				Description: "Base64 encoded PKCS#12 certificate bundle to use when authenticating as a Service Principal using a Client Certificate",
			},

			"client_certificate_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path to the Client Certificate associated with the Service Principal for use when authenticating as a Service Principal using a Client Certificate.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_CERTIFICATE_PATH", ""),
			},

			"client_certificate_password": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The password associated with the Client Certificate. For use when authenticating as a Service Principal using a Client Certificate",
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_CERTIFICATE_PASSWORD", ""),
			},

			// Client Secret specific fields
			"client_secret": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Client Secret which should be used. For use When authenticating as a Service Principal using a Client Secret.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_SECRET", ""),
			},

			"client_secret_file_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path to a file containing the Client Secret which should be used. For use When authenticating as a Service Principal using a Client Secret.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_SECRET_FILE_PATH", nil),
			},

			// OIDC specific fields
			"use_oidc": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_USE_OIDC", false),
				Description: "Allow OpenID Connect to be used for authentication",
			},

			"ado_pipeline_service_connection_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID", "ARM_OIDC_AZURE_SERVICE_CONNECTION_ID"}, nil),
				Description: "The Azure DevOps Pipeline Service Connection ID.",
			},

			"oidc_request_token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"ARM_OIDC_REQUEST_TOKEN", "ACTIONS_ID_TOKEN_REQUEST_TOKEN", "SYSTEM_ACCESSTOKEN"}, nil),
				Description: "The bearer token for the request to the OIDC provider. For use when authenticating as a Service Principal using OpenID Connect.",
			},

			"oidc_request_url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"ARM_OIDC_REQUEST_URL", "ACTIONS_ID_TOKEN_REQUEST_URL", "SYSTEM_OIDCREQUESTURI"}, nil),
				Description: "The URL for the OIDC provider from which to request an ID token. For use when authenticating as a Service Principal using OpenID Connect.",
			},

			"oidc_token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_OIDC_TOKEN", ""),
				Description: "The OIDC ID token for use when authenticating as a Service Principal using OpenID Connect.",
			},

			"oidc_token_file_path": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_OIDC_TOKEN_FILE_PATH", ""),
				Description: "The path to a file containing an OIDC ID token for use when authenticating as a Service Principal using OpenID Connect.",
			},

			// Managed Identity specific fields
			"use_msi": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Allow Managed Identity to be used for Authentication.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_USE_MSI", false),
			},

			"msi_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path to a custom endpoint for Managed Identity - in most circumstances this should be detected automatically.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_MSI_ENDPOINT", ""),
			},

			// Azure CLI specific fields
			"use_cli": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_USE_CLI", true),
				Description: "Allow Azure CLI to be used for Authentication.",
			},

			// Azure AKS Workload Identity fields
			"use_aks_workload_identity": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_USE_AKS_WORKLOAD_IDENTITY", false),
				Description: "Allow Azure AKS Workload Identity to be used for Authentication.",
			},

			// Feature Flags
			"use_azuread_auth": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether to use Azure Active Directory authentication to access the Storage Data Plane APIs.",
				DefaultFunc: schema.EnvDefaultFunc("ARM_USE_AZUREAD", false),
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
	apiClient     *Client
	containerName string
	keyName       string
	accountName   string
	snapshot      bool
}

type BackendConfig struct {
	AuthConfig               *auth.Credentials
	SubscriptionID           string
	ResourceGroupName        string
	StorageAccountName       string
	AccessKey                string
	SasToken                 string
	UseAzureADAuthentication bool
}

func (b *Backend) configure(ctx context.Context) error {
	// This is to make the go-azure-sdk/sdk/client Client happy.
	if _, ok := ctx.Deadline(); !ok {
		ctx, _ = context.WithTimeout(ctx, 5*time.Minute)
	}

	// Grab the resource data
	data := schema.FromContextBackendConfig(ctx)
	b.containerName = data.Get("container_name").(string)
	b.accountName = data.Get("storage_account_name").(string)
	b.keyName = data.Get("key").(string)
	b.snapshot = data.Get("snapshot").(bool)

	var clientCertificateData []byte
	if encodedCert := data.Get("client_certificate").(string); encodedCert != "" {
		var err error
		clientCertificateData, err = decodeCertificate(encodedCert)
		if err != nil {
			return err
		}
	}

	oidcToken, err := getOidcToken(data)
	if err != nil {
		return err
	}

	clientSecret, err := getClientSecret(data)
	if err != nil {
		return err
	}

	clientId, err := getClientId(data)
	if err != nil {
		return err
	}

	tenantId, err := getTenantId(data)
	if err != nil {
		return err
	}

	var (
		env *environments.Environment

		envName      = data.Get("environment").(string)
		metadataHost = data.Get("metadata_host").(string)
	)

	if metadataHost != "" {
		logEntry("[DEBUG] Configuring cloud environment from Metadata Service at %q", metadataHost)
		if env, err = environments.FromEndpoint(ctx, fmt.Sprintf("https://%s", metadataHost)); err != nil {
			return err
		}
	} else {
		logEntry("[DEBUG] Configuring built-in cloud environment by name: %q", envName)
		if env, err = environments.FromName(envName); err != nil {
			return err
		}
	}

	var (
		enableAzureCli        = data.Get("use_cli").(bool)
		enableManagedIdentity = data.Get("use_msi").(bool)
		enableOidc            = data.Get("use_oidc").(bool) || data.Get("use_aks_workload_identity").(bool)
	)

	authConfig := &auth.Credentials{
		Environment: *env,
		ClientID:    *clientId,
		TenantID:    *tenantId,

		ClientCertificateData:     clientCertificateData,
		ClientCertificatePath:     data.Get("client_certificate_path").(string),
		ClientCertificatePassword: data.Get("client_certificate_password").(string),
		ClientSecret:              *clientSecret,

		OIDCAssertionToken:             *oidcToken,
		OIDCTokenRequestURL:            data.Get("oidc_request_url").(string),
		OIDCTokenRequestToken:          data.Get("oidc_request_token").(string),
		ADOPipelineServiceConnectionID: data.Get("ado_pipeline_service_connection_id").(string),

		CustomManagedIdentityEndpoint: data.Get("msi_endpoint").(string),

		EnableAuthenticatingUsingClientCertificate: true,
		EnableAuthenticatingUsingClientSecret:      true,
		EnableAuthenticatingUsingAzureCLI:          enableAzureCli,
		EnableAuthenticatingUsingManagedIdentity:   enableManagedIdentity,
		EnableAuthenticationUsingOIDC:              enableOidc,
		EnableAuthenticationUsingGitHubOIDC:        enableOidc,
		EnableAuthenticationUsingADOPipelineOIDC:   enableOidc,
	}

	backendConfig := BackendConfig{
		AuthConfig:               authConfig,
		SubscriptionID:           data.Get("subscription_id").(string),
		ResourceGroupName:        data.Get("resource_group_name").(string),
		StorageAccountName:       data.Get("storage_account_name").(string),
		AccessKey:                data.Get("access_key").(string),
		SasToken:                 data.Get("sas_token").(string),
		UseAzureADAuthentication: data.Get("use_azuread_auth").(bool),
	}

	propertiesNeededToLookupAccessKeySpecified := backendConfig.AccessKey == "" && backendConfig.SasToken == "" && backendConfig.ResourceGroupName == ""
	if propertiesNeededToLookupAccessKeySpecified && !backendConfig.UseAzureADAuthentication {
		return fmt.Errorf("either an Access Key / SAS Token or the Resource Group for the Storage Account must be specified - or Azure AD Authentication must be enabled")
	}

	client, err := buildClient(ctx, backendConfig)
	if err != nil {
		return err
	}

	b.apiClient = client
	return nil
}
