// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/2017-03-09/resources/mgmt/resources"
	armKeyVault "github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/keyvault/mgmt/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	armStorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-01-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/gofrs/uuid"
	sasStorage "github.com/hashicorp/go-azure-helpers/storage"
	"github.com/tombuildsstuff/giovanni/storage/2018-11-09/blob/containers"
	"github.com/tombuildsstuff/kermit/sdk/keyvault/7.4/keyvault"
)

const (
	// required for Azure Stack
	sasSignedVersion = "2015-04-05"
	// This is hex of AES-32 byte (256 bit) key generated with
	// bash equivalient :: openssl rand 32 | xxd -p | xxd -p -r
	kvSecretHex = "aa913b33a385c806b54bc224183addea1266343e99bf47d581c976e14b318fa4"
)

type KVARMClient struct {
	armClient    *ArmClient
	tenantID     string
	clientID     string
	clientSecret string
	armLocation  string
}

// verify that we are doing ACC tests or the Azure tests specifically
func testAccAzureBackend(t *testing.T) {
	skip := os.Getenv("TF_ACC") == "" && os.Getenv("TF_AZURE_TEST") == ""
	if skip {
		t.Log("azure backend tests require setting TF_ACC or TF_AZURE_TEST")
		t.Skip()
	}
}

// these kind of tests can only run when within Azure (e.g. MSI)
func testAccAzureBackendRunningInAzure(t *testing.T) {
	testAccAzureBackend(t)

	if os.Getenv("TF_RUNNING_IN_AZURE") == "" {
		t.Skip("Skipping test since not running in Azure")
	}
}

// these kind of tests can only run when within GitHub Actions (e.g. OIDC)
func testAccAzureBackendRunningInGitHubActions(t *testing.T) {
	testAccAzureBackend(t)

	if os.Getenv("TF_RUNNING_IN_GITHUB_ACTIONS") == "" {
		t.Skip("Skipping test since not running in GitHub Actions")
	}
}

func buildTestClient(t *testing.T, res resourceNames) *ArmClient {
	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	tenantID := os.Getenv("ARM_TENANT_ID")
	clientID := os.Getenv("ARM_CLIENT_ID")
	clientSecret := os.Getenv("ARM_CLIENT_SECRET")
	msiEnabled := strings.EqualFold(os.Getenv("ARM_USE_MSI"), "true")
	environment := os.Getenv("ARM_ENVIRONMENT")

	hasCredentials := (clientID != "" && clientSecret != "") || msiEnabled
	if !hasCredentials {
		t.Fatal("Azure credentials missing or incomplete")
	}

	if subscriptionID == "" {
		t.Fatalf("Missing ARM_SUBSCRIPTION_ID")
	}

	if tenantID == "" {
		t.Fatalf("Missing ARM_TENANT_ID")
	}

	if environment == "" {
		t.Fatalf("Missing ARM_ENVIRONMENT")
	}

	// location isn't used in this method, but is in the other test methods
	location := os.Getenv("ARM_LOCATION")
	if location == "" {
		t.Fatalf("Missing ARM_LOCATION")
	}

	// Endpoint is optional (only for Stack)
	endpoint := os.Getenv("ARM_ENDPOINT")

	armClient, err := buildArmClient(context.TODO(), BackendConfig{
		SubscriptionID:                subscriptionID,
		TenantID:                      tenantID,
		ClientID:                      clientID,
		ClientSecret:                  clientSecret,
		CustomResourceManagerEndpoint: endpoint,
		Environment:                   environment,
		ResourceGroupName:             res.resourceGroup,
		StorageAccountName:            res.storageAccountName,
		UseMsi:                        msiEnabled,
		UseAzureADAuthentication:      res.useAzureADAuth,
		KeyVaultSecretURI:             res.keyVaultSecretURI,
	})
	if err != nil {
		t.Fatalf("Failed to build ArmClient: %+v", err)
	}

	return armClient
}

func buildSasToken(accountName, accessKey string) (*string, error) {
	// grant full access to Objects in the Blob Storage Account
	permissions := "rwdlacup" // full control
	resourceTypes := "sco"    // service, container, object
	services := "b"           // blob

	// Details on how to do this are here:
	// https://docs.microsoft.com/en-us/rest/api/storageservices/Constructing-an-Account-SAS
	signedProtocol := "https,http"
	signedIp := ""
	signedVersion := sasSignedVersion

	utcNow := time.Now().UTC()

	// account for servers being up to 5 minutes out
	startDate := utcNow.Add(time.Minute * -5).Format(time.RFC3339)
	endDate := utcNow.Add(time.Hour * 24).Format(time.RFC3339)

	sasToken, err := sasStorage.ComputeAccountSASToken(accountName, accessKey, permissions, services, resourceTypes,
		startDate, endDate, signedProtocol, signedIp, signedVersion)
	if err != nil {
		return nil, fmt.Errorf("Error computing SAS Token: %+v", err)
	}
	log.Printf("SAS Token should be %q", sasToken)
	return &sasToken, nil
}

type resourceNames struct {
	resourceGroup           string
	location                string
	storageAccountName      string
	storageContainerName    string
	storageKeyName          string
	storageAccountAccessKey string
	useAzureADAuth          bool
	keyVaultSecretURI       string
}

func testResourceNames(rString string, keyName string) resourceNames {
	return resourceNames{
		resourceGroup:        fmt.Sprintf("acctestRG-backend-%s-%s", strings.Replace(time.Now().Local().Format("060102150405.00"), ".", "", 1), rString),
		location:             os.Getenv("ARM_LOCATION"),
		storageAccountName:   fmt.Sprintf("acctestsa%s", rString),
		storageContainerName: "acctestcont",
		storageKeyName:       keyName,
		useAzureADAuth:       false,
		keyVaultSecretURI:    "",
	}
}

func (c *ArmClient) buildTestResources(ctx context.Context, names *resourceNames) error {
	log.Printf("Creating Resource Group %q", names.resourceGroup)
	_, err := c.groupsClient.CreateOrUpdate(ctx, names.resourceGroup, resources.Group{Location: &names.location})
	if err != nil {
		return fmt.Errorf("failed to create test resource group: %s", err)
	}

	log.Printf("Creating Storage Account %q in Resource Group %q", names.storageAccountName, names.resourceGroup)
	storageProps := armStorage.AccountCreateParameters{
		Sku: &armStorage.Sku{
			Name: armStorage.StandardLRS,
			Tier: armStorage.Standard,
		},
		Location: &names.location,
		// TLS 1.0 & 1.1 will be retired from 01.11.2024
		AccountPropertiesCreateParameters: &armStorage.AccountPropertiesCreateParameters{
			MinimumTLSVersion: armStorage.TLS12,
		},
	}
	if names.useAzureADAuth {
		allowSharedKeyAccess := false
		storageProps.AccountPropertiesCreateParameters = &armStorage.AccountPropertiesCreateParameters{
			AllowSharedKeyAccess: &allowSharedKeyAccess,
			MinimumTLSVersion:    armStorage.TLS12,
		}
	}

	// create kv, secret and set aes-256 secret
	if names.useAzureADAuth && names.keyVaultSecretURI != "" {

		kvARMClient := KVARMClient{
			armClient:    c,
			tenantID:     os.Getenv("ARM_TENANT_ID"),
			clientID:     os.Getenv("ARM_CLIENT_ID"),
			clientSecret: os.Getenv("ARM_CLIENT_SECRET"),
			armLocation:  os.Getenv("ARM_LOCATION"),
		}

		appID, err := kvARMClient.fetchAppIDFromGraphAPI(ctx)
		if err != nil {
			return fmt.Errorf("Error fetching application id of service principal: %s", err)
		}

		kvName, secretName, _, err := extractKeyVaultAndSecretNames(c.keyVaultSecretURI)
		if err != nil {
			return fmt.Errorf("Error parsing kevault secret URI: %s", err)
		}

		kvProperties, err := kvARMClient.createKV(ctx, kvName, *appID.Value)
		if err != nil {
			return fmt.Errorf("Error creating keyvault: %s", err)
		}

		// set secret via properties here
		secretProps := keyvault.SecretSetParameters{
			Value: to.StringPtr(kvSecretHex),
		}

		_, err = c.keyVaultSecretsClient.SetSecret(ctx, *kvProperties.VaultURI, secretName, secretProps)
		if err != nil {
			return fmt.Errorf("Error creating the keyvault secret: %s", err)
		}
	}

	future, err := c.storageAccountsClient.Create(ctx, names.resourceGroup, names.storageAccountName, storageProps)
	if err != nil {
		return fmt.Errorf("failed to create test storage account: %s", err)
	}

	err = future.WaitForCompletionRef(ctx, c.storageAccountsClient.Client)
	if err != nil {
		return fmt.Errorf("failed waiting for the creation of storage account: %s", err)
	}

	containersClient := containers.NewWithEnvironment(c.environment)
	if names.useAzureADAuth {
		containersClient.Client.Authorizer = *c.azureAdStorageAuth
	} else {
		log.Printf("fetching access key for storage account")
		resp, err := c.storageAccountsClient.ListKeys(ctx, names.resourceGroup, names.storageAccountName, "")
		if err != nil {
			return fmt.Errorf("failed to list storage account keys %s:", err)
		}

		keys := *resp.Keys
		accessKey := *keys[0].Value
		names.storageAccountAccessKey = accessKey

		storageAuth, err := autorest.NewSharedKeyAuthorizer(names.storageAccountName, accessKey, autorest.SharedKey)
		if err != nil {
			return fmt.Errorf("Error building Authorizer: %+v", err)
		}

		containersClient.Client.Authorizer = storageAuth
	}

	log.Printf("Creating Container %q in Storage Account %q (Resource Group %q)", names.storageContainerName, names.storageAccountName, names.resourceGroup)
	_, err = containersClient.Create(ctx, names.storageAccountName, names.storageContainerName, containers.CreateInput{})
	if err != nil {
		return fmt.Errorf("failed to create storage container: %s", err)
	}

	return nil
}

func (k *KVARMClient) createKV(ctx context.Context, kvName, appID string) (*armKeyVault.VaultProperties, error) {
	authorizer, err := k.getAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err)
	}
	kvClient := armKeyVault.NewVaultsClient(k.armClient.groupsClient.SubscriptionID)
	kvClient.Authorizer = authorizer

	// convert tenant_id to UUID
	tenant_id, err := uuid.FromString(k.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse UUID from ARM_TENANT_ID: %s", err)
	}

	keyVaultProps := armKeyVault.VaultProperties{
		// disable softdelete (not required for tests)
		EnableSoftDelete:          to.BoolPtr(false),
		SoftDeleteRetentionInDays: nil,
		TenantID:                  &tenant_id,
		Sku: &armKeyVault.Sku{
			Name:   armKeyVault.Standard,
			Family: to.StringPtr("A"),
		},
		EnableRbacAuthorization: to.BoolPtr(false),
		// KV accesspolicy is chosen here over RBAC
		AccessPolicies: &[]armKeyVault.AccessPolicyEntry{
			{
				ObjectID: &appID,
				TenantID: &tenant_id,
				Permissions: &armKeyVault.Permissions{
					Secrets: &[]armKeyVault.SecretPermissions{
						armKeyVault.SecretPermissionsAll,
					},
				},
			},
		},
	}

	log.Printf("Creating keyvault %q", kvName)
	future, err := kvClient.CreateOrUpdate(ctx, k.armClient.resourceGroupName, kvName, armKeyVault.VaultCreateOrUpdateParameters{
		Properties: &keyVaultProps,
		Location:   to.StringPtr(k.armLocation),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create test keyvault: %s", err)
	}

	err = future.WaitForCompletionRef(ctx, kvClient.Client)
	if err != nil {
		return nil, fmt.Errorf("failed waiting for the creation of keyvault: %s", err)
	}

	testKv, err := future.Result(kvClient)
	if err != nil {
		return nil, fmt.Errorf("failed fetching the created keyvault: %s", err)
	}

	return testKv.Properties, nil
}

// we need to fetch object ID of the service principal
// this is possible to find either with graphAPI or
// parsing jwt tokens (oid claim). Below func uses graphAPI
func (k *KVARMClient) fetchAppIDFromGraphAPI(ctx context.Context) (*graphrbac.ServicePrincipalObjectResult, error) {
	graphAuthorizer, err := k.getGraphAuthorizer()
	if err != nil {
		return nil, err
	}

	graphClient := graphrbac.NewApplicationsClient(k.tenantID)
	graphClient.Authorizer = graphAuthorizer

	app, err := graphClient.GetServicePrincipalsIDByAppID(ctx, k.clientID)
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// we create authorizer for keyvault client (only required for ACC tests)
func (k *KVARMClient) getAuthorizer() (autorest.Authorizer, error) {
	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, k.tenantID)
	if err != nil {
		return nil, err
	}
	spToken, err := adal.NewServicePrincipalToken(*oauthConfig, k.clientID, k.clientSecret, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}
	return autorest.NewBearerAuthorizer(spToken), nil
}

// we create authorizer for graph client (only required for ACC tests)
// this is different API from ARM resource endpoint
func (k *KVARMClient) getGraphAuthorizer() (autorest.Authorizer, error) {
	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, k.tenantID)
	if err != nil {
		return nil, err
	}
	spToken, err := adal.NewServicePrincipalToken(*oauthConfig, k.clientID, k.clientSecret, azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}
	return autorest.NewBearerAuthorizer(spToken), nil
}

func (c ArmClient) destroyTestResources(ctx context.Context, resources resourceNames) error {
	log.Printf("[DEBUG] Deleting Resource Group %q..", resources.resourceGroup)
	future, err := c.groupsClient.Delete(ctx, resources.resourceGroup)
	if err != nil {
		return fmt.Errorf("Error deleting Resource Group: %+v", err)
	}

	log.Printf("[DEBUG] Waiting for deletion of Resource Group %q..", resources.resourceGroup)
	err = future.WaitForCompletionRef(ctx, c.groupsClient.Client)
	if err != nil {
		return fmt.Errorf("Error waiting for the deletion of Resource Group: %+v", err)
	}

	return nil
}
