package azurerm

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/core/http"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMLoadBalancer_basic(t *testing.T) {
	name := "azurerm_load_balancer.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists(name),
					resource.TestCheckResourceAttr(name, "type", "internal"),
					// TODO: more
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancer_internal(t *testing.T) {
	name := "azurerm_load_balancer.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerConfig_internal,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists(name),
					resource.TestCheckResourceAttr(name, "type", "internal"),
					// TODO: more
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancer_public(t *testing.T) {
	name := "azurerm_load_balancer.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerConfig_public,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists(name),
					resource.TestCheckResourceAttr(name, "type", "public"),
					// TODO: more
				),
			},
		},
	})
}

func testCheckAzureRMLoadBalancerExists(name string) resource.TestCheckFunc {

}

func testCheckAzureRMLoadBalancerDestroy(s *terraform.State) error {

}

var testAccAzureRMLoadBalancerConfig_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_load_balancer" "test" {
  name = "acctestlb-%d"
  type = "internal"
  location = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
}
`
var testAccAzureRMLoadBalancerConfig_internal = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"

    tags {
  		environment = "Production"
  		cost_center = "MSFT"
    }
}

resource "azurerm_load_balancer" "test" {
  name = "acctestblb-%d"
  type = "internal"
  location = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"

  frontend_ip_configuration {
    name = "examplelbfront"
    private_ip_allocation_method = "static"
    private_ip_address = "10.123.234.156"
    subnet = "subnetid"
  }

  backend_address_pool {
    name = "examplebackend"
  }

  load_balancing_rule {
    name = "examplelbrule"
    frontend_ip_configuration = "examplelbfront"
    backend_address_pool = "examplebackend"
    probe = "examplelbprobe"
    protocol = "Tcp"
		load_distribution = "Default"
    frontend_port = 80
    backend_port = 80
    idle_timeout_in_minutes = 15
		enable_floating_ip = true
  }

  probe {
    name = "examplelbprobe"
    protocol = "Tcp"
    port = 80
    number_of_probes = 2
    interval_in_seconds = 15
  }

  tags {
    environment = "Production"
    cost_center = "MSFT"
  }
}
`

var testAccAzureRMLoadBalancerConfig_internal = `
TODO: implement me
`
