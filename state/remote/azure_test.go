package remote

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	mainStorage "github.com/Azure/azure-sdk-for-go/storage"
	riviera "github.com/jen20/riviera/azure"
	"github.com/jen20/riviera/storage"
)

func TestAzureClient_impl(t *testing.T) {
	var _ Client = new(AzureClient)
}

func TestAzureClient(t *testing.T) {
	// This test creates a bucket in Azure and populates it.
	// It may incur costs, so it will only run if Azure credential environment
	// variables are present.

	config := map[string]string{
		"arm_subscription_id": os.Getenv("ARM_SUBSCRIPTION_ID"),
		"arm_client_id":       os.Getenv("ARM_CLIENT_ID"),
		"arm_client_secret":   os.Getenv("ARM_CLIENT_SECRET"),
		"arm_tenant_id":       os.Getenv("ARM_TENANT_ID"),
	}

	for k, v := range config {
		if v == "" {
			t.Skipf("skipping; %s must be set", strings.ToUpper(k))
		}
	}

	config["resource_group_name"] = fmt.Sprintf("terraform-%x", time.Now().Unix())
	config["storage_account_name"] = fmt.Sprintf("terraform%x", time.Now().Unix())
	config["container_name"] = "terraform"
	config["key"] = "test.tfstate"

	setup(t, config)
	defer teardown(t, config)

	client, err := azureFactory(config)
	if err != nil {
		t.Fatalf("Error for valid config: %v", err)
	}

	testClient(t, client)
}

func setup(t *testing.T, conf map[string]string) {
	creds, err := getCredentialsFromConf(conf)
	if err != nil {
		t.Fatalf("Error getting credentials from conf: %v", err)
	}
	rivieraClient, err := getRivieraClient(creds)
	if err != nil {
		t.Fatalf("Error instantiating the riviera client: %v", err)
	}

	// Create resource group
	r := rivieraClient.NewRequest()
	r.Command = riviera.CreateResourceGroup{
		Name:     conf["resource_group_name"],
		Location: riviera.WestUS,
	}
	response, err := r.Execute()
	if err != nil {
		t.Fatalf("Error creating a resource group: %v", err)
	}
	if !response.IsSuccessful() {
		t.Fatalf("Error creating a resource group: %v", response.Error.Error())
	}

	// Create storage account
	r = rivieraClient.NewRequest()
	r.Command = storage.CreateStorageAccount{
		ResourceGroupName: conf["resource_group_name"],
		Name:              conf["storage_account_name"],
		AccountType:       riviera.String("Standard_LRS"),
		Location:          riviera.WestUS,
	}
	response, err = r.Execute()
	if err != nil {
		t.Fatalf("Error creating a storage account: %v", err)
	}
	if !response.IsSuccessful() {
		t.Fatalf("Error creating a storage account: %v", response.Error.Error())
	}

	// Create container
	accessKey, err := getStorageAccountAccessKey(conf, conf["resource_group_name"], conf["storage_account_name"])
	if err != nil {
		t.Fatalf("Error creating a storage account: %v", err)
	}
	storageClient, err := mainStorage.NewBasicClient(conf["storage_account_name"], accessKey)
	if err != nil {
		t.Fatalf("Error creating storage client for storage account %q: %s", conf["storage_account_name"], err)
	}
	blobClient := storageClient.GetBlobService()
	_, err = blobClient.CreateContainerIfNotExists(conf["container_name"], mainStorage.ContainerAccessTypePrivate)
	if err != nil {
		t.Fatalf("Couldn't create container with name %s: %s.", conf["container_name"], err)
	}
}

func teardown(t *testing.T, conf map[string]string) {
	creds, err := getCredentialsFromConf(conf)
	if err != nil {
		t.Fatalf("Error getting credentials from conf: %v", err)
	}
	rivieraClient, err := getRivieraClient(creds)
	if err != nil {
		t.Fatalf("Error instantiating the riviera client: %v", err)
	}

	r := rivieraClient.NewRequest()
	r.Command = riviera.DeleteResourceGroup{
		Name: conf["resource_group_name"],
	}
	response, err := r.Execute()
	if err != nil {
		t.Fatalf("Error deleting the resource group: %v", err)
	}
	if !response.IsSuccessful() {
		t.Fatalf("Error deleting the resource group: %v", err)
	}
}

func getRivieraClient(credentials *riviera.AzureResourceManagerCredentials) (*riviera.Client, error) {
	rivieraClient, err := riviera.NewClient(credentials)
	if err != nil {
		return nil, fmt.Errorf("Error creating Riviera client: %s", err)
	}

	request := rivieraClient.NewRequest()
	request.Command = riviera.RegisterResourceProvider{
		Namespace: "Microsoft.Storage",
	}

	response, err := request.Execute()
	if err != nil {
		return nil, fmt.Errorf("Cannot request provider registration for Azure Resource Manager: %s.", err)
	}

	if !response.IsSuccessful() {
		return nil, fmt.Errorf("Credentials for acessing the Azure Resource Manager API are likely " +
			"to be incorrect, or\n  the service principal does not have permission to use " +
			"the Azure Service Management\n  API.")
	}

	return rivieraClient, nil
}
