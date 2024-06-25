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
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/2017-03-09/resources/mgmt/resources"
	armStorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-01-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	authWrapper "github.com/hashicorp/go-azure-sdk/sdk/auth/autorest"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/hashicorp/terraform/internal/httpclient"
	"github.com/hashicorp/terraform/version"
	"github.com/tombuildsstuff/giovanni/storage/2018-11-09/blob/blobs"
	"github.com/tombuildsstuff/giovanni/storage/2018-11-09/blob/containers"
)

type ArmClient struct {
	// These Clients are only initialized if an Access Key isn't provided
	groupsClient          *resources.GroupsClient
	storageAccountsClient *armStorage.AccountsClient
	containersClient      *containers.Client
	blobsClient           *blobs.Client

	// azureAdStorageAuth is only here if we're using AzureAD Authentication but is an Authorizer for Storage
	azureAdStorageAuth *auth.Authorizer

	accessKey          string
	environment        environments.Environment
	resourceGroupName  string
	storageAccountName string
	sasToken           string
}

func buildArmClient(ctx context.Context, config BackendConfig) (*ArmClient, error) {
	var (
		env *environments.Environment
		err error
	)

	if config.MetadataHost != "" {
		if env, err = environments.FromEndpoint(ctx, fmt.Sprintf("https://%s", config.MetadataHost)); err != nil {
			return nil, err
		}
	} else {
		if env, err = environments.FromName(config.Environment); err != nil {
			return nil, err
		}
	}

	if config.CustomResourceManagerEndpoint != "" {
		env.ResourceManager = environments.ResourceManagerAPI(config.CustomResourceManagerEndpoint)
	}

	client := ArmClient{
		environment:        *env,
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

	oidcToken, err := getOidcToken(config)
	if err != nil {
		return nil, err
	}

	authConfig := &auth.Credentials{
		Environment:                *env,
		ClientID:                   config.ClientID,
		TenantID:                   config.TenantID,
		AzureCliSubscriptionIDHint: config.SubscriptionID,

		// Service Principal (Client Certificate)
		ClientCertificatePath:     config.ClientCertificatePath,
		ClientCertificatePassword: config.ClientCertificatePassword,

		// Service Principal (Client Secret)
		ClientSecret: config.ClientSecret,

		// Managed Service Identity
		CustomManagedIdentityEndpoint: config.MsiEndpoint,

		// OIDC
		OIDCAssertionToken:          *oidcToken,
		GitHubOIDCTokenRequestURL:   config.OIDCRequestURL,
		GitHubOIDCTokenRequestToken: config.OIDCRequestToken,

		// Feature Toggles
		EnableAuthenticatingUsingClientCertificate: true,
		EnableAuthenticatingUsingClientSecret:      true,
		EnableAuthenticatingUsingAzureCLI:          true,
		EnableAuthenticatingUsingManagedIdentity:   config.UseMsi,
		EnableAuthenticationUsingOIDC:              config.UseOIDC,
		EnableAuthenticationUsingGitHubOIDC:        config.UseOIDC,
	}

	authorizer, err := auth.NewAuthorizerFromCredentials(ctx, *authConfig, authConfig.Environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("unable to build authorizer for Resource Manager API: %+v", err)
	}

	subscriptionId, err := getSubscriptionId(ctx, *authConfig)
	if err != nil {
		return nil, fmt.Errorf("building account: %+v", err)
	}

	if config.UseAzureADAuthentication {
		log.Printf("[DEBUG] Obtaining an MSAL / Microsoft Graph token for Storage..")
		client.azureAdStorageAuth = &authorizer
	}

	resourceManagerEndpoint, ok := authConfig.Environment.ResourceManager.Endpoint()
	if !ok {
		return nil, fmt.Errorf("unable to determine resource manager endpoint for the current environment")
	}

	autorestAuthorizer := authWrapper.AutorestAuthorizer(authorizer)
	accountsClient := armStorage.NewAccountsClientWithBaseURI(*resourceManagerEndpoint, *subscriptionId)
	client.configureClient(&accountsClient.Client, autorestAuthorizer)
	client.storageAccountsClient = &accountsClient

	groupsClient := resources.NewGroupsClientWithBaseURI(*resourceManagerEndpoint, *subscriptionId)
	client.configureClient(&groupsClient.Client, autorestAuthorizer)
	client.groupsClient = &groupsClient

	return &client, nil
}

func (c ArmClient) getBlobClient(ctx context.Context) (*blobs.Client, error) {
	if c.sasToken != "" {
		log.Printf("[DEBUG] Building the Blob Client from a SAS Token")
		storageAuth, err := autorest.NewSASTokenAuthorizer(c.sasToken)
		if err != nil {
			return nil, fmt.Errorf("Error building Authorizer: %+v", err)
		}

		blobsClient := blobs.New()
		domainSuffix, ok := c.environment.Storage.DomainSuffix()
		if !ok {
			return nil, fmt.Errorf("Error retrieving domain suffix for storage account, environment: %v", c.environment)
		}
		blobsClient.BaseURI = *domainSuffix
		c.configureClient(&blobsClient.Client, storageAuth)
		return &blobsClient, nil
	}

	if c.azureAdStorageAuth != nil {
		blobsClient := blobs.New()
		domainSuffix, ok := c.environment.Storage.DomainSuffix()
		if !ok {
			return nil, fmt.Errorf("Error retrieving domain suffix for storage account, environment: %v", c.environment)
		}
		blobsClient.BaseURI = *domainSuffix
		c.configureClient(&blobsClient.Client, authWrapper.AutorestAuthorizer(*c.azureAdStorageAuth))
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

	blobsClient := blobs.New()
	domainSuffix, ok := c.environment.Storage.DomainSuffix()
	if !ok {
		return nil, fmt.Errorf("Error retrieving domain suffix for storage account, environment: %v", c.environment)
	}
	blobsClient.BaseURI = *domainSuffix
	c.configureClient(&blobsClient.Client, storageAuth)
	return &blobsClient, nil
}

func (c ArmClient) getContainersClient(ctx context.Context) (*containers.Client, error) {
	if c.sasToken != "" {
		log.Printf("[DEBUG] Building the Container Client from a SAS Token")
		storageAuth, err := autorest.NewSASTokenAuthorizer(c.sasToken)
		if err != nil {
			return nil, fmt.Errorf("Error building Authorizer: %+v", err)
		}

		containersClient := containers.New()
		domainSuffix, ok := c.environment.Storage.DomainSuffix()
		if !ok {
			return nil, fmt.Errorf("Error retrieving domain suffix for storage account, environment: %v", c.environment)
		}
		containersClient.BaseURI = *domainSuffix
		c.configureClient(&containersClient.Client, storageAuth)
		return &containersClient, nil
	}

	if c.azureAdStorageAuth != nil {
		containersClient := containers.New()
		domainSuffix, ok := c.environment.Storage.DomainSuffix()
		if !ok {
			return nil, fmt.Errorf("Error retrieving domain suffix for storage account, environment: %v", c.environment)
		}
		containersClient.BaseURI = *domainSuffix
		c.configureClient(&containersClient.Client, authWrapper.AutorestAuthorizer(*c.azureAdStorageAuth))
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

	containersClient := containers.New()
	domainSuffix, ok := c.environment.Storage.DomainSuffix()
	if !ok {
		return nil, fmt.Errorf("Error retrieving domain suffix for storage account, environment: %v", c.environment)
	}
	containersClient.BaseURI = *domainSuffix
	c.configureClient(&containersClient.Client, storageAuth)
	return &containersClient, nil
}

func (c *ArmClient) configureClient(client *autorest.Client, auth autorest.Authorizer) {
	client.UserAgent = buildUserAgent()
	client.Authorizer = auth
	client.Sender = buildSender()
	client.SkipResourceProviderRegistration = false
	client.PollingDuration = 60 * time.Minute
}

func buildUserAgent() string {
	userAgent := httpclient.TerraformUserAgent(version.Version)

	// append the CloudShell version to the user agent if it exists
	if azureAgent := os.Getenv("AZURE_HTTP_USER_AGENT"); azureAgent != "" {
		userAgent = fmt.Sprintf("%s %s", userAgent, azureAgent)
	}

	return userAgent
}

func getOidcToken(config BackendConfig) (*string, error) {
	idToken := strings.TrimSpace(config.OIDCToken)

	if path := config.OIDCTokenFilePath; path != "" {
		fileTokenRaw, err := os.ReadFile(path)

		if err != nil {
			return nil, fmt.Errorf("reading OIDC Token from file %q: %v", path, err)
		}

		fileToken := strings.TrimSpace(string(fileTokenRaw))

		if idToken != "" && idToken != fileToken {
			return nil, fmt.Errorf("mismatch between supplied OIDC token and supplied OIDC token file contents - please either remove one or ensure they match")
		}

		idToken = fileToken
	}

	return &idToken, nil
}

func getSubscriptionId(ctx context.Context, config auth.Credentials) (*string, error) {
	if config.AzureCliSubscriptionIDHint != "" {
		return &config.AzureCliSubscriptionIDHint, nil
	}
	authorizer, err := auth.NewAuthorizerFromCredentials(ctx, config, config.Environment.MicrosoftGraph)
	if err != nil {
		return nil, fmt.Errorf("unable to build authorizer for Microsoft Graph API: %+v", err)
	}

	// Acquire an access token so we can inspect the claims
	_, err = authorizer.Token(ctx, &http.Request{})
	if err != nil {
		return nil, fmt.Errorf("could not acquire access token to parse claims: %+v", err)
	}
	// Finally, defer to Azure CLI to obtain tenant ID, subscription ID and client ID when not specified and missing from claims
	realAuthorizer := authorizer
	if cache, ok := authorizer.(*auth.CachedAuthorizer); ok {
		realAuthorizer = cache.Source
	}
	subscriptionId := ""
	if cli, ok := realAuthorizer.(*auth.AzureCliAuthorizer); ok {

		if cli.DefaultSubscriptionID == "" {
			return nil, fmt.Errorf("azure-cli could not determine subscription ID to use and no subscription was specified")
		}

		subscriptionId = cli.DefaultSubscriptionID
		log.Printf("[DEBUG] Using default subscription ID from Azure CLI: %q", subscriptionId)
	}

	if subscriptionId == "" {
		return nil, fmt.Errorf("unable to configure ResourceManagerAccount: subscription ID could not be determined and was not specified")
	}

	return &subscriptionId, nil
}
