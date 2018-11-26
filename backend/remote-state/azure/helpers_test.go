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
	armStorage "github.com/Azure/azure-sdk-for-go/profiles/2017-03-09/storage/mgmt/storage"
	"github.com/Azure/azure-sdk-for-go/storage"
	sasStorage "github.com/hashicorp/go-azure-helpers/storage"
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

	armClient, err := buildArmClient(BackendConfig{
		SubscriptionID:                subscriptionID,
		TenantID:                      tenantID,
		ClientID:                      clientID,
		ClientSecret:                  clientSecret,
		CustomResourceManagerEndpoint: endpoint,
		Environment:                   environment,
		ResourceGroupName:             res.resourceGroup,
		StorageAccountName:            res.storageAccountName,
		UseMsi:                        msiEnabled,
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

	sasToken, err := sasStorage.ComputeSASToken(accountName, accessKey, permissions, services, resourceTypes,
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
}

func testResourceNames(rString string, keyName string) resourceNames {
	return resourceNames{
		resourceGroup:        fmt.Sprintf("acctestrg-backend-%s", rString),
		location:             os.Getenv("ARM_LOCATION"),
		storageAccountName:   fmt.Sprintf("acctestsa%s", rString),
		storageContainerName: "acctestcont",
		storageKeyName:       keyName,
	}
}

func (c *ArmClient) buildTestResources(ctx context.Context, names *resourceNames) error {
	log.Printf("Creating Resource Group %q", names.resourceGroup)
	_, err := c.groupsClient.CreateOrUpdate(ctx, names.resourceGroup, resources.Group{Location: &names.location})
	if err != nil {
		return fmt.Errorf("failed to create test resource group: %s", err)
	}

	log.Printf("Creating Storage Account %q in Resource Group %q", names.storageAccountName, names.resourceGroup)
	future, err := c.storageAccountsClient.Create(ctx, names.resourceGroup, names.storageAccountName, armStorage.AccountCreateParameters{
		Sku: &armStorage.Sku{
			Name: armStorage.StandardLRS,
			Tier: armStorage.Standard,
		},
		Location: &names.location,
	})
	if err != nil {
		return fmt.Errorf("failed to create test storage account: %s", err)
	}

	err = future.WaitForCompletionRef(ctx, c.storageAccountsClient.Client)
	if err != nil {
		return fmt.Errorf("failed waiting for the creation of storage account: %s", err)
	}

	log.Printf("fetching access key for storage account")
	resp, err := c.storageAccountsClient.ListKeys(ctx, names.resourceGroup, names.storageAccountName)
	if err != nil {
		return fmt.Errorf("failed to list storage account keys %s:", err)
	}

	keys := *resp.Keys
	accessKey := *keys[0].Value
	names.storageAccountAccessKey = accessKey

	storageClient, err := storage.NewBasicClientOnSovereignCloud(names.storageAccountName, accessKey, c.environment)
	if err != nil {
		return fmt.Errorf("failed to list storage account keys %s:", err)
	}

	log.Printf("Creating Container %q in Storage Account %q (Resource Group %q)", names.storageContainerName, names.storageAccountName, names.resourceGroup)
	blobService := storageClient.GetBlobService()
	container := blobService.GetContainerReference(names.storageContainerName)
	err = container.Create(&storage.CreateContainerOptions{})
	if err != nil {
		return fmt.Errorf("failed to create storage container: %s", err)
	}

	return nil
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
