// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/storage/2023-01-01/storageaccounts"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	"github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/hashicorp/terraform/internal/httpclient"
	"github.com/hashicorp/terraform/version"
	"github.com/jackofallops/giovanni/storage/2023-11-03/blob/blobs"
	"github.com/jackofallops/giovanni/storage/2023-11-03/blob/containers"
)

type Client struct {
	environment        environments.Environment
	storageAccountName string

	// Storage ARM client is used for looking up the blob endpoint, or/and listing access key (if not specified).
	storageAccountsClient *storageaccounts.StorageAccountsClient
	// This is only non-nil if the config has specified to lookup the blob endpoint
	accountDetail *AccountDetails

	// Caching
	containersClient *containers.Client
	blobsClient      *blobs.Client

	// Only one of them shall be specified
	accessKey          string
	sasToken           string
	azureAdStorageAuth auth.Authorizer
}

func buildClient(ctx context.Context, config BackendConfig) (*Client, error) {
	client := Client{
		environment:        config.AuthConfig.Environment,
		storageAccountName: config.StorageAccountName,
	}

	var armAuthRequired bool
	switch {
	case config.AccessKey != "":
		client.accessKey = config.AccessKey
	case config.SasToken != "":
		sasToken := config.SasToken
		if strings.TrimSpace(sasToken) == "" {
			return nil, fmt.Errorf("sasToken cannot be empty")
		}
		client.sasToken = strings.TrimPrefix(sasToken, "?")
	case config.UseAzureADAuthentication:
		var err error
		client.azureAdStorageAuth, err = auth.NewAuthorizerFromCredentials(ctx, *config.AuthConfig, config.AuthConfig.Environment.Storage)
		if err != nil {
			return nil, fmt.Errorf("unable to build authorizer for Storage API: %+v", err)
		}
	default:
		// AAD authentication (ARM scope) is required only when no auth method is specified, which falls back to listing the access key via ARM API.
		armAuthRequired = true
	}

	// If `config.LookupBlobEndpoint` is true, we need to authenticate with ARM to lookup the blob endpoint
	if config.LookupBlobEndpoint {
		armAuthRequired = true
	}

	if armAuthRequired {
		resourceManagerAuth, err := auth.NewAuthorizerFromCredentials(ctx, *config.AuthConfig, config.AuthConfig.Environment.ResourceManager)
		if err != nil {
			return nil, fmt.Errorf("unable to build authorizer for Resource Manager API: %+v", err)
		}

		// When using Azure CLI to auth, the user can leave the "subscription_id" unspecified. In this case the subscription id is inferred from
		// the Azure CLI default subscription.
		if config.SubscriptionID == "" {
			if cachedAuth, ok := resourceManagerAuth.(*auth.CachedAuthorizer); ok {
				if cliAuth, ok := cachedAuth.Source.(*auth.AzureCliAuthorizer); ok && cliAuth.DefaultSubscriptionID != "" {
					config.SubscriptionID = cliAuth.DefaultSubscriptionID
				}
			}
		}
		if config.SubscriptionID == "" {
			return nil, fmt.Errorf("subscription id not specified")
		}

		// Setup the SA client.
		client.storageAccountsClient, err = storageaccounts.NewStorageAccountsClientWithBaseURI(config.AuthConfig.Environment.ResourceManager)
		if err != nil {
			return nil, fmt.Errorf("building Storage Accounts client: %+v", err)
		}
		client.configureClient(client.storageAccountsClient.Client, resourceManagerAuth)

		// Populating the storage account detail
		storageAccountId := commonids.NewStorageAccountID(config.SubscriptionID, config.ResourceGroupName, client.storageAccountName)
		resp, err := client.storageAccountsClient.GetProperties(ctx, storageAccountId, storageaccounts.DefaultGetPropertiesOperationOptions())
		if err != nil {
			return nil, fmt.Errorf("retrieving %s: %+v", storageAccountId, err)
		}
		if resp.Model == nil {
			return nil, fmt.Errorf("retrieving %s: model was nil", storageAccountId)
		}
		client.accountDetail, err = populateAccountDetails(storageAccountId, *resp.Model)
		if err != nil {
			return nil, fmt.Errorf("populating details for %s: %+v", storageAccountId, err)
		}
	}

	return &client, nil
}

func (c *Client) getBlobClient(ctx context.Context) (bc *blobs.Client, err error) {
	if c.blobsClient != nil {
		return c.blobsClient, nil
	}

	defer func() {
		if err == nil {
			c.blobsClient = bc
		}
	}()

	var baseUri string
	if c.accountDetail != nil {
		// Use the actual blob endpoint if available
		pBaseUri, err := c.accountDetail.DataPlaneEndpoint(EndpointTypeBlob)
		if err != nil {
			return nil, err
		}
		baseUri = *pBaseUri
	} else {
		baseUri, err = naiveStorageAccountBlobBaseURL(c.environment, c.storageAccountName)
		if err != nil {
			return nil, err
		}
	}

	blobsClient, err := blobs.NewWithBaseUri(baseUri)
	if err != nil {
		return nil, fmt.Errorf("new blob client: %v", err)
	}

	switch {
	case c.sasToken != "":
		log.Printf("[DEBUG] Building the Blob Client from a SAS Token")
		c.configureClient(blobsClient.Client, nil)
		blobsClient.Client.AppendRequestMiddleware(func(r *http.Request) (*http.Request, error) {
			if r.URL.RawQuery == "" {
				r.URL.RawQuery = c.sasToken
			} else if !strings.Contains(r.URL.RawQuery, c.sasToken) {
				r.URL.RawQuery = fmt.Sprintf("%s&%s", r.URL.RawQuery, c.sasToken)
			}
			return r, nil
		})
		return blobsClient, nil

	case c.accessKey != "":
		log.Printf("[DEBUG] Building the Blob Client from an Access Key")
		authorizer, err := auth.NewSharedKeyAuthorizer(c.storageAccountName, c.accessKey, auth.SharedKey)
		if err != nil {
			return nil, fmt.Errorf("new shared key authorizer: %v", err)
		}
		c.configureClient(blobsClient.Client, authorizer)
		return blobsClient, nil

	case c.azureAdStorageAuth != nil:
		log.Printf("[DEBUG] Building the Blob Client from AAD auth")
		c.configureClient(blobsClient.Client, c.azureAdStorageAuth)
		return blobsClient, nil

	default:
		// Neither shared access key, sas token, or AAD Auth were specified so we have to call the management plane API to get the key.
		log.Printf("[DEBUG] Building the Blob Client from an Access Key (key is listed using client credentials)")
		key, err := c.accountDetail.AccountKey(ctx, c.storageAccountsClient)
		if err != nil {
			return nil, fmt.Errorf("retrieving key for Storage Account %q: %s", c.storageAccountName, err)
		}
		authorizer, err := auth.NewSharedKeyAuthorizer(c.storageAccountName, *key, auth.SharedKey)
		if err != nil {
			return nil, fmt.Errorf("new shared key authorizer: %v", err)
		}
		c.configureClient(blobsClient.Client, authorizer)
		return blobsClient, nil
	}
}

func (c *Client) getContainersClient(ctx context.Context) (cc *containers.Client, err error) {
	if c.containersClient != nil {
		return c.containersClient, nil
	}

	defer func() {
		if err == nil {
			c.containersClient = cc
		}
	}()

	var baseUri string
	if c.accountDetail != nil {
		// Use the actual blob endpoint if available
		pBaseUri, err := c.accountDetail.DataPlaneEndpoint(EndpointTypeBlob)
		if err != nil {
			return nil, err
		}
		baseUri = *pBaseUri
	} else {
		baseUri, err = naiveStorageAccountBlobBaseURL(c.environment, c.storageAccountName)
		if err != nil {
			return nil, err
		}
	}

	containersClient, err := containers.NewWithBaseUri(baseUri)
	if err != nil {
		return nil, fmt.Errorf("new container client: %v", err)
	}

	switch {
	case c.sasToken != "":
		log.Printf("[DEBUG] Building the Container Client from a SAS Token")
		c.configureClient(containersClient.Client, nil)
		containersClient.Client.AppendRequestMiddleware(func(r *http.Request) (*http.Request, error) {
			if r.URL.RawQuery == "" {
				r.URL.RawQuery = c.sasToken
			} else if !strings.Contains(r.URL.RawQuery, c.sasToken) {
				r.URL.RawQuery = fmt.Sprintf("%s&%s", r.URL.RawQuery, c.sasToken)
			}
			return r, nil
		})
		return containersClient, nil

	case c.accessKey != "":
		log.Printf("[DEBUG] Building the Container Client from an Access Key")
		authorizer, err := auth.NewSharedKeyAuthorizer(c.storageAccountName, c.accessKey, auth.SharedKey)
		if err != nil {
			return nil, fmt.Errorf("new shared key authorizer: %v", err)
		}
		c.configureClient(containersClient.Client, authorizer)
		return containersClient, nil

	case c.azureAdStorageAuth != nil:
		log.Printf("[DEBUG] Building the Container Client from AAD auth")
		c.configureClient(containersClient.Client, c.azureAdStorageAuth)
		return containersClient, nil

	default:
		// Neither shared access key, sas token, or AAD Auth were specified so we have to call the management plane API to get the key.
		log.Printf("[DEBUG] Building the Container Client from an Access Key (key is listed using user credentials)")
		key, err := c.accountDetail.AccountKey(ctx, c.storageAccountsClient)
		if err != nil {
			return nil, fmt.Errorf("retrieving key for Storage Account %q: %s", c.storageAccountName, err)
		}
		authorizer, err := auth.NewSharedKeyAuthorizer(c.storageAccountName, *key, auth.SharedKey)
		if err != nil {
			return nil, fmt.Errorf("new shared key authorizer: %v", err)
		}
		c.configureClient(containersClient.Client, authorizer)
		return containersClient, nil
	}
}

func (c *Client) configureClient(client client.BaseClient, authorizer auth.Authorizer) {
	client.SetAuthorizer(authorizer)
	client.SetUserAgent(buildUserAgent(client.GetUserAgent()))
}

func buildUserAgent(userAgent string) string {
	userAgent = strings.TrimSpace(fmt.Sprintf("%s %s", userAgent, httpclient.TerraformUserAgent(version.Version)))

	// append the CloudShell version to the user agent if it exists
	if azureAgent := os.Getenv("AZURE_HTTP_USER_AGENT"); azureAgent != "" {
		userAgent = fmt.Sprintf("%s %s", userAgent, azureAgent)
	}

	return userAgent
}
