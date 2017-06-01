package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMLocalNetworkGateway_basic(t *testing.T) {
	name := "azurerm_local_network_gateway.test"

	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLocalNetworkGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLocalNetworkGatewayConfig_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLocalNetworkGatewayExists(name),
					resource.TestCheckResourceAttr(name, "gateway_address", "127.0.0.1"),
					resource.TestCheckResourceAttr(name, "address_space.0", "127.0.0.0/8"),
				),
			},
		},
	})
}

func TestAccAzureRMLocalNetworkGateway_disappears(t *testing.T) {
	name := "azurerm_local_network_gateway.test"
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLocalNetworkGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLocalNetworkGatewayConfig_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLocalNetworkGatewayExists(name),
					resource.TestCheckResourceAttr(name, "gateway_address", "127.0.0.1"),
					resource.TestCheckResourceAttr(name, "address_space.0", "127.0.0.0/8"),
					testCheckAzureRMLocalNetworkGatewayDisappears(name),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// testCheckAzureRMLocalNetworkGatewayExists returns the resurce.TestCheckFunc
// which checks whether or not the expected local network gateway exists both
// in the schema, and on Azure.
func testCheckAzureRMLocalNetworkGatewayExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// first check within the schema for the local network gateway:
		res, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Local network gateway '%s' not found.", name)
		}

		// then, extract the name and the resource group:
		id, err := parseAzureResourceID(res.Primary.ID)
		if err != nil {
			return err
		}
		localNetName := id.Path["localNetworkGateways"]
		resGrp := id.ResourceGroup

		// and finally, check that it exists on Azure:
		lnetClient := testAccProvider.Meta().(*ArmClient).localNetConnClient

		resp, err := lnetClient.Get(resGrp, localNetName)
		if err != nil {
			if resp.StatusCode == http.StatusNotFound {
				return fmt.Errorf("Local network gateway '%s' (resource group '%s') does not exist on Azure.", localNetName, resGrp)
			}
			return fmt.Errorf("Error reading the state of local network gateway '%s'.", localNetName)
		}

		return nil
	}
}

func testCheckAzureRMLocalNetworkGatewayDisappears(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// first check within the schema for the local network gateway:
		res, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Local network gateway '%s' not found.", name)
		}

		// then, extract the name and the resource group:
		id, err := parseAzureResourceID(res.Primary.ID)
		if err != nil {
			return err
		}
		localNetName := id.Path["localNetworkGateways"]
		resGrp := id.ResourceGroup

		// and finally, check that it exists on Azure:
		lnetClient := testAccProvider.Meta().(*ArmClient).localNetConnClient

		deleteResp, error := lnetClient.Delete(resGrp, localNetName, make(chan struct{}))
		resp := <-deleteResp
		err = <-error
		if err != nil {
			if resp.StatusCode == http.StatusNotFound {
				return fmt.Errorf("Local network gateway '%s' (resource group '%s') does not exist on Azure.", localNetName, resGrp)
			}
			return fmt.Errorf("Error deleting the state of local network gateway '%s'.", localNetName)
		}

		return nil
	}
}

func testCheckAzureRMLocalNetworkGatewayDestroy(s *terraform.State) error {
	for _, res := range s.RootModule().Resources {
		if res.Type != "azurerm_local_network_gateway" {
			continue
		}

		id, err := parseAzureResourceID(res.Primary.ID)
		if err != nil {
			return err
		}
		localNetName := id.Path["localNetworkGateways"]
		resGrp := id.ResourceGroup

		lnetClient := testAccProvider.Meta().(*ArmClient).localNetConnClient
		resp, err := lnetClient.Get(resGrp, localNetName)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Local network gateway still exists:\n%#v", resp.LocalNetworkGatewayPropertiesFormat)
		}
	}

	return nil
}

func testAccAzureRMLocalNetworkGatewayConfig_basic(rInt int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctest-%d"
    location = "West US"
}

resource "azurerm_local_network_gateway" "test" {
	name = "acctestlng-%d"
	location = "${azurerm_resource_group.test.location}"
	resource_group_name = "${azurerm_resource_group.test.name}"
	gateway_address = "127.0.0.1"
	address_space = ["127.0.0.0/8"]
}
`, rInt, rInt)
}
