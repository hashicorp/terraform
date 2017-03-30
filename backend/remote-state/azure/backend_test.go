package azure

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	"github.com/Azure/azure-storage-go"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/acctest"
)

// verify that we are doing ACC tests or the Azure tests specifically
func testACC(t *testing.T) {
	skip := os.Getenv("TF_ACC") == "" && os.Getenv("TF_AZURE_TEST") == ""
	if skip {
		t.Log("azure backend tests require setting TF_ACC or TF_AZURE_TEST")
		t.Skip()
	}
}

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackendConfig(t *testing.T) {
	// This test just instantiates the client. Shouldn't make any actual
	// requests nor incur any costs.

	config := map[string]interface{}{
		"storage_account_name": "tfaccount",
		"container_name":       "tfcontainer",
		"key":                  "state",
		// Access Key must be Base64
		"access_key": "QUNDRVNTX0tFWQ0K",
	}

	b := backend.TestBackendConfig(t, New(), config).(*Backend)

	if b.containerName != "tfcontainer" {
		t.Fatalf("Incorrect bucketName was populated")
	}
	if b.keyName != "state" {
		t.Fatalf("Incorrect keyName was populated")
	}
}

func TestBackend(t *testing.T) {
	testACC(t)

	keyName := "testState"
	res := setupResources(t, keyName)
	defer destroyResources(t, res.resourceGroupName)

	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"storage_account_name": res.storageAccountName,
		"container_name":       res.containerName,
		"key":                  keyName,
		"access_key":           res.accessKey,
	}).(*Backend)

	backend.TestBackend(t, b, nil)
}

func TestBackendLocked(t *testing.T) {
	testACC(t)

	keyName := "testState"
	res := setupResources(t, keyName)
	defer destroyResources(t, res.resourceGroupName)

	b1 := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"storage_account_name": res.storageAccountName,
		"container_name":       res.containerName,
		"key":                  keyName,
		"access_key":           res.accessKey,
	}).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"storage_account_name": res.storageAccountName,
		"container_name":       res.containerName,
		"key":                  keyName,
		"access_key":           res.accessKey,
	}).(*Backend)

	backend.TestBackend(t, b1, b2)
}

type testResources struct {
	resourceGroupName  string
	storageAccountName string
	containerName      string
	keyName            string
	accessKey          string
}

func setupResources(t *testing.T, keyName string) testResources {
	clients := getTestClient(t)

	ri := acctest.RandInt()
	rs := acctest.RandString(4)
	res := testResources{
		resourceGroupName:  fmt.Sprintf("terraform-backend-testing-%d", ri),
		storageAccountName: fmt.Sprintf("tfbackendtesting%s", rs),
		containerName:      "terraform",
		keyName:            keyName,
	}

	location := os.Getenv("ARM_LOCATION")
	if location == "" {
		location = "westus"
	}

	t.Logf("creating resource group %s", res.resourceGroupName)
	_, err := clients.groupsClient.CreateOrUpdate(res.resourceGroupName, resources.Group{Location: &location})
	if err != nil {
		t.Fatalf("failed to create test resource group: %s", err)
	}

	t.Logf("creating storage account %s", res.storageAccountName)
	_, err = clients.storageAccountsClient.Create(res.resourceGroupName, res.storageAccountName, armStorage.AccountCreateParameters{
		Sku: &armStorage.Sku{
			Name: armStorage.StandardLRS,
			Tier: armStorage.Standard,
		},
		Location: &location,
	}, make(chan struct{}))
	if err != nil {
		destroyResources(t, res.resourceGroupName)
		t.Fatalf("failed to create test storage account: %s", err)
	}

	t.Log("fetching access key for storage account")
	resp, err := clients.storageAccountsClient.ListKeys(res.resourceGroupName, res.storageAccountName)
	if err != nil {
		destroyResources(t, res.resourceGroupName)
		t.Fatalf("failed to list storage account keys %s:", err)
	}

	keys := *resp.Keys
	res.accessKey = *keys[0].Value

	storageClient, err := storage.NewClient(res.storageAccountName, res.accessKey,
		clients.environment.StorageEndpointSuffix, storage.DefaultAPIVersion, true)
	if err != nil {
		destroyResources(t, res.resourceGroupName)
		t.Fatalf("failed to list storage account keys %s:", err)
	}

	t.Logf("creating container %s", res.containerName)
	container := storageClient.GetBlobService().GetContainerReference(res.containerName)
	err = container.Create()
	if err != nil {
		destroyResources(t, res.resourceGroupName)
		t.Fatalf("failed to create storage container: %s", err)
	}

	return res
}

func destroyResources(t *testing.T, resourceGroupName string) {
	warning := "WARNING: Failed to delete the test Azure resources. They may incur charges. (error was %s)"

	clients := getTestClient(t)

	t.Log("destroying created resources")

	// destroying is simple as deleting the resource group will destroy everything else
	_, err := clients.groupsClient.Delete(resourceGroupName, make(chan struct{}))
	if err != nil {
		t.Logf(warning, err)
		return
	}

	t.Log("Azure resources destroyed")
}

type testClient struct {
	subscriptionID        string
	tenantID              string
	clientID              string
	clientSecret          string
	environment           azure.Environment
	groupsClient          resources.GroupsClient
	storageAccountsClient armStorage.AccountsClient
}

func getTestClient(t *testing.T) testClient {
	client := testClient{
		subscriptionID: os.Getenv("ARM_SUBSCRIPTION_ID"),
		tenantID:       os.Getenv("ARM_TENANT_ID"),
		clientID:       os.Getenv("ARM_CLIENT_ID"),
		clientSecret:   os.Getenv("ARM_CLIENT_SECRET"),
	}

	if client.subscriptionID == "" || client.tenantID == "" || client.clientID == "" || client.clientSecret == "" {
		t.Fatal("Azure credentials missing or incomplete")
	}

	env, err := getAzureEnvironment(os.Getenv("ARM_ENVIRONMENT"))
	if err != nil {
		t.Fatalf("Failed to detect Azure environment from ARM_ENVIRONMENT value: %s", os.Getenv("ARM_ENVIRONMENT"))
	}
	client.environment = env

	oauthConfig, err := env.OAuthConfigForTenant(client.tenantID)
	if err != nil {
		t.Fatalf("Failed to get OAuth config: %s", err)
	}

	spt, err := azure.NewServicePrincipalToken(*oauthConfig, client.clientID, client.clientSecret, env.ResourceManagerEndpoint)
	if err != nil {
		t.Fatalf("Failed to create Service Principal Token: %s", err)
	}

	client.groupsClient = resources.NewGroupsClientWithBaseURI(env.ResourceManagerEndpoint, client.subscriptionID)
	client.groupsClient.Authorizer = spt

	client.storageAccountsClient = armStorage.NewAccountsClientWithBaseURI(env.ResourceManagerEndpoint, client.subscriptionID)
	client.storageAccountsClient.Authorizer = spt

	return client
}
