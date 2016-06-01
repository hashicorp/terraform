package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestValidateArmStorageAccountType(t *testing.T) {
	testCases := []struct {
		input       string
		shouldError bool
	}{
		{"standard_lrs", false},
		{"invalid", true},
	}

	for _, test := range testCases {
		_, es := validateArmStorageAccountType(test.input, "account_type")

		if test.shouldError && len(es) == 0 {
			t.Fatalf("Expected validating account_type %q to fail", test.input)
		}
	}
}

func TestValidateArmStorageAccountName(t *testing.T) {
	testCases := []struct {
		input       string
		shouldError bool
	}{
		{"ab", true},
		{"ABC", true},
		{"abc", false},
		{"123456789012345678901234", false},
		{"1234567890123456789012345", true},
		{"abc12345", false},
	}

	for _, test := range testCases {
		_, es := validateArmStorageAccountName(test.input, "name")

		if test.shouldError && len(es) == 0 {
			t.Fatalf("Expected validating name %q to fail", test.input)
		}
	}
}

func TestAccAzureRMStorageAccount_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMStorageAccountDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMStorageAccount_basic,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMStorageAccountExists("azurerm_storage_account.testsa"),
					resource.TestCheckResourceAttr("azurerm_storage_account.testsa", "account_type", "Standard_LRS"),
					resource.TestCheckResourceAttr("azurerm_storage_account.testsa", "tags.#", "1"),
					resource.TestCheckResourceAttr("azurerm_storage_account.testsa", "tags.environment", "production"),
				),
			},

			resource.TestStep{
				Config: testAccAzureRMStorageAccount_update,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMStorageAccountExists("azurerm_storage_account.testsa"),
					resource.TestCheckResourceAttr("azurerm_storage_account.testsa", "account_type", "Standard_GRS"),
					resource.TestCheckResourceAttr("azurerm_storage_account.testsa", "tags.#", "1"),
					resource.TestCheckResourceAttr("azurerm_storage_account.testsa", "tags.environment", "staging"),
				),
			},
		},
	})
}

func testCheckAzureRMStorageAccountExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		storageAccount := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		// Ensure resource group exists in API
		conn := testAccProvider.Meta().(*ArmClient).storageServiceClient

		resp, err := conn.GetProperties(resourceGroup, storageAccount)
		if err != nil {
			return fmt.Errorf("Bad: Get on storageServiceClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: StorageAccount %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMStorageAccountDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).storageServiceClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_storage_account" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.GetProperties(resourceGroup, name)
		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Storage Account still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

var testAccAzureRMStorageAccount_basic = `
resource "azurerm_resource_group" "testrg" {
    name = "testAccAzureRMStorageAccountBasic"
    location = "westus"
}

resource "azurerm_storage_account" "testsa" {
    name = "unlikely23exst2acct1435"
    resource_group_name = "${azurerm_resource_group.testrg.name}"

    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "production"
    }
}`

var testAccAzureRMStorageAccount_update = `
resource "azurerm_resource_group" "testrg" {
    name = "testAccAzureRMStorageAccountBasic"
    location = "westus"
}

resource "azurerm_storage_account" "testsa" {
    name = "unlikely23exst2acct1435"
    resource_group_name = "${azurerm_resource_group.testrg.name}"

    location = "westus"
    account_type = "Standard_GRS"

    tags {
        environment = "staging"
    }
}`
