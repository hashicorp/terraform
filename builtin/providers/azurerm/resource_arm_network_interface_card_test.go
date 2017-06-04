package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMNetworkInterface_basic(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkInterfaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMNetworkInterface_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test"),
				),
			},
		},
	})
}

func TestAccAzureRMNetworkInterface_disappears(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkInterfaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMNetworkInterface_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test"),
					testCheckAzureRMNetworkInterfaceDisappears("azurerm_network_interface.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAzureRMNetworkInterface_enableIPForwarding(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkInterfaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMNetworkInterface_ipForwarding(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "enable_ip_forwarding", "true"),
				),
			},
		},
	})
}

func TestAccAzureRMNetworkInterface_multipleLoadBalancers(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkInterfaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMNetworkInterface_multipleLoadBalancers(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test1"),
					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test2"),
				),
			},
		},
	})
}

func TestAccAzureRMNetworkInterface_withTags(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkInterfaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMNetworkInterface_withTags(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "tags.%", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "tags.environment", "Production"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "tags.cost_center", "MSFT"),
				),
			},
			{
				Config: testAccAzureRMNetworkInterface_withTagsUpdate(rInt),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMNetworkInterfaceExists("azurerm_network_interface.test"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_network_interface.test", "tags.environment", "staging"),
				),
			},
		},
	})
}

func testCheckAzureRMNetworkInterfaceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for availability set: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).ifaceClient

		resp, err := conn.Get(resourceGroup, name, "")
		if err != nil {
			return fmt.Errorf("Bad: Get on ifaceClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Network Interface %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMNetworkInterfaceDisappears(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for availability set: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).ifaceClient

		_, error := conn.Delete(resourceGroup, name, make(chan struct{}))
		err := <-error
		if err != nil {
			return fmt.Errorf("Bad: Delete on ifaceClient: %s", err)
		}

		return nil
	}
}

func testCheckAzureRMNetworkInterfaceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).ifaceClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_network_interface" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name, "")

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Network Interface still exists:\n%#v", resp.InterfacePropertiesFormat)
		}
	}

	return nil
}

func testAccAzureRMNetworkInterface_basic(rInt int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctest-rg-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acceptanceTestVirtualNetwork1"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "testsubnet"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acceptanceTestNetworkInterface1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}
`, rInt)
}

func testAccAzureRMNetworkInterface_ipForwarding(rInt int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctest-rg-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acceptanceTestVirtualNetwork1"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "testsubnet"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acceptanceTestNetworkInterface1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    enable_ip_forwarding = true

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }
}
`, rInt)
}

func testAccAzureRMNetworkInterface_withTags(rInt int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctest-rg-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acceptanceTestVirtualNetwork1"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "testsubnet"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acceptanceTestNetworkInterface1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }

    tags {
	environment = "Production"
	cost_center = "MSFT"
    }
}
`, rInt)
}

func testAccAzureRMNetworkInterface_withTagsUpdate(rInt int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctest-rg-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acceptanceTestVirtualNetwork1"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "testsubnet"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acceptanceTestNetworkInterface1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }

    tags {
	environment = "staging"
    }
}
`, rInt)
}

func testAccAzureRMNetworkInterface_multipleLoadBalancers(rInt int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctest-rg-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acceptanceTestVirtualNetwork1"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "testsubnet"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_public_ip" "testext" {
  name                         = "testpublicipext"
  location                     = "West US"
  resource_group_name          = "${azurerm_resource_group.test.name}"
  public_ip_address_allocation = "static"
}

resource "azurerm_lb" "testext" {
  name                = "testlbext"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"

  frontend_ip_configuration {
    name                 = "publicipext"
    public_ip_address_id = "${azurerm_public_ip.testext.id}"
  }
}

resource "azurerm_lb_backend_address_pool" "testext" {
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id     = "${azurerm_lb.testext.id}"
  name                = "testbackendpoolext"
}

resource "azurerm_lb_nat_rule" "testext" {
  name = "testnatruleext"
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.testext.id}"
  protocol = "Tcp"
  frontend_port = 3389
  backend_port = 3390
  frontend_ip_configuration_name = "publicipext"
}

resource "azurerm_public_ip" "testint" {
  name                         = "testpublicipint"
  location                     = "West US"
  resource_group_name          = "${azurerm_resource_group.test.name}"
  public_ip_address_allocation = "static"
}

resource "azurerm_lb" "testint" {
  name                = "testlbint"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"

  frontend_ip_configuration {
    name                 		  = "publicipint"
    subnet_id                     = "${azurerm_subnet.test.id}"
    private_ip_address_allocation = "Dynamic"
  }
}

resource "azurerm_lb_backend_address_pool" "testint" {
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id     = "${azurerm_lb.testint.id}"
  name                = "testbackendpoolint"
}

resource "azurerm_lb_nat_rule" "testint" {
  name = "testnatruleint"
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.testint.id}"
  protocol = "Tcp"
  frontend_port = 3389
  backend_port = 3391
  frontend_ip_configuration_name = "publicipint"
}

resource "azurerm_network_interface" "test1" {
    name = "acceptanceTestNetworkInterface1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    enable_ip_forwarding = true

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
		load_balancer_backend_address_pools_ids = [
			"${azurerm_lb_backend_address_pool.testext.id}",
			"${azurerm_lb_backend_address_pool.testint.id}",
		]
    }
}

resource "azurerm_network_interface" "test2" {
    name = "acceptanceTestNetworkInterface2"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    enable_ip_forwarding = true

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
		load_balancer_inbound_nat_rules_ids = [
			"${azurerm_lb_nat_rule.testext.id}",
			"${azurerm_lb_nat_rule.testint.id}",
		]
    }
}
`, rInt)
}
