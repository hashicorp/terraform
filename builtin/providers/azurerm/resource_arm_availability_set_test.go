package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMAvailabilitySet_basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMAvailabilitySetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMVAvailabilitySet_basic,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMAvailabilitySetExists("azurerm_availability_set.test"),
					resource.TestCheckResourceAttr(
						"azurerm_availability_set.test", "name", "acceptanceTestAvailabilitySet1"),
					resource.TestCheckResourceAttr(
						"azurerm_availability_set.test", "platform_update_domain_count", "5"),
					resource.TestCheckResourceAttr(
						"azurerm_availability_set.test", "platform_fault_domain_count", "3"),
				),
			},
		},
	})
}

func TestAccAzureRMAvailabilitySet_withDomainCounts(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMAvailabilitySetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMVAvailabilitySet_withDomainCounts,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMAvailabilitySetExists("azurerm_availability_set.test"),
					resource.TestCheckResourceAttr(
						"azurerm_availability_set.test", "name", "acceptanceTestAvailabilitySet1"),
					resource.TestCheckResourceAttr(
						"azurerm_availability_set.test", "platform_update_domain_count", "10"),
					resource.TestCheckResourceAttr(
						"azurerm_availability_set.test", "platform_fault_domain_count", "1"),
				),
			},
		},
	})
}

func testCheckAzureRMAvailabilitySetExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		availSetName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for availability set: %s", availSetName)
		}

		conn := testAccProvider.Meta().(*ArmClient).availSetClient

		resp, err := conn.Get(resourceGroup, availSetName)
		if err != nil {
			return fmt.Errorf("Bad: Get on availSetClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Availability Set %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMAvailabilitySetDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).vnetClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_availability_set" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Availability Set still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

var testAccAzureRMVAvailabilitySet_basic = `
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}
resource "azurerm_availability_set" "test" {
    name = "acceptanceTestAvailabilitySet1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
`

var testAccAzureRMVAvailabilitySet_withDomainCounts = `
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}
resource "azurerm_availability_set" "test" {
    name = "acceptanceTestAvailabilitySet1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    platform_update_domain_count = 10
    platform_fault_domain_count = 1
}
`
