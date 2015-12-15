package azurerm

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/core/http"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMResourceGroup_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMResourceGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMResourceGroup_basic,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMResourceGroupExists("azurerm_resource_group.test"),
				),
			},
		},
	})
}

func testCheckAzureRMResourceGroupExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		resourceGroup := rs.Primary.Attributes["name"]

		// Ensure resource group exists in API
		conn := testAccProvider.Meta().(*ArmClient).resourceGroupClient

		resp, err := conn.Get(resourceGroup)
		if err != nil {
			return fmt.Errorf("Bad: Get on resourceGroupClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Virtual Network %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMResourceGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).resourceGroupClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_resource_group" {
			continue
		}

		resourceGroup := rs.Primary.ID

		resp, err := conn.Get(resourceGroup)
		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Resource Group still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

var testAccAzureRMResourceGroup_basic = `
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1_basic"
    location = "West US"
}
`
