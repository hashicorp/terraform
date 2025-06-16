// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendbase"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// New creates a new backend for Azure remote state.
func New() backend.Backend {
	return &Backend{
		Base: backendbase.Base{
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{

					"subscription_id": {
						Type:        cty.String,
						Optional:    true,
						Description: "The Subscription ID where the Storage Account is located.",
					},
					"resource_group_name": {
						Type:        cty.String,
						Optional:    true,
						Description: "The Resource Group where the Storage Account is located.",
					},
					"storage_account_name": {
						Type:        cty.String,
						Required:    true,
						Description: "The name of the storage account.",
					},
					"container_name": {
						Type:        cty.String,
						Required:    true,
						Description: "The container name to use in the Storage Account.",
					},
					"key": {
						Type:        cty.String,
						Required:    true,
						Description: "The blob key to use in the Storage Container.",
					},
					"lookup_blob_endpoint": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Whether to look up the storage account blob endpoint. This is necessary when the storage account uses the Azure DNS zone endpoint.",
					},
					"snapshot": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Whether to enable automatic blob snapshotting.",
					},
					"environment": {
						Type:        cty.String,
						Optional:    true,
						Description: "The Cloud Environment which should be used. Possible values are public, usgovernment, and china. Defaults to public. Not used and should not be specified when `metadata_host` is specified.",
					},
					"metadata_host": {
						Type:        cty.String,
						Optional:    true,
						Description: "The Hostname which should be used for the Azure Metadata Service.",
					},
					"access_key": {
						Type:        cty.String,
						Optional:    true,
						Description: "The access key to use when authenticating using a Storage Access Key.",
					},
					"sas_token": {
						Type:        cty.String,
						Optional:    true,
						Description: "The SAS Token to use when authenticating using a SAS Token.",
					},
					"tenant_id": {
						Type:        cty.String,
						Optional:    true,
						Description: "The Tenant ID to use when authenticating using Azure Active Directory.",
					},
					"client_id": {
						Type:        cty.String,
						Optional:    true,
						Description: "The Client ID to use when authenticating using Azure Active Directory.",
					},
					"client_id_file_path": {
						Type:        cty.String,
						Optional:    true,
						Description: "The path to a file containing the Client ID which should be used.",
					},
					"endpoint": {
						Type:        cty.String,
						Optional:    true,
						Deprecated:  true,
						Description: "`endpoint` is deprecated in favor of `msi_endpoint`, it will be removed in a future version of Terraform",
					},

					// Client Certificate specific fields
					"client_certificate": {
						Type:        cty.String,
						Optional:    true,
						Description: "Base64 encoded PKCS#12 certificate bundle to use when authenticating as a Service Principal using a Client Certificate",
					},
					"client_certificate_path": {
						Type:        cty.String,
						Optional:    true,
						Description: "The path to the Client Certificate associated with the Service Principal for use when authenticating as a Service Principal using a Client Certificate.",
					},
					"client_certificate_password": {
						Type:        cty.String,
						Optional:    true,
						Description: "The password associated with the Client Certificate. For use when authenticating as a Service Principal using a Client Certificate",
					},

					// Client Secret specific fields
					"client_secret": {
						Type:        cty.String,
						Optional:    true,
						Description: "The Client Secret which should be used. For use When authenticating as a Service Principal using a Client Secret.",
					},
					"client_secret_file_path": {
						Type:        cty.String,
						Optional:    true,
						Description: "The path to a file containing the Client Secret which should be used. For use When authenticating as a Service Principal using a Client Secret.",
					},

					// OIDC specific fields
					"use_oidc": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Allow OpenID Connect to be used for authentication",
					},
					"ado_pipeline_service_connection_id": {
						Type:        cty.String,
						Optional:    true,
						Description: "The Azure DevOps Pipeline Service Connection ID.",
					},
					"oidc_request_token": {
						Type:        cty.String,
						Optional:    true,
						Description: "The bearer token for the request to the OIDC provider. For use when authenticating as a Service Principal using OpenID Connect.",
					},
					"oidc_request_url": {
						Type:        cty.String,
						Optional:    true,
						Description: "The URL for the OIDC provider from which to request an ID token. For use when authenticating as a Service Principal using OpenID Connect.",
					},
					"oidc_token": {
						Type:        cty.String,
						Optional:    true,
						Description: "The OIDC ID token for use when authenticating as a Service Principal using OpenID Connect.",
					},
					"oidc_token_file_path": {
						Type:        cty.String,
						Optional:    true,
						Description: "The path to a file containing an OIDC ID token for use when authenticating as a Service Principal using OpenID Connect.",
					},

					// Managed Identity specific fields
					"use_msi": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Allow Managed Identity to be used for Authentication.",
					},
					"msi_endpoint": {
						Type:        cty.String,
						Optional:    true,
						Description: "The path to a custom endpoint for Managed Identity - in most circumstances this should be detected automatically.",
					},

					// Azure CLI specific fields
					"use_cli": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Allow Azure CLI to be used for Authentication.",
					},

					// Azure AKS Workload Identity fields
					"use_aks_workload_identity": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Allow Azure AKS Workload Identity to be used for Authentication.",
					},

					// Feature Flags
					"use_azuread_auth": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Whether to use Azure Active Directory authentication to access the Storage Data Plane APIs.",
					},
				},
			},
			SDKLikeDefaults: backendbase.SDKLikeDefaults{
				"subscription_id": {
					EnvVars:  []string{"ARM_SUBSCRIPTION_ID"},
					Fallback: "",
				},
				"lookup_blob_endpoint": {
					EnvVars:  []string{"ARM_USE_DNS_ZONE_ENDPOINT"},
					Fallback: "false",
				},
				"snapshot": {
					EnvVars:  []string{"ARM_SNAPSHOT"},
					Fallback: "false",
				},
				"environment": {
					EnvVars:  []string{"ARM_ENVIRONMENT"},
					Fallback: "public",
				},
				"metadata_host": {
					EnvVars:  []string{"ARM_METADATA_HOSTNAME", "ARM_METADATA_HOST"}, // TODO: remove support for `METADATA_HOST` in a future version
					Fallback: "",
				},
				"access_key": {
					EnvVars:  []string{"ARM_ACCESS_KEY"},
					Fallback: "",
				},
				"sas_token": {
					EnvVars:  []string{"ARM_SAS_TOKEN"},
					Fallback: "",
				},
				"tenant_id": {
					EnvVars:  []string{"ARM_TENANT_ID"},
					Fallback: "",
				},
				"client_id": {
					EnvVars:  []string{"ARM_CLIENT_ID"},
					Fallback: "",
				},
				"client_id_file_path": {
					EnvVars: []string{"ARM_CLIENT_ID_FILE_PATH"},
					// no fallback
				},

				// Client Certificate specific fields
				"client_certificate": {
					EnvVars:  []string{"ARM_CLIENT_CERTIFICATE"},
					Fallback: "",
				},
				"client_certificate_path": {
					EnvVars:  []string{"ARM_CLIENT_CERTIFICATE_PATH"},
					Fallback: "",
				},
				"client_certificate_password": {
					EnvVars:  []string{"ARM_CLIENT_CERTIFICATE_PASSWORD"},
					Fallback: "",
				},

				// Client Secret specific fields
				"client_secret": {
					EnvVars:  []string{"ARM_CLIENT_SECRET"},
					Fallback: "",
				},
				"client_secret_file_path": {
					EnvVars: []string{"ARM_CLIENT_SECRET_FILE_PATH"},
					// no fallback
				},

				// OIDC specific fields
				"use_oidc": {
					EnvVars:  []string{"ARM_USE_OIDC"},
					Fallback: "false",
				},
				"ado_pipeline_service_connection_id": {
					EnvVars: []string{"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID", "ARM_OIDC_AZURE_SERVICE_CONNECTION_ID"},
					// no fallback
				},
				"oidc_request_token": {
					EnvVars: []string{"ARM_OIDC_REQUEST_TOKEN", "ACTIONS_ID_TOKEN_REQUEST_TOKEN", "SYSTEM_ACCESSTOKEN"},
					// no fallback
				},
				"oidc_request_url": {
					EnvVars: []string{"ARM_OIDC_REQUEST_URL", "ACTIONS_ID_TOKEN_REQUEST_URL", "SYSTEM_OIDCREQUESTURI"},
					// no fallback
				},
				"oidc_token": {
					EnvVars:  []string{"ARM_OIDC_TOKEN"},
					Fallback: "",
				},
				"oidc_token_file_path": {
					EnvVars:  []string{"ARM_OIDC_TOKEN_FILE_PATH"},
					Fallback: "",
				},

				// Managed Identity specific fields
				"use_msi": {
					EnvVars:  []string{"ARM_USE_MSI"},
					Fallback: "false",
				},
				"msi_endpoint": {
					EnvVars:  []string{"ARM_MSI_ENDPOINT"},
					Fallback: "",
				},

				// Azure CLI specific fields
				"use_cli": {
					EnvVars:  []string{"ARM_USE_CLI"},
					Fallback: "true",
				},

				// Azure AKS Workload Identity fields
				"use_aks_workload_identity": {
					EnvVars:  []string{"ARM_USE_AKS_WORKLOAD_IDENTITY"},
					Fallback: "false",
				},

				// Feature Flags
				"use_azuread_auth": {
					EnvVars:  []string{"ARM_USE_AZUREAD"},
					Fallback: "false",
				},
			},
		},
	}
}

type Backend struct {
	backendbase.Base

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
	LookupBlobEndpoint       bool
	AccessKey                string
	SasToken                 string
	UseAzureADAuthentication bool
}

func (b *Backend) Configure(configVal cty.Value) tfdiags.Diagnostics {
	// This is to make the go-azure-sdk/sdk/client Client happy.
	ctx := context.Background()
	if _, ok := ctx.Deadline(); !ok {
		ctx, _ = context.WithTimeout(ctx, 5*time.Minute)
	}

	// This backend was originally written against the legacy plugin SDK, so
	// we use some shimming here to keep things working mostly the same.
	data := backendbase.NewSDKLikeData(configVal)

	b.containerName = data.String("container_name")
	b.accountName = data.String("storage_account_name")
	b.keyName = data.String("key")
	b.snapshot = data.Bool("snapshot")

	var clientCertificateData []byte
	if encodedCert := data.String("client_certificate"); encodedCert != "" {
		var err error
		clientCertificateData, err = decodeCertificate(encodedCert)
		if err != nil {
			return backendbase.ErrorAsDiagnostics(err)
		}
	}

	oidcToken, err := getOidcToken(&data)
	if err != nil {
		return backendbase.ErrorAsDiagnostics(err)
	}

	clientSecret, err := getClientSecret(&data)
	if err != nil {
		return backendbase.ErrorAsDiagnostics(err)
	}

	clientId, err := getClientId(&data)
	if err != nil {
		return backendbase.ErrorAsDiagnostics(err)
	}

	tenantId, err := getTenantId(&data)
	if err != nil {
		return backendbase.ErrorAsDiagnostics(err)
	}

	var (
		env *environments.Environment

		envName      = data.String("environment")
		metadataHost = data.String("metadata_host")
	)

	if metadataHost != "" {
		logEntry("[DEBUG] Configuring cloud environment from Metadata Service at %q", metadataHost)
		if env, err = environments.FromEndpoint(ctx, fmt.Sprintf("https://%s", metadataHost)); err != nil {
			return backendbase.ErrorAsDiagnostics(err)
		}
	} else {
		logEntry("[DEBUG] Configuring built-in cloud environment by name: %q", envName)
		if env, err = environments.FromName(envName); err != nil {
			return backendbase.ErrorAsDiagnostics(err)
		}
	}

	var (
		enableAzureCli        = data.Bool("use_cli")
		enableManagedIdentity = data.Bool("use_msi")
		enableOidc            = data.Bool("use_oidc") || data.Bool("use_aks_workload_identity")
	)

	authConfig := &auth.Credentials{
		Environment: *env,
		ClientID:    *clientId,
		TenantID:    *tenantId,

		ClientCertificateData:     clientCertificateData,
		ClientCertificatePath:     data.String("client_certificate_path"),
		ClientCertificatePassword: data.String("client_certificate_password"),
		ClientSecret:              *clientSecret,

		OIDCAssertionToken:             *oidcToken,
		OIDCTokenRequestURL:            data.String("oidc_request_url"),
		OIDCTokenRequestToken:          data.String("oidc_request_token"),
		ADOPipelineServiceConnectionID: data.String("ado_pipeline_service_connection_id"),

		CustomManagedIdentityEndpoint: data.String("msi_endpoint"),

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
		SubscriptionID:           data.String("subscription_id"),
		ResourceGroupName:        data.String("resource_group_name"),
		StorageAccountName:       data.String("storage_account_name"),
		LookupBlobEndpoint:       data.Bool("lookup_blob_endpoint"),
		AccessKey:                data.String("access_key"),
		SasToken:                 data.String("sas_token"),
		UseAzureADAuthentication: data.Bool("use_azuread_auth"),
	}

	needToLookupAccessKey := backendConfig.AccessKey == "" && backendConfig.SasToken == "" && !backendConfig.UseAzureADAuthentication
	if backendConfig.ResourceGroupName == "" {
		if needToLookupAccessKey {
			backendbase.ErrorAsDiagnostics(fmt.Errorf("One of `access_key`, `sas_token`, `use_azuread_auth` and `resource_group_name` must be specifieid"))
		}
		if backendConfig.LookupBlobEndpoint {
			backendbase.ErrorAsDiagnostics(fmt.Errorf("`resource_group_name` is required when `lookup_blob_endpoint` is set"))
		}
	}

	client, err := buildClient(ctx, backendConfig)
	if err != nil {
		return backendbase.ErrorAsDiagnostics(err)
	}

	b.apiClient = client
	return nil
}
