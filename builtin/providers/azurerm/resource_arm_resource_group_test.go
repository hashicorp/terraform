package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMResourceGroup_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMResourceGroup_basic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMResourceGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMResourceGroupExists("azurerm_resource_group.test"),
				),
			},
		},
	})
}

func TestAccAzureRMResourceGroup_disappears(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMResourceGroup_basic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMResourceGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMResourceGroupExists("azurerm_resource_group.test"),
					testCheckAzureRMResourceGroupDisappears("azurerm_resource_group.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAzureRMResourceGroup_withTags(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMResourceGroup_withTags, ri)
	postConfig := fmt.Sprintf(testAccAzureRMResourceGroup_withTagsUpdated, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMResourceGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMResourceGroupExists("azurerm_resource_group.test"),
					resource.TestCheckResourceAttr(
						"azurerm_resource_group.test", "tags.%", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_resource_group.test", "tags.environment", "Production"),
					resource.TestCheckResourceAttr(
						"azurerm_resource_group.test", "tags.cost_center", "MSFT"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMResourceGroupExists("azurerm_resource_group.test"),
					resource.TestCheckResourceAttr(
						"azurerm_resource_group.test", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_resource_group.test", "tags.environment", "staging"),
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

func testCheckAzureRMResourceGroupDisappears(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		resourceGroup := rs.Primary.Attributes["name"]

		// Ensure resource group exists in API
		conn := testAccProvider.Meta().(*ArmClient).resourceGroupClient

		_, error := conn.Delete(resourceGroup, make(chan struct{}))
		err := <-error
		if err != nil {
			return fmt.Errorf("Bad: Delete on resourceGroupClient: %s", err)
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
    name = "acctestRG-%d"
    location = "West US"
}
`

var testAccAzureRMResourceGroup_withTags = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"

    tags {
		environment = "Production"
		cost_center = "MSFT"
    }
}
`

var testAccAzureRMResourceGroup_withTagsUpdated = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"

    tags {
	environment = "staging"
    }
}
`
