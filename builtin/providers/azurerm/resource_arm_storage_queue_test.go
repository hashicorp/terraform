package azurerm

import (
	"fmt"
	"testing"

	"strings"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceAzureRMStorageQueueName_Validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "testing_123",
			ErrCount: 1,
		},
		{
			Value:    "testing123-",
			ErrCount: 1,
		},
		{
			Value:    "-testing123",
			ErrCount: 1,
		},
		{
			Value:    "TestingSG",
			ErrCount: 1,
		},
		{
			Value:    acctest.RandString(256),
			ErrCount: 1,
		},
		{
			Value:    acctest.RandString(1),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateArmStorageQueueName(tc.Value, "azurerm_storage_queue")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the ARM Storage Queue Name to trigger a validation error")
		}
	}
}

func TestAccAzureRMStorageQueue_basic(t *testing.T) {
	ri := acctest.RandInt()
	rs := strings.ToLower(acctest.RandString(11))
	config := fmt.Sprintf(testAccAzureRMStorageQueue_basic, ri, rs, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMStorageQueueDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMStorageQueueExists("azurerm_storage_queue.test"),
				),
			},
		},
	})
}

func testCheckAzureRMStorageQueueExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for storage queue: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)
		queueClient, accountExists, err := armClient.getQueueServiceClientForStorageAccount(resourceGroup, storageAccountName)
		if err != nil {
			return err
		}
		if !accountExists {
			return fmt.Errorf("Bad: Storage Account %q does not exist", storageAccountName)
		}

		queueReference := queueClient.GetQueueReference(name)
		exists, err := queueReference.Exists()
		if err != nil {
			return err
		}

		if !exists {
			return fmt.Errorf("Bad: Storage Queue %q (storage account: %q) does not exist", name, storageAccountName)
		}

		return nil
	}
}

func testCheckAzureRMStorageQueueDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_storage_queue" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for storage queue: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)
		queueClient, accountExists, err := armClient.getQueueServiceClientForStorageAccount(resourceGroup, storageAccountName)
		if err != nil {
			return nil
		}
		if !accountExists {
			return nil
		}

		queueReference := queueClient.GetQueueReference(name)
		exists, err := queueReference.Exists()
		if err != nil {
			return nil
		}

		if exists {
			return fmt.Errorf("Bad: Storage Queue %q (storage account: %q) still exists", name, storageAccountName)
		}
	}

	return nil
}

var testAccAzureRMStorageQueue_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
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

resource "azurerm_storage_queue" "test" {
    name = "mysamplequeue-%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
}
`
