package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMRoute_basic(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMRoute_basic, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRouteDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMRouteExists("azurerm_route.test"),
				),
			},
		},
	})
}

func TestAccAzureRMRoute_disappears(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMRoute_basic, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRouteDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMRouteExists("azurerm_route.test"),
					testCheckAzureRMRouteDisappears("azurerm_route.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAzureRMRoute_multipleRoutes(t *testing.T) {

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMRoute_basic, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMRoute_multipleRoutes, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRouteDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMRouteExists("azurerm_route.test"),
				),
			},

			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMRouteExists("azurerm_route.test1"),
				),
			},
		},
	})
}

func testCheckAzureRMRouteExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		rtName := rs.Primary.Attributes["route_table_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for route: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).routesClient

		resp, err := conn.Get(resourceGroup, rtName, name)
		if err != nil {
			return fmt.Errorf("Bad: Get on routesClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Route %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMRouteDisappears(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		rtName := rs.Primary.Attributes["route_table_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for route: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).routesClient

		_, error := conn.Delete(resourceGroup, rtName, name, make(chan struct{}))
		err := <-error
		if err != nil {
			return fmt.Errorf("Bad: Delete on routesClient: %s", err)
		}

		return nil
	}
}

func testCheckAzureRMRouteDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).routesClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_route" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		rtName := rs.Primary.Attributes["route_table_name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, rtName, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Route still exists:\n%#v", resp.RoutePropertiesFormat)
		}
	}

	return nil
}

var testAccAzureRMRoute_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_route_table" "test" {
    name = "acctestrt%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_route" "test" {
    name = "acctestroute%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    route_table_name = "${azurerm_route_table.test.name}"

    address_prefix = "10.1.0.0/16"
    next_hop_type = "vnetlocal"
}
`

var testAccAzureRMRoute_multipleRoutes = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_route_table" "test" {
    name = "acctestrt%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_route" "test1" {
    name = "acctestroute%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    route_table_name = "${azurerm_route_table.test.name}"

    address_prefix = "10.2.0.0/16"
    next_hop_type = "none"
}
`
