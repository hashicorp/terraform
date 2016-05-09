package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"regexp"
)

func TestAccARMSimpleLB_basic(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureSimpleLB_basic, ri, ri)

	justBeThere := regexp.MustCompile(".*")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckARMSimpleRMLBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckARMSimpleLBExists("azurerm_simple_lb.test"),
					resource.TestMatchResourceAttr("azurerm_simple_lb.test", "frontend_id", justBeThere),
					resource.TestMatchResourceAttr("azurerm_simple_lb.test", "backend_pool_id", justBeThere),
				),
			},
		},
	})
}

func TestAccARMSimpleLB_updateTage(t *testing.T) {

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureSimpleLB_tags, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureSimpleLB_updateTags, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckARMSimpleRMLBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckARMSimpleLBExists("azurerm_simple_lb.test"),
					resource.TestCheckResourceAttr(
						"azurerm_simple_lb.test", "tags.#", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_simple_lb.test", "tags.environment", "Production"),
					resource.TestCheckResourceAttr(
						"azurerm_simple_lb.test", "tags.cost_center", "MSFT"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckARMSimpleLBExists("azurerm_simple_lb.test"),
					resource.TestCheckResourceAttr(
						"azurerm_simple_lb.test", "tags.#", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_simple_lb.test", "tags.environment", "staging"),
				),
			},
		},
	})
}

func testCheckARMSimpleLBExists(name string) resource.TestCheckFunc {
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

func testCheckARMSimpleRMLBDestroy(s *terraform.State) error {
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

var testAccAzureSimpleLB_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestlbrg-%d"
    location = "West US"
}

resource "azurerm_public_ip" "test" {
    name = "simplelbip"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"
}

resource "azurerm_simple_lb" "test" {
    name = "acctestlb%d"
    location = "West US"
    type = "Microsoft.Network/loadBalancers"
    resource_group_name = "${azurerm_resource_group.test.name}"
    frontend_allocation_method = "Dynamic"
    frontend_public_ip_address = "${azurerm_public_ip.test.id}"

    probe {
        name = "testProbe1"
        protocol = "Tcp"
        port = 22
        interval = 5
        number_of_probes = 16
    }
    rule {
	protocol = "Tcp"
	load_distribution = "Default"
	frontend_port = 22
	backend_port = 22
	name = "rule1"
	probe_name = "testProbe1"
    }
}
`

var testAccAzureSimpleLB_tags = `
resource "azurerm_resource_group" "test" {
    name = "acctestlbrg-%d"
    location = "West US"
}

resource "azurerm_public_ip" "test" {
    name = "simplelbip"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"
}

resource "azurerm_simple_lb" "test" {
    name = "acctestlb%d"
    location = "West US"
    type = "Microsoft.Network/loadBalancers"
    resource_group_name = "${azurerm_resource_group.test.name}"
    frontend_allocation_method = "Dynamic"
    frontend_public_ip_address = "${azurerm_public_ip.test.id}"

    probe {
        name = "testProbe1"
        protocol = "Tcp"
        port = 22
        interval = 5
        number_of_probes = 16
    }
    rule {
	protocol = "Tcp"
	load_distribution = "Default"
	frontend_port = 22
	backend_port = 22
	name = "rule1"
	probe_name = "testProbe1"
    }

    tags {
	environment = "Production"
	cost_center = "MSFT"
    }
}
`

var testAccAzureSimpleLB_updateTags = `
resource "azurerm_resource_group" "test" {
    name = "acctestlbrg-%d"
    location = "West US"
}

resource "azurerm_public_ip" "test" {
    name = "simplelbip"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"
}

resource "azurerm_simple_lb" "test" {
    name = "acctestlb%d"
    location = "West US"
    type = "Microsoft.Network/loadBalancers"
    resource_group_name = "${azurerm_resource_group.test.name}"
    frontend_allocation_method = "Dynamic"
    frontend_public_ip_address = "${azurerm_public_ip.test.id}"

    probe {
        name = "testProbe1"
        protocol = "Tcp"
        port = 22
        interval = 5
        number_of_probes = 16
    }
    rule {
	protocol = "Tcp"
	load_distribution = "Default"
	frontend_port = 22
	backend_port = 22
	name = "rule1"
	probe_name = "testProbe1"
    }

    tags {
	environment = "Production"
	cost_center = "MSFT"
    }
}
`
