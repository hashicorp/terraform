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
	"github.com/Azure/go-autorest/autorest"
	sasStorage "github.com/hashicorp/go-azure-helpers/storage"

	keyvaultKey "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"
	"github.com/tombuildsstuff/giovanni/storage/2018-11-09/blob/containers"
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

func getDefaultBackendConfig(t *testing.T, resourceGroup, storageAccountName, keyVaultKeyIdentifier string) *BackendConfig {
	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	tenantID := os.Getenv("ARM_TENANT_ID")
	clientID := os.Getenv("ARM_CLIENT_ID")
	objectID := os.Getenv("ARM_OBJECT_ID")
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

	if objectID == "" {
		t.Fatalf("Missing ARM_OBJECT_ID")
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

	backend := &BackendConfig{
		SubscriptionID:                subscriptionID,
		TenantID:                      tenantID,
		ClientID:                      clientID,
		ClientSecret:                  clientSecret,
		CustomResourceManagerEndpoint: endpoint,
		Environment:                   environment,
		ResourceGroupName:             resourceGroup,
		StorageAccountName:            storageAccountName,
		KeyVaultKeyIdentifier:         keyVaultKeyIdentifier,
		UseMsi:                        msiEnabled,
	}

	return backend
}

func buildTestClient(t *testing.T, res resourceNames) *ArmClient {
	backendConfig := getDefaultBackendConfig(t, res.resourceGroup, res.storageAccountName, "")

	armClient, err := buildArmClient(*backendConfig)
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
	keyVaultName            string
	keyVaultKeyName         string
}

func testResourceNames(rString string, keyName string) resourceNames {
	return resourceNames{
		resourceGroup:        fmt.Sprintf("acctestRG-backend-%s-%s", strings.Replace(time.Now().Local().Format("060102150405.00"), ".", "", 1), rString),
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

	storageAuth, err := autorest.NewSharedKeyAuthorizer(names.storageAccountName, accessKey, autorest.SharedKey)
	if err != nil {
		return fmt.Errorf("Error building Authorizer: %+v", err)
	}

	containersClient := containers.NewWithEnvironment(c.environment)
	containersClient.Client.Authorizer = storageAuth

	log.Printf("Creating Container %q in Storage Account %q (Resource Group %q)", names.storageContainerName, names.storageAccountName, names.resourceGroup)
	_, err = containersClient.Create(ctx, names.storageAccountName, names.storageContainerName, containers.CreateInput{})
	if err != nil {
		return fmt.Errorf("failed to create storage container: %s", err)
	}

	// Azure Key Vault
	if names.keyVaultName != "" && names.keyVaultKeyName != "" {
		vaultsClient := keyvault.NewVaultsClient(os.Getenv("ARM_SUBSCRIPTION_ID"))
		vaultsClient.Authorizer = c.groupsClient.Authorizer
		vaultsClient.AddToUserAgent(c.groupsClient.UserAgent)

		tenantID, err := uuid.FromString(os.Getenv("ARM_TENANT_ID"))
		if err != nil {
			return err
		}

		apList := []keyvault.AccessPolicyEntry{}
		ap := keyvault.AccessPolicyEntry{
			TenantID: &tenantID,
			Permissions: &keyvault.Permissions{
				Keys: &[]keyvault.KeyPermissions{
					keyvault.KeyPermissionsGet,
					keyvault.KeyPermissionsCreate,
					keyvault.KeyPermissionsList,
					keyvault.KeyPermissionsEncrypt,
					keyvault.KeyPermissionsDecrypt,
				},
			},
		}

		ap.ObjectID = to.StringPtr(os.Getenv("ARM_OBJECT_ID"))
		apList = append(apList, ap)

		log.Printf("creating Key Vault https://%s.vault.azure.net/", names.keyVaultName)
		vault, err := vaultsClient.CreateOrUpdate(
			ctx,
			names.resourceGroup,
			names.keyVaultName,
			keyvault.VaultCreateOrUpdateParameters{
				Location: &names.location,
				Properties: &keyvault.VaultProperties{
					TenantID: &tenantID,
					Sku: &keyvault.Sku{
						Family: to.StringPtr("A"),
						Name:   keyvault.Standard,
					},
					AccessPolicies: &apList,
				},
			},
		)

		if err != nil {
			log.Printf("failed to create Key Vault https://%s.vault.azure.net/", names.keyVaultName)
			return err
		}

		log.Printf("creating key inside Key Vault")

		vaultURL := *vault.Properties.VaultURI
		key, err := c.encClient.KvClient.CreateKey(
			ctx,
			vaultURL,
			names.keyVaultKeyName,
			keyvaultKey.KeyCreateParameters{
				KeyAttributes: &keyvaultKey.KeyAttributes{
					Enabled: to.BoolPtr(true),
				},
				KeySize: to.Int32Ptr(2048),
				KeyOps: &[]keyvaultKey.JSONWebKeyOperation{
					keyvaultKey.Encrypt,
					keyvaultKey.Decrypt,
				},
				Kty: keyvaultKey.RSA,
			})

		if err != nil {
			log.Printf("failed to create Key")
			return err
		}

		log.Printf("success at creating key. id: %s", *key.Key.Kid)
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

func testResourceNamesWithKeyVault(rString, storageKeyName, keyVaultName, keyVaultKeyName string) resourceNames {
	resources := testResourceNames(rString, storageKeyName)
	resources.keyVaultName = keyVaultName
	resources.keyVaultKeyName = keyVaultKeyName

	return resources
}

func buildTestClientWithKeyVault(t *testing.T, res resourceNames) *ArmClient {
	keyIdentifier := fmt.Sprintf("https://%s.vault.azure.net/keys/%s", res.keyVaultName, res.keyVaultKeyName)

	backendConfig := getDefaultBackendConfig(t, res.resourceGroup, res.storageAccountName, keyIdentifier)

	armClient, err := buildArmClient(*backendConfig)
	if err != nil {
		t.Fatalf("Failed to build ArmClient: %+v", err)
	}

	return armClient
}
