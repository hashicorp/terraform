package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceAzureRMRouteTableNextHopType_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Random",
			ErrCount: 1,
		},
		{
			Value:    "VirtualNetworkGateway",
			ErrCount: 0,
		},
		{
			Value:    "VNETLocal",
			ErrCount: 0,
		},
		{
			Value:    "Internet",
			ErrCount: 0,
		},
		{
			Value:    "VirtualAppliance",
			ErrCount: 0,
		},
		{
			Value:    "None",
			ErrCount: 0,
		},
		{
			Value:    "VIRTUALNETWORKGATEWAY",
			ErrCount: 0,
		},
		{
			Value:    "virtualnetworkgateway",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateRouteTableNextHopType(tc.Value, "azurerm_route_table")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Route Table nextHopType to trigger a validation error")
		}
	}
}

func TestAccAzureRMRouteTable_basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRouteTableDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMRouteTable_basic,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMRouteTableExists("azurerm_route_table.test"),
				),
			},
		},
	})
}

func TestAccAzureRMRouteTable_multipleRoutes(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRouteTableDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMRouteTable_basic,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMRouteTableExists("azurerm_route_table.test"),
					resource.TestCheckResourceAttr(
						"azurerm_route_table.test", "route.#", "1"),
				),
			},

			resource.TestStep{
				Config: testAccAzureRMRouteTable_multipleRoutes,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMRouteTableExists("azurerm_route_table.test"),
					resource.TestCheckResourceAttr(
						"azurerm_route_table.test", "route.#", "2"),
				),
			},
		},
	})
}

func testCheckAzureRMRouteTableExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for route table: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).routeTablesClient

		resp, err := conn.Get(resourceGroup, name, "")
		if err != nil {
			return fmt.Errorf("Bad: Get on routeTablesClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Route Table %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMRouteTableDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).routeTablesClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_route_table" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name, "")

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Route Table still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

var testAccAzureRMRouteTable_basic = `
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}

resource "azurerm_route_table" "test" {
    name = "acceptanceTestSecurityGroup1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    route {
    	name = "route1"
    	address_prefix = "*"
    	next_hop_type = "internet"
    }
}
`

var testAccAzureRMRouteTable_multipleRoutes = `
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}

resource "azurerm_route_table" "test" {
    name = "acceptanceTestSecurityGroup1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    route {
    	name = "route1"
    	address_prefix = "*"
    	next_hop_type = "internet"
    }

    route {
    	name = "route2"
    	address_prefix = "*"
    	next_hop_type = "virtualappliance"
    }
}
`
