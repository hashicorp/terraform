package azurerm

import (
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMStorageShare_basic(t *testing.T) {
	var sS storage.Share

	ri := acctest.RandInt()
	rs := strings.ToLower(acctest.RandString(11))
	config := fmt.Sprintf(testAccAzureRMStorageShare_basic, ri, rs)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMStorageShareDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMStorageShareExists("azurerm_storage_share.test", &sS),
				),
			},
		},
	})
}

func TestAccAzureRMStorageShare_disappears(t *testing.T) {
	var sS storage.Share

	ri := acctest.RandInt()
	rs := strings.ToLower(acctest.RandString(11))
	config := fmt.Sprintf(testAccAzureRMStorageShare_basic, ri, rs)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMStorageShareDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMStorageShareExists("azurerm_storage_share.test", &sS),
					testAccARMStorageShareDisappears("azurerm_storage_share.test", &sS),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testCheckAzureRMStorageShareExists(name string, sS *storage.Share) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		resourceGroupName, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for share: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)
		fileClient, accountExists, err := armClient.getFileServiceClientForStorageAccount(resourceGroupName, storageAccountName)
		if err != nil {
			return err
		}
		if !accountExists {
			return fmt.Errorf("Bad: Storage Account %q does not exist", storageAccountName)
		}

		shares, err := fileClient.ListShares(storage.ListSharesParameters{
			Prefix:  name,
			Timeout: 90,
		})

		if len(shares.Shares) == 0 {
			return fmt.Errorf("Bad: Share %q (storage account: %q) does not exist", name, storageAccountName)
		}

		var found bool
		for _, share := range shares.Shares {
			if share.Name == name {
				found = true
				*sS = share
			}
		}

		if !found {
			return fmt.Errorf("Bad: Share %q (storage account: %q) does not exist", name, storageAccountName)
		}

		return nil
	}
}

func testAccARMStorageShareDisappears(name string, sS *storage.Share) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)

		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		resourceGroupName, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for storage share: %s", sS.Name)
		}

		fileClient, accountExists, err := armClient.getFileServiceClientForStorageAccount(resourceGroupName, storageAccountName)
		if err != nil {
			return err
		}
		if !accountExists {
			log.Printf("[INFO]Storage Account %q doesn't exist so the share won't exist", storageAccountName)
			return nil
		}

		reference := fileClient.GetShareReference(sS.Name)
		options := &storage.FileRequestOptions{}
		err = reference.Create(options)

		if _, err = reference.DeleteIfExists(options); err != nil {
			return fmt.Errorf("Error deleting storage file %q: %s", name, err)
		}

		return nil
	}
}

func testCheckAzureRMStorageShareDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_storage_share" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		resourceGroupName, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for share: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)
		fileClient, accountExists, err := armClient.getFileServiceClientForStorageAccount(resourceGroupName, storageAccountName)
		if err != nil {
			//If we can't get keys then the blob can't exist
			return nil
		}
		if !accountExists {
			return nil
		}

		shares, err := fileClient.ListShares(storage.ListSharesParameters{
			Prefix:  name,
			Timeout: 90,
		})

		if err != nil {
			return nil
		}

		var found bool
		for _, share := range shares.Shares {
			if share.Name == name {
				found = true
			}
		}

		if found {
			return fmt.Errorf("Bad: Share %q (storage account: %q) still exists", name, storageAccountName)
		}
	}

	return nil
}

func TestValidateArmStorageShareName(t *testing.T) {
	validNames := []string{
		"valid-name",
		"valid02-name",
	}
	for _, v := range validNames {
		_, errors := validateArmStorageShareName(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Share Name: %q", v, errors)
		}
	}

	invalidNames := []string{
		"InvalidName1",
		"-invalidname1",
		"invalid_name",
		"invalid!",
		"double-hyphen--invalid",
		"ww",
		strings.Repeat("w", 65),
	}
	for _, v := range invalidNames {
		_, errors := validateArmStorageShareName(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid Share Name", v)
		}
	}
}

var testAccAzureRMStorageShare_basic = `
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

resource "azurerm_storage_share" "test" {
    name = "testshare"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
}
`
