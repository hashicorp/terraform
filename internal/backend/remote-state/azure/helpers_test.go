// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	sasStorage "github.com/hashicorp/go-azure-helpers/storage"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2024-03-01/resourcegroups"
	"github.com/hashicorp/go-azure-sdk/resource-manager/storage/2023-01-01/storageaccounts"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/jackofallops/giovanni/storage/2023-11-03/blob/blobs"
	"github.com/jackofallops/giovanni/storage/2023-11-03/blob/containers"
)

const (
	// required for Azure Stack
	sasSignedVersion = "2015-04-05"
)

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

// these kind of tests can only run when within ADO Pipelines (e.g. OIDC)
func testAccAzureBackendRunningInADOPipelines(t *testing.T) {
	testAccAzureBackend(t)

	if os.Getenv("TF_RUNNING_IN_ADO_PIPELINES") == "" {
		t.Skip("Skipping test since not running in ADO Pipelines")
	}
}

// clearARMEnv cleans up the azure related environment variables.
// This is to ensure the configuration only comes from HCL, which avoids
// env vars for test setup interfere the behavior.
//
// NOTE: Since `go test` runs all test cases in a single process, clearing
// environment has a whole process impact to other test cases. While this
// impact can be eliminated given all the tests are implemented in a similar
// pattern that those env vars will be consumed at the very begining. The test
// runner has to ensure to set a **big enough parallelism**.
func clearARMEnv() {
	for _, evexp := range os.Environ() {
		k, _, ok := strings.Cut(evexp, "=")
		if !ok {
			continue
		}
		if strings.HasPrefix(k, "ARM_") {
			os.Unsetenv(k)
		}
	}
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
	signedEncryptionScope := ""

	utcNow := time.Now().UTC()

	// account for servers being up to 5 minutes out
	startDate := utcNow.Add(time.Minute * -5).Format(time.RFC3339)
	endDate := utcNow.Add(time.Hour * 24).Format(time.RFC3339)

	sasToken, err := sasStorage.ComputeAccountSASToken(accountName, accessKey, permissions, services, resourceTypes,
		startDate, endDate, signedProtocol, signedIp, signedVersion, signedEncryptionScope)
	if err != nil {
		return nil, fmt.Errorf("Error computing SAS Token: %+v", err)
	}
	log.Printf("SAS Token should be %q", sasToken)
	return &sasToken, nil
}

type resourceNames struct {
	resourceGroup        string
	storageAccountName   string
	storageContainerName string
	storageKeyName       string
}

func testResourceNames(rString string, keyName string) resourceNames {
	return resourceNames{
		resourceGroup:        fmt.Sprintf("acctestRG-backend-%s-%s", strings.Replace(time.Now().Local().Format("060102150405.00"), ".", "", 1), rString),
		storageAccountName:   fmt.Sprintf("acctestsa%s", rString),
		storageContainerName: "acctestcont",
		storageKeyName:       keyName,
	}
}

type TestMeta struct {
	names resourceNames

	clientId     string
	clientSecret string

	tenantId       string
	subscriptionId string
	location       string
	env            environments.Environment

	// This is populated during test resource deploying
	storageAccessKey string

	// This is populated during test resoruce deploying
	blobBaseUri string

	resourceGroupsClient  *resourcegroups.ResourceGroupsClient
	storageAccountsClient *storageaccounts.StorageAccountsClient
}

func BuildTestMeta(t *testing.T, ctx context.Context) *TestMeta {
	names := testResourceNames(randString(10), "testState")

	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Fatalf("Missing ARM_SUBSCRIPTION_ID")
	}

	tenantID := os.Getenv("ARM_TENANT_ID")
	if tenantID == "" {
		t.Fatalf("Missing ARM_TENANT_ID")
	}

	location := os.Getenv("ARM_LOCATION")
	if location == "" {
		t.Fatalf("Missing ARM_LOCATION")
	}

	clientID := os.Getenv("ARM_CLIENT_ID")
	clientSecret := os.Getenv("ARM_CLIENT_SECRET")

	environment := "public"
	if v := os.Getenv("ARM_ENVIRONMENT"); v != "" {
		environment = v
	}
	env, err := environments.FromName(environment)
	if err != nil {
		t.Fatalf("Failed to build environment for %s: %v", environment, err)
	}

	// For deploying test resources, we support the followings:
	// - Client secret: For most of the tests
	// - Client certificate: For client certificate related tests
	// - MSI: For MSI related tests
	// - OIDC: For OIDC related tests
	authConfig := &auth.Credentials{
		Environment:                    *env,
		TenantID:                       tenantID,
		ClientID:                       clientID,
		ClientSecret:                   clientSecret,
		ClientCertificatePath:          os.Getenv("ARM_CLIENT_CERTIFICATE_PATH"),
		ClientCertificatePassword:      os.Getenv("ARM_CLIENT_CERTIFICATE_PASSWORD"),
		OIDCTokenRequestURL:            getEnvvars("ACTIONS_ID_TOKEN_REQUEST_URL", "SYSTEM_OIDCREQUESTURI"),
		OIDCTokenRequestToken:          getEnvvars("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "SYSTEM_ACCESSTOKEN"),
		ADOPipelineServiceConnectionID: os.Getenv("ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID"),

		EnableAuthenticatingUsingClientSecret:      true,
		EnableAuthenticatingUsingClientCertificate: true,
		EnableAuthenticatingUsingManagedIdentity:   true,
		EnableAuthenticationUsingGitHubOIDC:        true,
		EnableAuthenticationUsingADOPipelineOIDC:   true,
	}

	resourceManagerAuth, err := auth.NewAuthorizerFromCredentials(ctx, *authConfig, env.ResourceManager)
	if err != nil {
		t.Fatalf("unable to build authorizer for Resource Manager API: %+v", err)
	}

	resourceGroupsClient, err := resourcegroups.NewResourceGroupsClientWithBaseURI(env.ResourceManager)
	if err != nil {
		t.Fatalf("building Resource Groups client: %+v", err)
	}
	resourceGroupsClient.Client.SetAuthorizer(resourceManagerAuth)

	storageAccountsClient, err := storageaccounts.NewStorageAccountsClientWithBaseURI(env.ResourceManager)
	if err != nil {
		t.Fatalf("building Storage Accounts client: %+v", err)
	}
	storageAccountsClient.Client.SetAuthorizer(resourceManagerAuth)

	return &TestMeta{
		names: names,

		clientId:     clientID,
		clientSecret: clientSecret,

		tenantId:       tenantID,
		subscriptionId: subscriptionID,
		location:       location,
		env:            *env,

		resourceGroupsClient:  resourceGroupsClient,
		storageAccountsClient: storageAccountsClient,
	}
}

func (c *TestMeta) buildTestResources(ctx context.Context) error {
	log.Printf("Creating Resource Group %q", c.names.resourceGroup)
	rgid := commonids.NewResourceGroupID(c.subscriptionId, c.names.resourceGroup)
	if _, err := c.resourceGroupsClient.CreateOrUpdate(ctx, rgid, resourcegroups.ResourceGroup{Location: c.location}); err != nil {
		return fmt.Errorf("failed to create test resource group: %s", err)
	}

	log.Printf("Creating Storage Account %q in Resource Group %q", c.names.storageAccountName, c.names.resourceGroup)
	storageProps := storageaccounts.StorageAccountCreateParameters{
		Kind: storageaccounts.KindStorageVTwo,
		Sku: storageaccounts.Sku{
			Name: storageaccounts.SkuNameStandardLRS,
			Tier: pointer.To(storageaccounts.SkuTierStandard),
		},
		Location: c.location,
	}

	said := commonids.NewStorageAccountID(c.subscriptionId, c.names.resourceGroup, c.names.storageAccountName)
	if err := c.storageAccountsClient.CreateThenPoll(ctx, said, storageProps); err != nil {
		return fmt.Errorf("failed to create test storage account: %s", err)
	}

	// Populate the storage account access key
	resp, err := c.storageAccountsClient.GetProperties(ctx, said, storageaccounts.DefaultGetPropertiesOperationOptions())
	if err != nil {
		return fmt.Errorf("retrieving %s: %+v", said, err)
	}
	if resp.Model == nil {
		return fmt.Errorf("unexpected null model of %s", said)
	}
	accountDetail, err := populateAccountDetails(said, *resp.Model)
	if err != nil {
		return fmt.Errorf("populating details for %s: %+v", said, err)
	}

	accountKey, err := accountDetail.AccountKey(ctx, c.storageAccountsClient)
	if err != nil {
		return fmt.Errorf("listing access key for %s: %+v", said, err)
	}
	c.storageAccessKey = *accountKey

	blobBaseUri, err := accountDetail.DataPlaneEndpoint(EndpointTypeBlob)
	if err != nil {
		return err
	}
	c.blobBaseUri = *blobBaseUri

	containersClient, err := containers.NewWithBaseUri(*blobBaseUri)
	if err != nil {
		return fmt.Errorf("failed to new container client: %v", err)
	}

	authorizer, err := auth.NewSharedKeyAuthorizer(c.names.storageAccountName, *accountKey, auth.SharedKey)
	if err != nil {
		return fmt.Errorf("new shared key authorizer: %v", err)
	}
	containersClient.Client.Authorizer = authorizer

	log.Printf("Creating Container %q in Storage Account %q (Resource Group %q)", c.names.storageContainerName, c.names.storageAccountName, c.names.resourceGroup)
	if _, err = containersClient.Create(ctx, c.names.storageContainerName, containers.CreateInput{}); err != nil {
		return fmt.Errorf("failed to create storage container: %s", err)
	}

	return nil
}

func (c *TestMeta) destroyTestResources(ctx context.Context) error {
	log.Printf("[DEBUG] Deleting Resource Group %q..", c.names.resourceGroup)
	rgid := commonids.NewResourceGroupID(c.subscriptionId, c.names.resourceGroup)
	if err := c.resourceGroupsClient.DeleteThenPoll(ctx, rgid, resourcegroups.DefaultDeleteOperationOptions()); err != nil {
		return fmt.Errorf("Error deleting Resource Group: %+v", err)
	}
	return nil
}

func (c *TestMeta) getBlobClient(ctx context.Context) (bc *blobs.Client, err error) {
	blobsClient, err := blobs.NewWithBaseUri(c.blobBaseUri)
	if err != nil {
		return nil, fmt.Errorf("new blob client: %v", err)
	}

	authorizer, err := auth.NewSharedKeyAuthorizer(c.names.storageAccountName, c.storageAccessKey, auth.SharedKey)
	if err != nil {
		return nil, fmt.Errorf("new shared key authorizer: %v", err)
	}
	blobsClient.Client.SetAuthorizer(authorizer)

	return blobsClient, nil
}

// randString generates a random alphanumeric string of the length specified
func randString(strlen int) string {
	const charSet = "abcdefghijklmnopqrstuvwxyz012346789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = charSet[rand.Intn(len(charSet))]
	}
	return string(result)
}

// getEnvvars return the first non-empty env var specified. If none is found, it returns empty string.
func getEnvvars(envvars ...string) string {
	for _, envvar := range envvars {
		if v := os.Getenv(envvar); v != "" {
			return v
		}
	}
	return ""
}
