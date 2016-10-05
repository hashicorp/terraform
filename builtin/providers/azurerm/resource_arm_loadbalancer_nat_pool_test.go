package azurerm

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMLoadbalancerNatPool_basic(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	natPoolName := fmt.Sprintf("NatPool-%d", ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadbalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadbalancerNatPool_basic(ri, natPoolName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadbalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadbalancerNatPoolExists(natPoolName, &lb),
				),
			},
		},
	})
}

func TestAccAzureRMLoadbalancerNatPool_removal(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	natPoolName := fmt.Sprintf("NatPool-%d", ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadbalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadbalancerNatPool_basic(ri, natPoolName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadbalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadbalancerNatPoolExists(natPoolName, &lb),
				),
			},
			{
				Config: testAccAzureRMLoadbalancerNatPool_removal(ri),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadbalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadbalancerNatPoolNotExists(natPoolName, &lb),
				),
			},
		},
	})
}

func testCheckAzureRMLoadbalancerNatPoolExists(natPoolName string, lb *network.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, _, exists := findLoadBalancerNatPoolByName(lb, natPoolName)
		if !exists {
			return fmt.Errorf("A NAT Pool with name %q cannot be found.", natPoolName)
		}

		return nil
	}
}

func testCheckAzureRMLoadbalancerNatPoolNotExists(natPoolName string, lb *network.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, _, exists := findLoadBalancerNatPoolByName(lb, natPoolName)
		if exists {
			return fmt.Errorf("A NAT Pool with name %q has been found.", natPoolName)
		}

		return nil
	}
}

func testAccAzureRMLoadbalancerNatPool_basic(rInt int, natPoolName string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_public_ip" "test" {
    name = "test-ip-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"
}

resource "azurerm_lb" "test" {
    name = "arm-test-loadbalancer-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    frontend_ip_configuration {
      name = "one-%d"
      public_ip_address_id = "${azurerm_public_ip.test.id}"
    }
}

resource "azurerm_lb_nat_pool" "test" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  protocol = "Tcp"
  frontend_port_start = 80
  frontend_port_end = 81
  backend_port = 3389
  frontend_ip_configuration_name = "one-%d"
}

`, rInt, rInt, rInt, rInt, natPoolName, rInt)
}

func testAccAzureRMLoadbalancerNatPool_removal(rInt int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_public_ip" "test" {
    name = "test-ip-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"
}

resource "azurerm_lb" "test" {
    name = "arm-test-loadbalancer-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    frontend_ip_configuration {
      name = "one-%d"
      public_ip_address_id = "${azurerm_public_ip.test.id}"
    }
}
`, rInt, rInt, rInt, rInt)
}
