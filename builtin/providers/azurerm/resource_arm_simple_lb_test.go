package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"regexp"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMSimpleLB_basic(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMSimpleLB_basic, ri, ri, ri)

	justBeThere := regexp.MustCompile(".*")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSimpleRMLBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSimpleLBExists("azurerm_simple_lb.test"),
					resource.TestMatchResourceAttr("azurerm_simple_lb.test", "frontend_id", justBeThere),
					resource.TestMatchResourceAttr("azurerm_simple_lb.test", "backend_pool_id", justBeThere),
				),
			},
		},
	})
}

func TestAccAzureRMSimpleLB_updateTag(t *testing.T) {

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMSimpleLB_tags, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMSimpleLB_updateTags, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSimpleRMLBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSimpleLBExists("azurerm_simple_lb.test"),
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
					testCheckAzureRMSimpleLBExists("azurerm_simple_lb.test"),
					resource.TestCheckResourceAttr(
						"azurerm_simple_lb.test", "tags.#", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_simple_lb.test", "tags.environment", "staging"),
				),
			},
		},
	})
}

func TestAccAzureRMSimpleLB_updateProbe(t *testing.T) {

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMSimpleLB_probe, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMSimpleLB_probeUpdate, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSimpleRMLBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSimpleLBExists("azurerm_simple_lb.test"),
					resource.TestCheckResourceAttr(
						"azurerm_simple_lb.test", "probe.#", "2"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSimpleLBExists("azurerm_simple_lb.test"),
					resource.TestCheckResourceAttr(
						"azurerm_simple_lb.test", "probe.#", "1"),
				),
			},
		},
	})
}

func TestAccAzureRMSimpleLB_dynamicFrontEndIPAddress(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMSimpleLB_dynamicFrontEndIPAddress, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSimpleRMLBDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testCheckAzureRMSimpleLBExists("azurerm_simple_lb.test"),
			},
			{
				Config: config,
				// left here to make it explicit that we expect an empty plan
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testCheckAzureRMSimpleLBExists(name string) resource.TestCheckFunc {
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

func testCheckAzureRMSimpleRMLBDestroy(s *terraform.State) error {
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

var testAccAzureRMSimpleLB_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestlbrg-%d"
    location = "West US"
}

resource "azurerm_public_ip" "test" {
    name = "simplelbip%d"
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

var testAccAzureRMSimpleLB_tags = `
resource "azurerm_resource_group" "test" {
    name = "acctestlbrg-%d"
    location = "West US"
}

resource "azurerm_public_ip" "test" {
    name = "simplelbip%d"
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

var testAccAzureRMSimpleLB_updateTags = `
resource "azurerm_resource_group" "test" {
    name = "acctestlbrg-%d"
    location = "West US"
}

resource "azurerm_public_ip" "test" {
    name = "simplelbip%d"
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
	environment = "staging"
    }
}
`

var testAccAzureRMSimpleLB_probe = `
resource "azurerm_resource_group" "test" {
    name = "acctestlbrg-%d"
    location = "West US"
}

resource "azurerm_public_ip" "test" {
    name = "simplelbip%d"
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

    probe {
        name = "testProbe2"
        protocol = "Tcp"
        port = 80
        interval = 5
        number_of_probes = 16
    }
    rule {
	protocol = "Tcp"
	load_distribution = "Default"
	frontend_port = 80
	backend_port = 80
	name = "rule2"
	probe_name = "testProbe2"
    }
}
`

var testAccAzureRMSimpleLB_probeUpdate = `
resource "azurerm_resource_group" "test" {
    name = "acctestlbrg-%d"
    location = "West US"
}

resource "azurerm_public_ip" "test" {
    name = "simplelbip%d"
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

const testAccAzureRMSimpleLB_dynamicFrontEndIPAddress = `
resource "azurerm_resource_group" "test" {
    name = "acctestlbrg-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
  name                = "acctestnet%d"
  address_space       = ["10.0.0.0/16"]
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
  name                 = "acctestsubnet%d"
  resource_group_name  = "${azurerm_resource_group.test.name}"
  virtual_network_name = "${azurerm_virtual_network.test.name}"
  address_prefix       = "10.0.0.0/24"
}

resource "azurerm_simple_lb" "test" {
    name = "acctestlb%d"
    location = "West US"
    type = "Microsoft.Network/loadBalancers"
    resource_group_name = "${azurerm_resource_group.test.name}"
    frontend_subnet = "${azurerm_subnet.test.id}"
    frontend_allocation_method = "Dynamic"

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
