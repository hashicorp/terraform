// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/2017-03-09/resources/mgmt/resources"
	armStorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-01-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/hashicorp/go-azure-helpers/authentication"
	"github.com/hashicorp/go-azure-helpers/sender"
	"github.com/hashicorp/terraform/internal/httpclient"
	"github.com/hashicorp/terraform/version"
	"github.com/manicminer/hamilton/environments"
	"github.com/tombuildsstuff/giovanni/storage/2018-11-09/blob/blobs"
	"github.com/tombuildsstuff/giovanni/storage/2018-11-09/blob/containers"
	"github.com/tombuildsstuff/kermit/sdk/keyvault/7.4/keyvault"
)

// we set these headers to encrypt tfstate blob with CMK
// https://learn.microsoft.com/en-us/azure/storage/blobs/encryption-customer-provided-keys#request-headers-for-specifying-customer-provided-keys
const (
	encryptionAlgorithm          = "AES256"
	cmkEncryptionAlgorithmHeader = "x-ms-encryption-algorithm"
	cmkEncryptionKeyHeader       = "x-ms-encryption-key"
	cmkEncryptionKeySHA256Header = "x-ms-encryption-key-sha256"
)

type ArmClient struct {
	// These Clients are only initialized if an Access Key isn't provided
	groupsClient          *resources.GroupsClient
	storageAccountsClient *armStorage.AccountsClient
	containersClient      *containers.Client
	blobsClient           *blobs.Client
	keyVaultSecretsClient keyvault.BaseClient

	// azureAdStorageAuth is only here if we're using AzureAD Authentication but is an Authorizer for Storage & keyvault
	azureAdStorageAuth  *autorest.Authorizer
	azureAdKVSecretAuth *autorest.Authorizer
	keyVaultSecretURI   string

	accessKey          string
	environment        azure.Environment
	resourceGroupName  string
	storageAccountName string
	sasToken           string
}

func buildArmClient(ctx context.Context, config BackendConfig) (*ArmClient, error) {
	env, err := authentication.AzureEnvironmentByNameFromEndpoint(ctx, config.MetadataHost, config.Environment)
	if err != nil {
		return nil, err
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

	builder := authentication.Builder{
		ClientID:                      config.ClientID,
		SubscriptionID:                config.SubscriptionID,
		TenantID:                      config.TenantID,
		CustomResourceManagerEndpoint: config.CustomResourceManagerEndpoint,
		MetadataHost:                  config.MetadataHost,
		Environment:                   config.Environment,
		ClientSecretDocsLink:          "https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/guides/service_principal_client_secret",

		// Service Principal (Client Certificate)
		ClientCertPassword: config.ClientCertificatePassword,
		ClientCertPath:     config.ClientCertificatePath,

		// Service Principal (Client Secret)
		ClientSecret: config.ClientSecret,

		// Managed Service Identity
		MsiEndpoint: config.MsiEndpoint,

		// OIDC
		IDToken:             config.OIDCToken,
		IDTokenFilePath:     config.OIDCTokenFilePath,
		IDTokenRequestURL:   config.OIDCRequestURL,
		IDTokenRequestToken: config.OIDCRequestToken,

		// Feature Toggles
		SupportsAzureCliToken:          true,
		SupportsClientCertAuth:         true,
		SupportsClientSecretAuth:       true,
		SupportsManagedServiceIdentity: config.UseMsi,
		SupportsOIDCAuth:               config.UseOIDC,
		UseMicrosoftGraph:              true,
	}
	armConfig, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("Error building ARM Config: %+v", err)
	}

	oauthConfig, err := armConfig.BuildOAuthConfig(env.ActiveDirectoryEndpoint)
	if err != nil {
		return nil, err
	}

	hamiltonEnv, err := environments.EnvironmentFromString(config.Environment)
	if err != nil {
		return nil, err
	}

	sender := sender.BuildSender("backend/remote-state/azure")
	log.Printf("[DEBUG] Obtaining an MSAL / Microsoft Graph token for Resource Manager..")
	auth, err := armConfig.GetMSALToken(ctx, hamiltonEnv.ResourceManager, sender, oauthConfig, env.TokenAudience)
	if err != nil {
		return nil, err
	}

	if config.UseAzureADAuthentication {
		log.Printf("[DEBUG] Obtaining an MSAL / Microsoft Graph token for Storage..")
		storageAuth, err := armConfig.GetMSALToken(ctx, hamiltonEnv.Storage, sender, oauthConfig, env.ResourceIdentifiers.Storage)
		if err != nil {
			return nil, err
		}
		client.azureAdStorageAuth = &storageAuth
	}

	if config.KeyVaultSecretURI != "" && config.UseAzureADAuthentication {
		// this is for keyvault secrets (data plan API)
		keyVaultSecretAuth, err := armConfig.GetMSALToken(ctx, hamiltonEnv.KeyVault, sender, oauthConfig, env.ResourceIdentifiers.KeyVault)
		if err != nil {
			return nil, err
		}
		client.azureAdKVSecretAuth = &keyVaultSecretAuth
		client.keyVaultSecretURI = config.KeyVaultSecretURI

		// we build keyvault secret client here
		kvSecretClient := keyvault.New()
		client.configureClient(&kvSecretClient.Client, keyVaultSecretAuth)
		client.keyVaultSecretsClient = kvSecretClient
	}

	accountsClient := armStorage.NewAccountsClientWithBaseURI(env.ResourceManagerEndpoint, armConfig.SubscriptionID)
	client.configureClient(&accountsClient.Client, auth)
	client.storageAccountsClient = &accountsClient

	groupsClient := resources.NewGroupsClientWithBaseURI(env.ResourceManagerEndpoint, armConfig.SubscriptionID)
	client.configureClient(&groupsClient.Client, auth)
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

		blobsClient := blobs.NewWithEnvironment(c.environment)
		c.configureClient(&blobsClient.Client, storageAuth)
		return &blobsClient, nil
	}

	if c.azureAdStorageAuth != nil {
		log.Printf("[DEBUG] Building the Blob Client using AD authentication")
		blobsClient := blobs.NewWithEnvironment(c.environment)
		if c.keyVaultSecretURI != "" {
			log.Printf("[DEBUG] Building the Blob Client using AD authentication with CMK enabled")
			secretValue, err := c.fetchSecretFromKV(ctx)
			if err != nil {
				return nil, fmt.Errorf("Error retrieving secret from keyvault %q: %s", c.keyVaultSecretURI, err)
			}
			if secretValue != "" {
				cmkEncryptHeaders, err := setEncryptionHeaders(secretValue)
				if err != nil {
					return nil, fmt.Errorf("Error setting up encryption headers %+v", err)
				}
				blobsClient.RequestInspector = autorest.WithHeaders(cmkEncryptHeaders)
			}
		}
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

// Azure KV supports both keys & secrets. While key is the perfect fit for CMK but
// due to it's asymmetric nature but it's not possible to extract the key from Azure.
// Secrets on the other hand offer AES symmetric encryption, would help in recovering
// statefile in the events of corruption etc.,
func (c ArmClient) fetchSecretFromKV(ctx context.Context) (string, error) {

	if !IsValidKeyVaultSecretURI(c.keyVaultSecretURI) {
		return "", fmt.Errorf("Error invalid keyvault secret URI passed. It should be in format like https://<keyvault-name>.vault.azure.net/secrets/<secret-name>/<version-if-not-latest-is-chosen>")
	}

	kvName, secretName, version, err := extractKeyVaultAndSecretNames(c.keyVaultSecretURI)
	if err != nil {
		return "", fmt.Errorf("Error parsing keyvault and secret from key_vault_secret_uri: %+v", err)
	}
	if kvName != "" && secretName != "" {
		vaultBaseURL := fmt.Sprintf("https://%s.vault.azure.net/", kvName)
		log.Printf("[DEBUG] Building the Keyvault Client from Azure AD auth")
		secret, err := c.keyVaultSecretsClient.GetSecret(ctx, vaultBaseURL, secretName, version)
		if err != nil {
			return "", fmt.Errorf("Error fetching secret from kv: %+v", err)
		}
		log.Printf("[DEBUG] Fetched secret of %v", *secret.ID)
		return *secret.Value, nil
	}

	return "", nil
}

func (c ArmClient) getContainersClient(ctx context.Context) (*containers.Client, error) {
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

// we extract vault, secret names & versions from a given keyvault uri
// version is not mandatory, if not passed latest version is chosen
func extractKeyVaultAndSecretNames(keyVaultURI string) (string, string, string, error) {
	parsedURL, err := url.Parse(keyVaultURI)
	if err != nil {
		return "", "", "", err
	}

	// Extract Key Vault name from the host
	keyVaultName := strings.Split(parsedURL.Host, ".")[0]

	// Extract Secret name from the path
	segments := strings.Split(parsedURL.Path, "/")
	secretName := segments[2] // Index 2 is the secret name

	// Extract version from the path (if available)
	var version string
	if len(segments) >= 4 {
		version = segments[3] // Index 3 is the version
	}

	return keyVaultName, secretName, version, nil
}

// calculateSHA256 calculates the SHA256 hash of a given data
func calculateSHA256(data []byte) ([]byte, error) {
	hash := sha256.New()
	_, err := hash.Write(data)
	if err != nil {
		return nil, err
	}
	return hash.Sum(nil), nil
}

func setEncryptionHeaders(secretValue string) (map[string]interface{}, error) {
	// the secret is recommended to be stored in hex format on azure kv
	// this is due to the fact that many invisible characters of AES encryption key are lost
	// during copy paste as they are not neither visible to human eyes nor to `echo` command
	// we decode here to binary before base64 encoding as required by Azure
	decodedBinaryFromHex, err := hex.DecodeString(secretValue)
	if err != nil {
		return nil, fmt.Errorf("Error decoding keyvault secret from hex. Ensure the AES secret key is in hexadecimal format with no special characters. %+v", err)
	}

	// we calculate sha-2 output of above secret key
	sha256output, err := calculateSHA256(decodedBinaryFromHex)
	if err != nil {
		return nil, fmt.Errorf("Error applying sha256 to secret key %+v", err)
	}

	headers := map[string]interface{}{
		cmkEncryptionAlgorithmHeader: encryptionAlgorithm,
		cmkEncryptionKeyHeader:       base64.StdEncoding.EncodeToString(decodedBinaryFromHex),
		cmkEncryptionKeySHA256Header: base64.StdEncoding.EncodeToString(sha256output),
	}
	return headers, nil
}

// isValidKeyVaultSecretURI checks if the provided string is a valid Azure Key Vault secret URI
func IsValidKeyVaultSecretURI(uri string) bool {
	// Key Vault URI pattern: https://<vault-name>.vault.azure.net/secrets/<secret-name>/<version>
	keyVaultSecretURIPattern := regexp.MustCompile(`^https:\/\/[a-zA-Z0-9-]+\.vault\.azure\.net\/secrets\/[a-zA-Z0-9-]+(\/[a-zA-Z0-9-]+)?$`)

	return keyVaultSecretURIPattern.MatchString(uri)
}
