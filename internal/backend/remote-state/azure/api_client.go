// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2024-03-01/resourcegroups"
	"github.com/hashicorp/go-azure-sdk/resource-manager/storage/2023-01-01/storageaccounts"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	"github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/hashicorp/terraform/internal/httpclient"
	"github.com/hashicorp/terraform/version"
	"github.com/tombuildsstuff/giovanni/storage/2023-11-03/blob/blobs"
	"github.com/tombuildsstuff/giovanni/storage/2023-11-03/blob/containers"
)

type Client struct {
	// These Clients are only initialized if an Access Key isn't provided
	resourceGroupsClient  *resourcegroups.ResourceGroupsClient
	storageAccountsClient *storageaccounts.StorageAccountsClient
	containersClient      *containers.Client
	blobsClient           *blobs.Client

	// azureAdStorageAuth is only here if we're using AzureAD Authentication but is an Authorizer for Storage
	azureAdStorageAuth *autorest.Authorizer

	accessKey          string
	environment        environments.Environment
	resourceGroupName  string
	storageAccountName string
	sasToken           string
}

func buildClient(ctx context.Context, config BackendConfig) (*Client, error) {
	resourceManagerAuth, err := auth.NewAuthorizerFromCredentials(ctx, *config.AuthConfig, config.AuthConfig.Environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("unable to build authorizer for Resource Manager API: %+v", err)
	}

	client := Client{
		environment:        config.AuthConfig.Environment,
		resourceGroupName:  config.ResourceGroupName,
		storageAccountName: config.StorageAccountName,
	}

	// if we have an Access Key - we don't need the other clients
	if config.AccessKey != "" {
		client.accessKey = config.AccessKey
		return &client, nil
	}

	// likewise with a SAS token
	if config.SasToken != "" {
		client.sasToken = config.SasToken
		return &client, nil
	}

	client.resourceGroupsClient, err = resourcegroups.NewResourceGroupsClientWithBaseURI(config.AuthConfig.Environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building Resource Groups client: %+v", err)
	}
	client.configureClient(client.resourceGroupsClient.Client, resourceManagerAuth)

	client.storageAccountsClient, err = storageaccounts.NewStorageAccountsClientWithBaseURI(config.AuthConfig.Environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building Storage Accounts client: %+v", err)
	}
	client.configureClient(client.storageAccountsClient.Client, resourceManagerAuth)

	return &client, nil
}

func (c *Client) getBlobClient(ctx context.Context) (*blobs.Client, error) {
	if c.sasToken != "" {
		log.Printf("[DEBUG] Building the Blob Client from a SAS Token")
		storageAuth, err := autorest.NewSASTokenAuthorizer(c.sasToken)
		if err != nil {
			return nil, fmt.Errorf("Error building Authorizer: %+v", err)
		}

		blobsClient := blobs.NewWithEnvironment(c.environment)
		c.configureClient(&blobsClient.Client, storageAuth)
		return &blobsClient, nil
	}

	if c.azureAdStorageAuth != nil {
		blobsClient := blobs.NewWithEnvironment(c.environment)
		c.configureClient(&blobsClient.Client, *c.azureAdStorageAuth)
		return &blobsClient, nil
	}

	accessKey := c.accessKey
	if accessKey == "" {
		log.Printf("[DEBUG] Building the Blob Client from an Access Token (using user credentials)")
		keys, err := c.storageAccountsClient.ListKeys(ctx, c.resourceGroupName, c.storageAccountName, "")
		if err != nil {
			return nil, fmt.Errorf("Error retrieving keys for Storage Account %q: %s", c.storageAccountName, err)
		}

		if keys.Keys == nil {
			return nil, fmt.Errorf("Nil key returned for storage account %q", c.storageAccountName)
		}

		accessKeys := *keys.Keys
		accessKey = *accessKeys[0].Value
	}

	storageAuth, err := autorest.NewSharedKeyAuthorizer(c.storageAccountName, accessKey, autorest.SharedKey)
	if err != nil {
		return nil, fmt.Errorf("Error building Authorizer: %+v", err)
	}

	blobsClient := blobs.NewWithEnvironment(c.environment)
	c.configureClient(&blobsClient.Client, storageAuth)
	return &blobsClient, nil
}

func (c *Client) getContainersClient(ctx context.Context) (*containers.Client, error) {
	if c.sasToken != "" {
		log.Printf("[DEBUG] Building the Container Client from a SAS Token")
		storageAuth, err := autorest.NewSASTokenAuthorizer(c.sasToken)
		if err != nil {
			return nil, fmt.Errorf("Error building Authorizer: %+v", err)
		}

		containersClient := containers.NewWithEnvironment(c.environment)
		c.configureClient(&containersClient.Client, storageAuth)
		return &containersClient, nil
	}

	if c.azureAdStorageAuth != nil {
		containersClient := containers.NewWithEnvironment(c.environment)
		c.configureClient(&containersClient.Client, *c.azureAdStorageAuth)
		return &containersClient, nil
	}

	accessKey := c.accessKey
	if accessKey == "" {
		log.Printf("[DEBUG] Building the Container Client from an Access Token (using user credentials)")
		keys, err := c.storageAccountsClient.ListKeys(ctx, c.resourceGroupName, c.storageAccountName, "")
		if err != nil {
			return nil, fmt.Errorf("Error retrieving keys for Storage Account %q: %s", c.storageAccountName, err)
		}

		if keys.Keys == nil {
			return nil, fmt.Errorf("Nil key returned for storage account %q", c.storageAccountName)
		}

		accessKeys := *keys.Keys
		accessKey = *accessKeys[0].Value
	}

	storageAuth, err := autorest.NewSharedKeyAuthorizer(c.storageAccountName, accessKey, autorest.SharedKey)
	if err != nil {
		return nil, fmt.Errorf("Error building Authorizer: %+v", err)
	}

	containersClient := containers.NewWithEnvironment(c.environment)
	c.configureClient(&containersClient.Client, storageAuth)
	return &containersClient, nil
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
