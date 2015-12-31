package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccArmPublicIPAddress(t *testing.T) {
	name := "azurerm_public_ip.test"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAccArmPublicIPDeleted,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMPublicIPAddress,
				Check: resource.ComposeTestCheckFunc(
					testCheckAccArmPublicIPExists(name),
					resource.TestCheckResourceAttr(name, "name", "acceptanceTestPublicIPAddress1"),
					resource.TestCheckResourceAttr(name, "location", "West US"),
					resource.TestCheckResourceAttr(name, "dns_name", "testAccDnsName1"),
				),
			},
		},
	})
}

// testCheckAccArmPublicIPExists returns the resource.TestCheckFunc which
// verifies that the public IP with the provided internal name exists and
// is well defined both within the schema, and on Azure.
func testCheckAccArmPublicIPExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// check forexistence in internal state:
		res, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Could not find public IP %q.", name)
		}

		resName := res.Primary.Attributes["name"]
		resGrp := res.Primary.Attributes["resource_group_name"]

		publicIPClient := testAccProvider.Meta().(*ArmClient).vnetClient

		resp, err := publicIPClient.Get(resGrp, resName)
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Public IP %q does not exist on Azure!", resName)
		}
		if err != nil {
			return fmt.Errorf("Error reading the state of public IP %q: %s", resName, err)
		}

		return nil
	}
}

// testCheckAccArmPublicIPDeleted is a resource.TestCheckFunc which checks
// that out public IP has been deleted off Azure.
func testCheckAccArmPublicIPDeleted(s *terraform.State) error {
	for _, res := range s.RootModule().Resources {
		if res.Type != "azurerm_public_ip" {
			continue
		}

		name := res.Primary.Attributes["name"]
		resGrp := res.Primary.Attributes["resource_group_name"]

		publicIPClient := testAccProvider.Meta().(ArmClient).publicIPClient
		resp, err := publicIPClient.Get(resGrp, name)

		if resp.StatusCode == http.StatusNotFound {
			return nil
		}

		if err != nil {
			return fmt.Errorf("Error checking if ARM public IP %q got deleted: %s", name, err)
		}
	}

	return nil
}

// testAccAzureRMPublicIPAddress is the config tests will be conducted upon.
// It is the same as the config for network public IPs as the two resource are
// so co-dependendt.
var testAccAzureRMPublicIPAddress = testAccAzureRMNetworkInterfaceConfig
