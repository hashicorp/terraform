// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"context"
	"fmt"

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
					"storage_account_name": {
						Type:        cty.String,
						Required:    true,
						Description: "The name of the storage account",
					},

					"container_name": {
						Type:        cty.String,
						Required:    true,
						Description: "The container name",
					},

					"key": {
						Type:        cty.String,
						Required:    true,
						Description: "The blob key.",
					},

					"metadata_host": {
						Type:        cty.String,
						Optional:    true,
						Description: "The Metadata URL which will be used to obtain the Cloud Environment.",
					},

					"environment": {
						Type:        cty.String,
						Optional:    true,
						Description: "The Azure cloud environment.",
					},

					"access_key": {
						Type:        cty.String,
						Optional:    true,
						Description: "The access key.",
					},

					"sas_token": {
						Type:        cty.String,
						Optional:    true,
						Description: "A SAS Token used to interact with the Blob Storage Account.",
					},

					"snapshot": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Enable/Disable automatic blob snapshotting",
					},

					"resource_group_name": {
						Type:        cty.String,
						Optional:    true,
						Description: "The resource group name.",
					},

					"client_id": {
						Type:        cty.String,
						Optional:    true,
						Description: "The Client ID.",
					},

					"endpoint": {
						Type:        cty.String,
						Optional:    true,
						Description: "A custom Endpoint used to access the Azure Resource Manager API's.",
					},

					"subscription_id": {
						Type:        cty.String,
						Optional:    true,
						Description: "The Subscription ID.",
					},

					"tenant_id": {
						Type:        cty.String,
						Optional:    true,
						Description: "The Tenant ID.",
					},

					// Service Principal (Client Certificate) specific
					"client_certificate_password": {
						Type:        cty.String,
						Optional:    true,
						Description: "The password associated with the Client Certificate specified in `client_certificate_path`",
					},
					"client_certificate_path": {
						Type:        cty.String,
						Optional:    true,
						Description: "The path to the PFX file used as the Client Certificate when authenticating as a Service Principal",
					},

					// Service Principal (Client Secret) specific
					"client_secret": {
						Type:        cty.String,
						Optional:    true,
						Description: "The Client Secret.",
					},

					// Managed Service Identity specific
					"use_msi": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Should Managed Service Identity be used?",
					},
					"msi_endpoint": {
						Type:        cty.String,
						Optional:    true,
						Description: "The Managed Service Identity Endpoint.",
					},

					// OIDC auth specific fields
					"use_oidc": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Allow OIDC to be used for authentication",
					},
					"oidc_token": {
						Type:        cty.String,
						Optional:    true,
						Description: "A generic JWT token that can be used for OIDC authentication. Should not be used in conjunction with `oidc_request_token`.",
					},
					"oidc_token_file_path": {
						Type:        cty.String,
						Optional:    true,
						Description: "Path to file containing a generic JWT token that can be used for OIDC authentication. Should not be used in conjunction with `oidc_request_token`.",
					},
					"oidc_request_url": {
						Type:        cty.String,
						Optional:    true,
						Description: "The URL of the OIDC provider from which to request an ID token. Needs to be used in conjunction with `oidc_request_token`. This is meant to be used for Github Actions.",
					},
					"oidc_request_token": {
						Type:        cty.String,
						Optional:    true,
						Description: "The bearer token to use for the request to the OIDC providers `oidc_request_url` URL to fetch an ID token. Needs to be used in conjunction with `oidc_request_url`. This is meant to be used for Github Actions.",
					},

					// Feature Flags
					"use_azuread_auth": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Should Terraform use AzureAD Authentication to access the Blob?",
					},
				},
			},
			SDKLikeDefaults: backendbase.SDKLikeDefaults{
				"metadata_host": {
					EnvVars: []string{"ARM_METADATA_HOST"},
				},
				"environment": {
					EnvVars:  []string{"ARM_ENVIRONMENT"},
					Fallback: "public",
				},
				"acccess_key": {
					EnvVars: []string{"ARM_ACCESS_KEY"},
				},
				"sas_token": {
					EnvVars: []string{"ARM_SAS_TOKEN"},
				},
				"snapshot": {
					EnvVars:  []string{"ARM_SNAPSHOT"},
					Fallback: "false",
				},
				"client_id": {
					EnvVars: []string{"ARM_CLIENT_ID"},
				},
				"endpoint": {
					EnvVars: []string{"ARM_ENDPOINT"},
				},
				"subscription_id": {
					EnvVars: []string{"ARM_SUBSCRIPTION_ID"},
				},
				"tenant_id": {
					EnvVars: []string{"ARM_TENANT_ID"},
				},
				"client_certificate_password": {
					EnvVars: []string{"ARM_CLIENT_CERTIFICATE_PASSWORD"},
				},
				"client_certificate_path": {
					EnvVars: []string{"ARM_CLIENT_CERTIFICATE_PATH"},
				},
				"client_secret": {
					EnvVars: []string{"ARM_CLIENT_SECRET"},
				},
				"use_msi": {
					EnvVars:  []string{"ARM_USE_MSI"},
					Fallback: "false",
				},
				"msi_endpoint": {
					EnvVars: []string{"ARM_MSI_ENDPOINT"},
				},
				"use_oidc": {
					EnvVars:  []string{"ARM_USE_OIDC"},
					Fallback: "false",
				},
				"oidc_token": {
					EnvVars: []string{"ARM_OIDC_TOKEN"},
				},
				"oidc_token_file_path": {
					EnvVars: []string{"ARM_OIDC_TOKEN_FILE_PATH"},
				},
				"oidc_request_url": {
					EnvVars: []string{"ARM_OIDC_REQUEST_URL", "ACTIONS_ID_TOKEN_REQUEST_URL"},
				},
				"oidc_request_token": {
					EnvVars: []string{"ARM_OIDC_REQUEST_TOKEN", "ACTIONS_ID_TOKEN_REQUEST_TOKEN"},
				},
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
	OIDCToken                     string
	OIDCTokenFilePath             string
	OIDCRequestURL                string
	OIDCRequestToken              string
	ResourceGroupName             string
	SasToken                      string
	SubscriptionID                string
	TenantID                      string
	UseMsi                        bool
	UseOIDC                       bool
	UseAzureADAuthentication      bool
}

func (b *Backend) Configure(configVal cty.Value) tfdiags.Diagnostics {
	if b.containerName != "" {
		return nil
	}

	// This backend was originally written against the legacy plugin SDK, so
	// we use some shimming here to keep things working mostly the same.
	data := backendbase.NewSDKLikeData(configVal)

	b.containerName = data.String("container_name")
	b.accountName = data.String("storage_account_name")
	b.keyName = data.String("key")
	b.snapshot = data.Bool("snapshot")

	config := BackendConfig{
		AccessKey:                     data.String("access_key"),
		ClientID:                      data.String("client_id"),
		ClientCertificatePassword:     data.String("client_certificate_password"),
		ClientCertificatePath:         data.String("client_certificate_path"),
		ClientSecret:                  data.String("client_secret"),
		CustomResourceManagerEndpoint: data.String("endpoint"),
		MetadataHost:                  data.String("metadata_host"),
		Environment:                   data.String("environment"),
		MsiEndpoint:                   data.String("msi_endpoint"),
		OIDCToken:                     data.String("oidc_token"),
		OIDCTokenFilePath:             data.String("oidc_token_file_path"),
		OIDCRequestURL:                data.String("oidc_request_url"),
		OIDCRequestToken:              data.String("oidc_request_token"),
		ResourceGroupName:             data.String("resource_group_name"),
		SasToken:                      data.String("sas_token"),
		StorageAccountName:            data.String("storage_account_name"),
		SubscriptionID:                data.String("subscription_id"),
		TenantID:                      data.String("tenant_id"),
		UseMsi:                        data.Bool("use_msi"),
		UseOIDC:                       data.Bool("use_oidc"),
		UseAzureADAuthentication:      data.Bool("use_azuread_auth"),
	}

	armClient, err := buildArmClient(context.TODO(), config)
	if err != nil {
		return backendbase.ErrorAsDiagnostics(err)
	}

	thingsNeededToLookupAccessKeySpecified := config.AccessKey == "" && config.SasToken == "" && config.ResourceGroupName == ""
	if thingsNeededToLookupAccessKeySpecified && !config.UseAzureADAuthentication {
		return backendbase.ErrorAsDiagnostics(
			fmt.Errorf("Either an Access Key / SAS Token or the Resource Group for the Storage Account must be specified - or Azure AD Authentication must be enabled"),
		)
	}

	b.armClient = armClient
	return nil
}
