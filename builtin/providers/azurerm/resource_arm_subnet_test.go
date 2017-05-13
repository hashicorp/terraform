package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMSubnet_basic(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMSubnet_basic, ri, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSubnetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSubnetExists("azurerm_subnet.test"),
				),
			},
		},
	})
}

<<<<<<< HEAD
func TestAccAzureRMSubnet_routeTableUpdate(t *testing.T) {
=======
func TestAccAzureRMSubnet_routeTable(t *testing.T) {
>>>>>>> 772fac7a1... Changes to vNet and subnet files to allow subnet updates without losing routing table references

	ri := acctest.RandInt()
	initConfig := fmt.Sprintf(testAccAzureRMSubnet_routeTable, ri, ri, ri)
	updatedConfig := fmt.Sprintf(testAccAzureRMSubnet_updatedRouteTable, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSubnetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: initConfig,
				Check: resource.ComposeTestCheckFunc(
<<<<<<< HEAD
					testCheckAzureRMSubnetExists("azurerm_subnet.test"),
=======
					testCheckAzureRMSubnetExists("azurerm_subnet.rtTable2"),
>>>>>>> 772fac7a1... Changes to vNet and subnet files to allow subnet updates without losing routing table references
				),
			},

			resource.TestStep{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
<<<<<<< HEAD
					testCheckAzureRMSubnetRouteTableExists("azurerm_subnet.test", "test-RT"),
=======
					testCheckAzureRMSubnetRouteTableExists("azurerm_subnet.rtTable2", "rtTable2-RT"),
				),
			},
		},
	})
}

func TestAccAzureRMSubnet_routeTable_subnetInline(t *testing.T) {

	ri := acctest.RandInt()
	//initConfig := fmt.Sprintf(testAccAzureRMSubnet_routeTable_subnetInline, ri, ri, ri)
	updatedConfig := fmt.Sprintf(testAccAzureRMSubnet_updatedRouteTable, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSubnetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSubnetRouteTableExists("azurerm_subnet.rtTable2", "rtTable2-RT"),
>>>>>>> 772fac7a1... Changes to vNet and subnet files to allow subnet updates without losing routing table references
				),
			},
		},
	})
}

func TestAccAzureRMSubnet_disappears(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMSubnet_basic, ri, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSubnetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSubnetExists("azurerm_subnet.test"),
					testCheckAzureRMSubnetDisappears("azurerm_subnet.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testCheckAzureRMSubnetExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		log.Printf("[INFO] Checking Subnet addition.")

		name := rs.Primary.Attributes["name"]
		vnetName := rs.Primary.Attributes["virtual_network_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for subnet: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).subnetClient

		resp, err := conn.Get(resourceGroup, vnetName, name, "")
		if err != nil {
			return fmt.Errorf("Bad: Get on subnetClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Subnet %q (resource group: %q) does not exist", name, resourceGroup)
		}

		if resp.RouteTable == nil {
			return fmt.Errorf("Bad: Subnet %q (resource group: %q) does not contain route tables after add", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMSubnetRouteTableExists(subnetName string, routeTableId string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[subnetName]
		if !ok {
			return fmt.Errorf("Not found: %s", subnetName)
		}

		log.Printf("[INFO] Checking Subnet update.")

		name := rs.Primary.Attributes["name"]
		vnetName := rs.Primary.Attributes["virtual_network_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for subnet: %s", name)
		}

		vnetConn := testAccProvider.Meta().(*ArmClient).vnetClient
		vnetResp, vnetErr := vnetConn.Get(resourceGroup, vnetName, "")
		if vnetErr != nil {
			return fmt.Errorf("Bad: Get on vnetClient: %s", vnetErr)
		}

		if vnetResp.Subnets == nil {
			return fmt.Errorf("Bad: Vnet %q (resource group: %q) does not have subnets after update", vnetName, resourceGroup)
		}

		conn := testAccProvider.Meta().(*ArmClient).subnetClient

		resp, err := conn.Get(resourceGroup, vnetName, name, "")
		if err != nil {
			return fmt.Errorf("Bad: Get on subnetClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Subnet %q (resource group: %q) does not exist", subnetName, resourceGroup)
		}

		if resp.RouteTable == nil {
			return fmt.Errorf("Bad: Subnet %q (resource group: %q) does not contain route tables after update", subnetName, resourceGroup)
		}

		if !strings.Contains(*resp.RouteTable.ID, routeTableId) {
			return fmt.Errorf("Bad: Subnet %q (resource group: %q) does not have route table %q", subnetName, resourceGroup, routeTableId)
		}

		return nil
	}
}

func testCheckAzureRMSubnetDisappears(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		vnetName := rs.Primary.Attributes["virtual_network_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for subnet: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).subnetClient

		_, err := conn.Delete(resourceGroup, vnetName, name, make(chan struct{}))
		if err != nil {
			return fmt.Errorf("Bad: Delete on subnetClient: %s", err)
		}

		return nil
	}
}

func testCheckAzureRMSubnetDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).subnetClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_subnet" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		vnetName := rs.Primary.Attributes["virtual_network_name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, vnetName, name, "")

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Subnet still exists:\n%#v", resp.SubnetPropertiesFormat)
		}
	}

	return nil
}

var testAccAzureRMSubnet_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acctestvirtnet%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctestsubnet%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
	route_table_id = "${azurerm_route_table.test.id}" 
}

resource "azurerm_route_table" "test" {
	name = "acctestroutetable%d"
	resource_group_name = "${azurerm_resource_group.test.name}"
	location = "West US"
}

resource "azurerm_route" "test" {
	name = "acctestroute%d"
	resource_group_name = "${azurerm_resource_group.test.name}"
	route_table_name  = "${azurerm_route_table.test.name}" 

	address_prefix = "10.100.0.0/14" 
	next_hop_type = "VirtualAppliance" 
	next_hop_in_ip_address = "10.10.1.1" 
}
`

var testAccAzureRMSubnet_routeTable = `
<<<<<<< HEAD
resource "azurerm_resource_group" "test" {
=======
resource "azurerm_resource_group" "rtTable2" {
>>>>>>> 772fac7a1... Changes to vNet and subnet files to allow subnet updates without losing routing table references
    name = "acctestRG-%d"
    location = "West US"
}

<<<<<<< HEAD
resource "azurerm_virtual_network" "test" {
    name = "acctestvirtnet%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "acctestsubnet%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
	route_table_id       = "${azurerm_route_table.test.id}"
}

resource "azurerm_route_table" "test" {
  name                = "test-RT"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
=======
resource "azurerm_virtual_network" "rtTable2" {
    name = "acctestvirtnet%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.rtTable2.name}"
}

resource "azurerm_subnet" "rtTable2" {
    name = "acctestsubnet%d"
    resource_group_name = "${azurerm_resource_group.rtTable2.name}"
    virtual_network_name = "${azurerm_virtual_network.rtTable2.name}"
    address_prefix = "10.0.2.0/24"
	route_table_id       = "${azurerm_route_table.rtTable2.id}"
}

resource "azurerm_route_table" "rtTable2" {
  name                = "rtTable2-RT"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.rtTable2.name}"
>>>>>>> 772fac7a1... Changes to vNet and subnet files to allow subnet updates without losing routing table references
}

resource "azurerm_route" "route_a" {
  name                = "TestRouteA"
<<<<<<< HEAD
  resource_group_name = "${azurerm_resource_group.test.name}"
  route_table_name    = "${azurerm_route_table.test.name}"
=======
  resource_group_name = "${azurerm_resource_group.rtTable2.name}"
  route_table_name    = "${azurerm_route_table.rtTable2.name}"

  address_prefix         = "10.100.0.0/14"
  next_hop_type          = "VirtualAppliance"
  next_hop_in_ip_address = "10.10.1.1"
}`

var testAccAzureRMSubnet_routeTable_subnetInline = `
resource "azurerm_resource_group" "rtTable2" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_virtual_network" "rtTable2" {
    name = "acctestvirtnet%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.rtTable2.name}"
 	subnet {
    	name           = "rtTable2"
    	address_prefix = "10.0.3.0/24"
		route_table    = "${azurerm_route_table.rtTable2.id}"
  	}
	tags {
		environment = "Testing"
	}
}

resource "azurerm_route_table" "rtTable2" {
  name                = "rtTable2-RT"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.rtTable2.name}"
}

resource "azurerm_route" "route_a" {
  name                = "TestRouteA"
  resource_group_name = "${azurerm_resource_group.rtTable2.name}"
  route_table_name    = "${azurerm_route_table.rtTable2.name}"
>>>>>>> 772fac7a1... Changes to vNet and subnet files to allow subnet updates without losing routing table references

  address_prefix         = "10.100.0.0/14"
  next_hop_type          = "VirtualAppliance"
  next_hop_in_ip_address = "10.10.1.1"
}`

var testAccAzureRMSubnet_updatedRouteTable = `
<<<<<<< HEAD
resource "azurerm_resource_group" "test" {
=======
resource "azurerm_resource_group" "rtTable2" {
>>>>>>> 772fac7a1... Changes to vNet and subnet files to allow subnet updates without losing routing table references
    name = "acctestRG-%d"
    location = "West US"
	tags {
		environment = "Testing"
	}
}

<<<<<<< HEAD
resource "azurerm_network_security_group" "test_secgroup" {
    name = "acceptanceTestSecurityGroup1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
=======
resource "azurerm_network_security_group" "rtTable2_secgroup" {
    name = "acceptanceTestSecurityGroup1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.rtTable2.name}"
>>>>>>> 772fac7a1... Changes to vNet and subnet files to allow subnet updates without losing routing table references

    security_rule {
        name = "test123"
        priority = 100
        direction = "Inbound"
        access = "Allow"
        protocol = "Tcp"
        source_port_range = "*"
        destination_port_range = "*"
        source_address_prefix = "*"
        destination_address_prefix = "*"
    }

    tags {
        environment = "Testing"
    }
}

<<<<<<< HEAD
resource "azurerm_virtual_network" "test" {
    name = "acctestvirtnet%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
=======
resource "azurerm_virtual_network" "rtTable2" {
    name = "acctestvirtnet%d"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.rtTable2.name}"
>>>>>>> 772fac7a1... Changes to vNet and subnet files to allow subnet updates without losing routing table references
	tags {
		environment = "Testing"
	}
}

<<<<<<< HEAD
resource "azurerm_subnet" "test" {
    name = "acctestsubnet%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.2.0/24"
	route_table_id       = "${azurerm_route_table.test.id}"
}

resource "azurerm_route_table" "test" {
  name                = "test-RT"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
=======
resource "azurerm_subnet" "rtTable2" {
    name = "acctestsubnet%d"
    resource_group_name = "${azurerm_resource_group.rtTable2.name}"
    virtual_network_name = "${azurerm_virtual_network.rtTable2.name}"
    address_prefix = "10.0.2.0/24"
	route_table_id       = "${azurerm_route_table.rtTable2.id}"
}

resource "azurerm_route_table" "rtTable2" {
  name                = "rtTable2-RT"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.rtTable2.name}"
>>>>>>> 772fac7a1... Changes to vNet and subnet files to allow subnet updates without losing routing table references
  tags {
    environment = "Testing"
  }
}

resource "azurerm_route" "route_a" {
  name                = "TestRouteA"
<<<<<<< HEAD
  resource_group_name = "${azurerm_resource_group.test.name}"
  route_table_name    = "${azurerm_route_table.test.name}"
=======
  resource_group_name = "${azurerm_resource_group.rtTable2.name}"
  route_table_name    = "${azurerm_route_table.rtTable2.name}"
>>>>>>> 772fac7a1... Changes to vNet and subnet files to allow subnet updates without losing routing table references

  address_prefix         = "10.100.0.0/14"
  next_hop_type          = "VirtualAppliance"
  next_hop_in_ip_address = "10.10.1.1"
}`
