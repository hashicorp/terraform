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
	// These Clients are only initialized if an Access Key isn't provided
	storageAccountsClient *storageaccounts.StorageAccountsClient

	// Caching
	containersClient *containers.Client
	blobsClient      *blobs.Client

	environment        environments.Environment
	storageAccountName string

	accountDetail *AccountDetails

	accessKey string
	sasToken  string
	// azureAdStorageAuth is only here if we're using AzureAD Authentication but is an Authorizer for Storage
	azureAdStorageAuth auth.Authorizer
}

func buildClient(ctx context.Context, config BackendConfig) (*Client, error) {
	client := Client{
		environment:        config.AuthConfig.Environment,
		storageAccountName: config.StorageAccountName,
	}

	// if we have an Access Key - we don't need the other clients
	if config.AccessKey != "" {
		client.accessKey = config.AccessKey
		return &client, nil
	}

	// likewise with a SAS token
	if config.SasToken != "" {
		sasToken := config.SasToken
		if strings.TrimSpace(sasToken) == "" {
			return nil, fmt.Errorf("sasToken cannot be empty")
		}
		client.sasToken = strings.TrimPrefix(sasToken, "?")

		return &client, nil
	}

	if config.UseAzureADAuthentication {
		var err error
		client.azureAdStorageAuth, err = auth.NewAuthorizerFromCredentials(ctx, *config.AuthConfig, config.AuthConfig.Environment.Storage)
		if err != nil {
			return nil, fmt.Errorf("unable to build authorizer for Storage API: %+v", err)
		}
	}

	resourceManagerAuth, err := auth.NewAuthorizerFromCredentials(ctx, *config.AuthConfig, config.AuthConfig.Environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("unable to build authorizer for Resource Manager API: %+v", err)
	}

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

	if c.sasToken != "" {
		log.Printf("[DEBUG] Building the Blob Client from a SAS Token")
		baseURL, err := naiveStorageAccountBlobBaseURL(c.environment, c.storageAccountName)
		if err != nil {
			return nil, fmt.Errorf("build storage account blob base URL: %v", err)
		}
		blobsClient, err := blobs.NewWithBaseUri(baseURL)
		if err != nil {
			return nil, fmt.Errorf("new blob client: %v", err)
		}
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
	}

	if c.accessKey != "" {
		log.Printf("[DEBUG] Building the Blob Client from an Access Key")
		baseURL, err := naiveStorageAccountBlobBaseURL(c.environment, c.storageAccountName)
		if err != nil {
			return nil, fmt.Errorf("build storage account blob base URL: %v", err)
		}
		blobsClient, err := blobs.NewWithBaseUri(baseURL)
		if err != nil {
			return nil, fmt.Errorf("new blob client: %v", err)
		}
		c.configureClient(blobsClient.Client, nil)

		authorizer, err := auth.NewSharedKeyAuthorizer(c.storageAccountName, c.accessKey, auth.SharedKey)
		if err != nil {
			return nil, fmt.Errorf("new shared key authorizer: %v", err)
		}
		c.configureClient(blobsClient.Client, authorizer)

		return blobsClient, nil
	}

	// Neither shared access key nor sas token specified, then we have the storage account details populated.
	// This detail can be used to get the "most" correct blob endpoint comparing to the naive construction.
	baseUri, err := c.accountDetail.DataPlaneEndpoint(EndpointTypeBlob)
	if err != nil {
		return nil, err
	}
	blobsClient, err := blobs.NewWithBaseUri(*baseUri)
	if err != nil {
		return nil, fmt.Errorf("new blob client: %v", err)
	}

	if c.azureAdStorageAuth != nil {
		log.Printf("[DEBUG] Building the Blob Client from AAD auth")
		c.configureClient(blobsClient.Client, c.azureAdStorageAuth)
		return blobsClient, nil
	}

	log.Printf("[DEBUG] Building the Blob Client from an Access Token (using user credentials)")
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

func (c *Client) getContainersClient(ctx context.Context) (cc *containers.Client, err error) {
	if c.containersClient != nil {
		return c.containersClient, nil
	}

	defer func() {
		if err == nil {
			c.containersClient = cc
		}
	}()

	if c.sasToken != "" {
		log.Printf("[DEBUG] Building the Container Client from a SAS Token")
		baseURL, err := naiveStorageAccountBlobBaseURL(c.environment, c.storageAccountName)
		if err != nil {
			return nil, fmt.Errorf("build storage account blob base URL: %v", err)
		}
		containersClient, err := containers.NewWithBaseUri(baseURL)
		if err != nil {
			return nil, fmt.Errorf("new container client: %v", err)
		}
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
	}

	if c.accessKey != "" {
		log.Printf("[DEBUG] Building the Container Client from an Access Key")
		baseURL, err := naiveStorageAccountBlobBaseURL(c.environment, c.storageAccountName)
		if err != nil {
			return nil, fmt.Errorf("build storage account blob base URL: %v", err)
		}
		containersClient, err := containers.NewWithBaseUri(baseURL)
		if err != nil {
			return nil, fmt.Errorf("new container client: %v", err)
		}
		c.configureClient(containersClient.Client, nil)

		authorizer, err := auth.NewSharedKeyAuthorizer(c.storageAccountName, c.accessKey, auth.SharedKey)
		if err != nil {
			return nil, fmt.Errorf("new shared key authorizer: %v", err)
		}
		c.configureClient(containersClient.Client, authorizer)

		return containersClient, nil
	}

	// Neither shared access key nor sas token specified, then we have the storage account details populated.
	// This detail can be used to get the "most" correct blob endpoint comparing to the naive construction.
	baseUri, err := c.accountDetail.DataPlaneEndpoint(EndpointTypeBlob)
	if err != nil {
		return nil, err
	}
	containersClient, err := containers.NewWithBaseUri(*baseUri)
	if err != nil {
		return nil, fmt.Errorf("new container client: %v", err)
	}

	if c.azureAdStorageAuth != nil {
		log.Printf("[DEBUG] Building the Container Client from AAD auth")
		c.configureClient(containersClient.Client, c.azureAdStorageAuth)
		return containersClient, nil
	}

	log.Printf("[DEBUG] Building the Container Client from an Access Token (using user credentials)")
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
