package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureLoadBalancer_basic(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccARMLoadBalancer_basic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckARMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckARMLoadBalancerExists("azurerm_load_balancer.test"),
				),
			},
		},
	})
}

func TestAccAzureLoadBalancer_withTags(t *testing.T) {

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccARMLoadBalancer_withTags, ri)
	postConfig := fmt.Sprintf(testAccARMLoadBalancer_withTagsUpdate, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckARMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckARMLoadBalancerExists("azurerm_load_balancer.test"),
					resource.TestCheckResourceAttr(
						"azurerm_load_balancer.test", "tags.environment", "tag1"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckARMLoadBalancerExists("azurerm_load_balancer.test"),
					resource.TestCheckResourceAttr(
						"azurerm_load_balancer.test", "tags.environment", "tag2"),
				),
			},
		},
	})
}

func testCheckARMLoadBalancerExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
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

		return nil
	}
}

func testCheckARMLoadBalancerDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).loadBalancerClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_load_balancer" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name, "")

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Load balancer still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

var testAccARMLoadBalancer_basic = `
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
    name = "buzzlb1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    type = "Microsoft.Network/loadBalancers"

    frontend_ip_name = "testfrontendip"
    frontend_ip_public_ip_id = "${azurerm_public_ip.test.id}"
    frontend_ip_private_ip_allocation = "Dynamic"
}
`

var testAccARMLoadBalancer_withTags = `
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
    name = "buzzlb1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    type = "Microsoft.Network/loadBalancers"

    frontend_ip_name = "testfrontendip"
    frontend_ip_public_ip_id = "${azurerm_public_ip.test.id}"
    frontend_ip_private_ip_allocation = "Dynamic"

    tags {
	environment = "tag1"
    }
}
`

var testAccARMLoadBalancer_withTagsUpdate = `
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
    name = "buzzlb1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    type = "Microsoft.Network/loadBalancers"

    frontend_ip_name = "testfrontendip"
    frontend_ip_public_ip_id = "${azurerm_public_ip.test.id}"
    frontend_ip_private_ip_allocation = "Dynamic"

    tags {
	environment = "tag2"
    }
}
`
