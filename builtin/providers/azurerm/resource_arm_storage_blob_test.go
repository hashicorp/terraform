package azurerm

import (
	"fmt"
	"testing"

	"strings"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceAzureRMStorageBlobType_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "unknown",
			ErrCount: 1,
		},
		{
			Value:    "page",
			ErrCount: 0,
		},
		{
			Value:    "blob",
			ErrCount: 0,
		},
		{
			Value:    "BLOB",
			ErrCount: 0,
		},
		{
			Value:    "Blob",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateArmStorageBlobType(tc.Value, "azurerm_storage_blob")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Storage Blob type to trigger a validation error")
		}
	}
}

func TestResourceAzureRMStorageBlobSize_validation(t *testing.T) {
	cases := []struct {
		Value    int
		ErrCount int
	}{
		{
			Value:    511,
			ErrCount: 1,
		},
		{
			Value:    512,
			ErrCount: 0,
		},
		{
			Value:    1024,
			ErrCount: 0,
		},
		{
			Value:    2048,
			ErrCount: 0,
		},
		{
			Value:    5120,
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateArmStorageBlobSize(tc.Value, "azurerm_storage_blob")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Storage Blob size to trigger a validation error")
		}
	}
}

func TestAccAzureRMStorageBlob_basic(t *testing.T) {
	ri := acctest.RandInt()
	rs := strings.ToLower(acctest.RandString(11))
	config := fmt.Sprintf(testAccAzureRMStorageBlob_basic, ri, rs)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMStorageBlobDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMStorageBlobExists("azurerm_storage_blob.test"),
				),
			},
		},
	})
}

func testCheckAzureRMStorageBlobExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		storageContainerName := rs.Primary.Attributes["storage_container_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for storage blob: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)
		blobClient, err := armClient.getBlobStorageClientForStorageAccount(resourceGroup, storageAccountName)
		if err != nil {
			return err
		}

		exists, err := blobClient.BlobExists(storageContainerName, name)
		if err != nil {
			return err
		}

		if !exists {
			return fmt.Errorf("Bad: Storage Blob %q (storage container: %q) does not exist", name, storageContainerName)
		}

		return nil
	}
}

func testCheckAzureRMStorageBlobDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_storage_blob" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		storageContainerName := rs.Primary.Attributes["storage_container_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for storage blob: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)
		blobClient, err := armClient.getBlobStorageClientForStorageAccount(resourceGroup, storageAccountName)
		if err != nil {
			return nil
		}

		exists, err := blobClient.BlobExists(storageContainerName, name)
		if err != nil {
			return nil
		}

		if exists {
			return fmt.Errorf("Bad: Storage Blob %q (storage container: %q) still exists", name, storageContainerName)
		}
	}

	return nil
}

var testAccAzureRMStorageBlob_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "westus"
}

resource "azurerm_storage_account" "test" {
    name = "acctestacc%s"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_storage_blob" "test" {
    name = "herpderp1.vhd"

    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    storage_container_name = "${azurerm_storage_container.test.name}"

    type = "page"
    size = 5120
}
`
