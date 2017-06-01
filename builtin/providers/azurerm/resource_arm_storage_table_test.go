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

func TestAccAzureRMStorageTable_basic(t *testing.T) {
	var table storage.Table

	ri := acctest.RandInt()
	rs := strings.ToLower(acctest.RandString(11))
	config := fmt.Sprintf(testAccAzureRMStorageTable_basic, ri, rs, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMStorageTableDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMStorageTableExists("azurerm_storage_table.test", &table),
				),
			},
		},
	})
}

func TestAccAzureRMStorageTable_disappears(t *testing.T) {
	var table storage.Table

	ri := acctest.RandInt()
	rs := strings.ToLower(acctest.RandString(11))
	config := fmt.Sprintf(testAccAzureRMStorageTable_basic, ri, rs, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMStorageTableDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMStorageTableExists("azurerm_storage_table.test", &table),
					testAccARMStorageTableDisappears("azurerm_storage_table.test", &table),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testCheckAzureRMStorageTableExists(name string, t *storage.Table) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for storage table: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)
		tableClient, accountExists, err := armClient.getTableServiceClientForStorageAccount(resourceGroup, storageAccountName)
		if err != nil {
			return err
		}
		if !accountExists {
			return fmt.Errorf("Bad: Storage Account %q does not exist", storageAccountName)
		}

		options := &storage.QueryTablesOptions{}
		tables, err := tableClient.QueryTables(storage.MinimalMetadata, options)

		if len(tables.Tables) == 0 {
			return fmt.Errorf("Bad: Storage Table %q (storage account: %q) does not exist", name, storageAccountName)
		}

		var found bool
		for _, table := range tables.Tables {
			if table.Name == name {
				found = true
				*t = table
			}
		}

		if !found {
			return fmt.Errorf("Bad: Storage Table %q (storage account: %q) does not exist", name, storageAccountName)
		}

		return nil
	}
}

func testAccARMStorageTableDisappears(name string, t *storage.Table) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)

		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for storage table: %s", t.Name)
		}

		tableClient, accountExists, err := armClient.getTableServiceClientForStorageAccount(resourceGroup, storageAccountName)
		if err != nil {
			return err
		}
		if !accountExists {
			log.Printf("[INFO]Storage Account %q doesn't exist so the table won't exist", storageAccountName)
			return nil
		}

		table := tableClient.GetTableReference(t.Name)
		timeout := uint(60)
		options := &storage.TableOptions{}
		err = table.Delete(timeout, options)
		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckAzureRMStorageTableDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_storage_table" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for storage table: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)
		tableClient, accountExists, err := armClient.getTableServiceClientForStorageAccount(resourceGroup, storageAccountName)
		if err != nil {
			//If we can't get keys then the table can't exist
			return nil
		}
		if !accountExists {
			return nil
		}

		options := &storage.QueryTablesOptions{}
		tables, err := tableClient.QueryTables(storage.NoMetadata, options)

		if err != nil {
			return nil
		}

		var found bool
		for _, table := range tables.Tables {
			if table.Name == name {
				found = true
			}
		}

		if found {
			return fmt.Errorf("Bad: Storage Table %q (storage account: %q) still exist", name, storageAccountName)
		}
	}

	return nil
}

func TestValidateArmStorageTableName(t *testing.T) {
	validNames := []string{
		"mytable01",
		"mytable",
		"myTable",
		"MYTABLE",
	}
	for _, v := range validNames {
		_, errors := validateArmStorageTableName(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Storage Table Name: %q", v, errors)
		}
	}

	invalidNames := []string{
		"table",
		"-invalidname1",
		"invalid_name",
		"invalid!",
		"ww",
		strings.Repeat("w", 65),
	}
	for _, v := range invalidNames {
		_, errors := validateArmStorageTableName(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid Storage Table Name", v)
		}
	}
}

var testAccAzureRMStorageTable_basic = `
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

resource "azurerm_storage_table" "test" {
    name = "tfacceptancetest%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
}
`
