package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMKeyVault_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMKeyVault_basic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMKeyVaultDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMKeyVaultExists("azurerm_key_vault.test"),
				),
			},
		},
	})
}

func TestAccAzureRMKeyVault_update(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMKeyVault_basic, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMKeyVault_update, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMKeyVaultDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMKeyVaultExists("azurerm_key_vault.test"),
					resource.TestCheckResourceAttr("azurerm_key_vault.test", "access_policy.0.key_permissions.0", "all"),
					resource.TestCheckResourceAttr("azurerm_key_vault.test", "access_policy.0.secret_permissions.0", "all"),
					resource.TestCheckResourceAttr("azurerm_key_vault.test", "tags.environment", "Production"),
				),
			},
			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("azurerm_key_vault.test", "access_policy.0.key_permissions.0", "get"),
					resource.TestCheckResourceAttr("azurerm_key_vault.test", "access_policy.0.secret_permissions.0", "get"),
					resource.TestCheckResourceAttr("azurerm_key_vault.test", "enabled_for_deployment", "true"),
					resource.TestCheckResourceAttr("azurerm_key_vault.test", "enabled_for_disk_encryption", "true"),
					resource.TestCheckResourceAttr("azurerm_key_vault.test", "enabled_for_template_deployment", "true"),
					resource.TestCheckResourceAttr("azurerm_key_vault.test", "tags.environment", "Staging"),
				),
			},
		},
	})
}

func testCheckAzureRMKeyVaultDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ArmClient).keyVaultClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_key_vault" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := client.Get(resourceGroup, name)
		if err != nil {
			if resp.StatusCode == http.StatusNotFound {
				return nil
			}
			return err
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Key Vault still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

func testCheckAzureRMKeyVaultExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		vaultName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for vault: %s", vaultName)
		}

		client := testAccProvider.Meta().(*ArmClient).keyVaultClient

		resp, err := client.Get(resourceGroup, vaultName)
		if err != nil {
			return fmt.Errorf("Bad: Get on keyVaultClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Vault %q (resource group: %q) does not exist", vaultName, resourceGroup)
		}

		return nil
	}
}

var testAccAzureRMKeyVault_basic = `
data "azurerm_client_config" "current" {}

resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_key_vault" "test" {
    name = "vault%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
	tenant_id = "${data.azurerm_client_config.current.tenant_id}"

    sku {
		name = "premium"
	}

	access_policy {
		tenant_id = "${data.azurerm_client_config.current.tenant_id}"
		object_id = "${data.azurerm_client_config.current.client_id}"

		key_permissions = [
			"all"
		]

		secret_permissions = [
			"all"
		]
	}

	tags {
		environment = "Production"
	}
}
`

var testAccAzureRMKeyVault_update = `
data "azurerm_client_config" "current" {}

resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_key_vault" "test" {
    name = "vault%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
	tenant_id = "${data.azurerm_client_config.current.tenant_id}"

    sku {
		name = "premium"
	}

	access_policy {
		tenant_id = "${data.azurerm_client_config.current.tenant_id}"
		object_id = "${data.azurerm_client_config.current.client_id}"

		key_permissions = [
			"get"
		]

		secret_permissions = [
			"get"
		]
	}

	enabled_for_deployment = true
	enabled_for_disk_encryption = true
	enabled_for_template_deployment = true

	tags {
		environment = "Staging"
	}
}
`
