package azurerm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMStorageContainer_basic(t *testing.T) {
	ri := acctest.RandInt()
	rs := strings.ToLower(acctest.RandString(11))
	config := fmt.Sprintf(testAccAzureRMStorageContainer_basic, ri, rs)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMStorageContainerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMStorageContainerExists("azurerm_storage_container.test"),
				),
			},
		},
	})
}

func testCheckAzureRMStorageContainerExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for storage container: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)
		blobClient, err := armClient.getBlobStorageClientForStorageAccount(resourceGroup, storageAccountName)
		if err != nil {
			return err
		}

		containers, err := blobClient.ListContainers(storage.ListContainersParameters{
			Prefix:  name,
			Timeout: 90,
		})

		if len(containers.Containers) == 0 {
			return fmt.Errorf("Bad: Storage Container %q (storage account: %q) does not exist", name, storageAccountName)
		}

		var found bool
		for _, container := range containers.Containers {
			if container.Name == name {
				found = true
			}
		}

		if !found {
			return fmt.Errorf("Bad: Storage Container %q (storage account: %q) does not exist", name, storageAccountName)
		}

		return nil
	}
}

func testCheckAzureRMStorageContainerDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_storage_container" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for storage container: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)
		blobClient, err := armClient.getBlobStorageClientForStorageAccount(resourceGroup, storageAccountName)
		if err != nil {
			//If we can't get keys then the blob can't exist
			return nil
		}

		containers, err := blobClient.ListContainers(storage.ListContainersParameters{
			Prefix:  name,
			Timeout: 90,
		})

		if err != nil {
			return nil
		}

		var found bool
		for _, container := range containers.Containers {
			if container.Name == name {
				found = true
			}
		}

		if found {
			return fmt.Errorf("Bad: Storage Container %q (storage account: %q) still exist", name, storageAccountName)
		}
	}

	return nil
}

var testAccAzureRMStorageContainer_basic = `
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
`
