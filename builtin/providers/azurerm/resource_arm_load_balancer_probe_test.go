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

func TestAccAzureLoadBalancerProbe_basic(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccARMLoadBalancerProbe_basic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckARMLoadBalancerProbeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckARMLoadBalancerProbeExists("azurerm_load_balancer_probe.test"),
				),
			},
		},
	})
}

func findProbeByName(probeName string, probeArray *[]network.Probe) bool {
	found := false
	for i := 0; i < len(*probeArray); i++ {
		if *(*probeArray)[i].Name == probeName {
			found = true
		}
	}
	return found
}

func testCheckARMLoadBalancerProbeExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["load_balancer_name"]
		probeName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for cdn endpoint: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).loadBalancerClient

		resp, err := conn.Get(resourceGroup, name, "")
		if err != nil {
			return fmt.Errorf("Bad: Get on loadBalancerClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Load Balancer %q (resource group: %q) does not exist", name, resourceGroup)
		}
		found := findProbeByName(probeName, resp.Properties.Probes)
		if !found {
			return fmt.Errorf("Failed to find the probe %s.", probeName)
		}

		return nil
	}
}

func testCheckARMLoadBalancerProbeDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).loadBalancerClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_load_balancer_probe" {
			continue
		}

		name := rs.Primary.Attributes["load_balancer_name"]
		probeName := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name, "")
		if err != nil {
			return nil
		}

		found := findProbeByName(probeName, resp.Properties.Probes)
		if found {
			return fmt.Errorf("Load balancer still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

var testAccARMLoadBalancerProbe_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestlbrg-%d"
    location = "West US"
}

resource "azurerm_public_ip" "test" {
    name = "testip"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"
}

resource "azurerm_load_balancer" "test" {
    name = "testb1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    type = "Microsoft.Network/loadBalancers"

    frontend_ip_name = "testfrontendip"
    frontend_ip_public_ip_id = "${azurerm_public_ip.test.id}"
    frontend_ip_private_ip_allocation = "Dynamic"
}

resource "azurerm_load_balancer_probe" "test" {
    name = "testprobe1"
    protocol = "Tcp"
    port = 22
    interval = 5
    number_of_probes = 16
    load_balancer_name = "${azurerm_load_balancer.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
`
