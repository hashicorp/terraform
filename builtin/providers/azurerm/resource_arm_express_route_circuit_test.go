package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMExpressRouteCircuit_basic(t *testing.T) {
	var erc network.ExpressRouteCircuit
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMExpressRouteCircuitDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMExpressRouteCircuit_basic(ri),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMExpressRouteCircuitExists("azurerm_express_route_circuit.test", &erc),
				),
			},
		},
	})
}

func testCheckAzureRMExpressRouteCircuitExists(name string, erc *network.ExpressRouteCircuit) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		expressRouteCircuitName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for Express Route Circuit: %s", expressRouteCircuitName)
		}

		conn := testAccProvider.Meta().(*ArmClient).expressRouteCircuitClient

		resp, err := conn.Get(resourceGroup, expressRouteCircuitName)
		if err != nil {
			if resp.StatusCode == http.StatusNotFound {
				return fmt.Errorf("Bad: Express Route Circuit %q (resource group: %q) does not exist", expressRouteCircuitName, resourceGroup)
			}

			return fmt.Errorf("Bad: Get on expressRouteCircuitClient: %s", err)
		}

		*erc = resp

		return nil
	}
}

func testCheckAzureRMExpressRouteCircuitDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).expressRouteCircuitClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_express_route_circuit" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Express Route Circuit still exists:\n%#v", resp.ExpressRouteCircuitPropertiesFormat)
		}
	}

	return nil
}

func testAccAzureRMExpressRouteCircuit_basic(rInt int) string {
	return fmt.Sprintf(`
		resource "azurerm_resource_group" "test" {
			name = "acctestrg-%d"
			location = "West US"
		}

		resource "azurerm_express_route_circuit" "test" {
			name = "acctest-erc-%[1]d"
			location = "West US"
			resource_group_name = "${azurerm_resource_group.test.name}"
			service_provider_name = "Equinix"
			peering_location = "Silicon Valley"
			bandwidth_in_mbps = 50
			sku {
			    tier = "Standard"
			    family = "MeteredData"
		    }
			allow_classic_operations = false

			tags {
				Environment = "production"
				Purpose = "AcceptanceTests"
			}
		}`, rInt)
}
