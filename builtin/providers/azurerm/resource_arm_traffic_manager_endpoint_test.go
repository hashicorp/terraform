package azurerm

import (
	"fmt"
	"net/http"
	"path"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMTrafficManagerEndpoint_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMTrafficManagerEndpoint_basic, ri, ri, ri, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMTrafficManagerEndpointDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.testAzure"),
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.testExternal"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_endpoint.testAzure", "endpoint_status", "Enabled"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_endpoint.testExternal", "endpoint_status", "Enabled"),
				),
			},
		},
	})
}

func TestAccAzureRMTrafficManagerEndpoint_disappears(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMTrafficManagerEndpoint_basic, ri, ri, ri, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMTrafficManagerEndpointDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.testAzure"),
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.testExternal"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_endpoint.testAzure", "endpoint_status", "Enabled"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_endpoint.testExternal", "endpoint_status", "Enabled"),
					testCheckAzureRMTrafficManagerEndpointDisappears("azurerm_traffic_manager_endpoint.testAzure"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAzureRMTrafficManagerEndpoint_basicDisableExternal(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMTrafficManagerEndpoint_basic, ri, ri, ri, ri, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMTrafficManagerEndpoint_basicDisableExternal, ri, ri, ri, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMTrafficManagerEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.testAzure"),
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.testExternal"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_endpoint.testAzure", "endpoint_status", "Enabled"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_endpoint.testExternal", "endpoint_status", "Enabled"),
				),
			},
			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.testAzure"),
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.testExternal"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_endpoint.testAzure", "endpoint_status", "Enabled"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_endpoint.testExternal", "endpoint_status", "Disabled"),
				),
			},
		},
	})
}

// Altering weight might be used to ramp up migration traffic
func TestAccAzureRMTrafficManagerEndpoint_updateWeight(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMTrafficManagerEndpoint_weight, ri, ri, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMTrafficManagerEndpoint_updateWeight, ri, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMTrafficManagerEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.testExternal"),
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.testExternalNew"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_endpoint.testExternal", "weight", "50"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_endpoint.testExternalNew", "weight", "50"),
				),
			},
			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.testExternal"),
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.testExternalNew"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_endpoint.testExternal", "weight", "25"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_endpoint.testExternalNew", "weight", "75"),
				),
			},
		},
	})
}

// Altering priority might be used to switch failover/active roles
func TestAccAzureRMTrafficManagerEndpoint_updatePriority(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMTrafficManagerEndpoint_priority, ri, ri, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMTrafficManagerEndpoint_updatePriority, ri, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMTrafficManagerEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.testExternal"),
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.testExternalNew"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_endpoint.testExternal", "priority", "1"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_endpoint.testExternalNew", "priority", "2"),
				),
			},
			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.testExternal"),
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.testExternalNew"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_endpoint.testExternal", "priority", "3"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_endpoint.testExternalNew", "priority", "2"),
				),
			},
		},
	})
}

func TestAccAzureRMTrafficManagerEndpoint_nestedEndpoints(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMTrafficManagerEndpoint_nestedEndpoints, ri, ri, ri, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMTrafficManagerEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.nested"),
					testCheckAzureRMTrafficManagerEndpointExists("azurerm_traffic_manager_endpoint.externalChild"),
				),
			},
		},
	})
}

func testCheckAzureRMTrafficManagerEndpointExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		endpointType := rs.Primary.Attributes["type"]
		profileName := rs.Primary.Attributes["profile_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for Traffic Manager Profile: %s", name)
		}

		// Ensure resource group/virtual network combination exists in API
		conn := testAccProvider.Meta().(*ArmClient).trafficManagerEndpointsClient

		resp, err := conn.Get(resourceGroup, profileName, path.Base(endpointType), name)
		if err != nil {
			return fmt.Errorf("Bad: Get on trafficManagerEndpointsClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Traffic Manager Endpoint %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMTrafficManagerEndpointDisappears(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		endpointType := rs.Primary.Attributes["type"]
		profileName := rs.Primary.Attributes["profile_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for Traffic Manager Profile: %s", name)
		}

		// Ensure resource group/virtual network combination exists in API
		conn := testAccProvider.Meta().(*ArmClient).trafficManagerEndpointsClient

		_, err := conn.Delete(resourceGroup, profileName, path.Base(endpointType), name)
		if err != nil {
			return fmt.Errorf("Bad: Delete on trafficManagerEndpointsClient: %s", err)
		}

		return nil
	}
}

func testCheckAzureRMTrafficManagerEndpointDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).trafficManagerEndpointsClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_traffic_manager_endpoint" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		endpointType := rs.Primary.Attributes["type"]
		profileName := rs.Primary.Attributes["profile_name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, profileName, path.Base(endpointType), name)
		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Traffic Manager Endpoint sitll exists:\n%#v", resp.EndpointProperties)
		}
	}

	return nil
}

var testAccAzureRMTrafficManagerEndpoint_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_traffic_manager_profile" "test" {
    name = "acctesttmp%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    traffic_routing_method = "Weighted"

    dns_config {
        relative_name = "acctesttmp%d"
        ttl = 30
    }

    monitor_config {
        protocol = "https"
        port = 443
        path = "/"
    }
}

resource "azurerm_public_ip" "test" {
    name = "acctestpublicip-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"
    domain_name_label = "acctestpublicip-%d"
}

resource "azurerm_traffic_manager_endpoint" "testAzure" {
    name = "acctestend-azure%d"
    type = "azureEndpoints"
    target_resource_id = "${azurerm_public_ip.test.id}"
    weight = 3
    profile_name = "${azurerm_traffic_manager_profile.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_traffic_manager_endpoint" "testExternal" {
    name = "acctestend-external%d"
    type = "externalEndpoints"
    target = "terraform.io"
    weight = 3
    profile_name = "${azurerm_traffic_manager_profile.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
`

var testAccAzureRMTrafficManagerEndpoint_basicDisableExternal = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_traffic_manager_profile" "test" {
    name = "acctesttmp%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    traffic_routing_method = "Weighted"

    dns_config {
        relative_name = "acctesttmp%d"
        ttl = 30
    }

    monitor_config {
        protocol = "https"
        port = 443
        path = "/"
    }
}

resource "azurerm_public_ip" "test" {
    name = "acctestpublicip-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"
    domain_name_label = "acctestpublicip-%d"
}

resource "azurerm_traffic_manager_endpoint" "testAzure" {
    name = "acctestend-azure%d"
    type = "azureEndpoints"
    target_resource_id = "${azurerm_public_ip.test.id}"
    weight = 3
    profile_name = "${azurerm_traffic_manager_profile.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_traffic_manager_endpoint" "testExternal" {
    name = "acctestend-external%d"
    endpoint_status = "Disabled"
    type = "externalEndpoints"
    target = "terraform.io"
    weight = 3
    profile_name = "${azurerm_traffic_manager_profile.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
`

var testAccAzureRMTrafficManagerEndpoint_weight = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_traffic_manager_profile" "test" {
    name = "acctesttmp%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    traffic_routing_method = "Weighted"

    dns_config {
        relative_name = "acctesttmp%d"
        ttl = 30
    }

    monitor_config {
        protocol = "https"
        port = 443
        path = "/"
    }
}

resource "azurerm_traffic_manager_endpoint" "testExternal" {
    name = "acctestend-external%d"
    type = "externalEndpoints"
    target = "terraform.io"
    weight = 50
    profile_name = "${azurerm_traffic_manager_profile.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_traffic_manager_endpoint" "testExternalNew" {
    name = "acctestend-external%d-2"
    type = "externalEndpoints"
    target = "www.terraform.io"
    weight = 50
    profile_name = "${azurerm_traffic_manager_profile.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
`

var testAccAzureRMTrafficManagerEndpoint_updateWeight = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_traffic_manager_profile" "test" {
    name = "acctesttmp%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    traffic_routing_method = "Weighted"

    dns_config {
        relative_name = "acctesttmp%d"
        ttl = 30
    }

    monitor_config {
        protocol = "https"
        port = 443
        path = "/"
    }
}

resource "azurerm_traffic_manager_endpoint" "testExternal" {
    name = "acctestend-external%d"
    type = "externalEndpoints"
    target = "terraform.io"
    weight = 25
    profile_name = "${azurerm_traffic_manager_profile.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_traffic_manager_endpoint" "testExternalNew" {
    name = "acctestend-external%d-2"
    type = "externalEndpoints"
    target = "www.terraform.io"
    weight = 75
    profile_name = "${azurerm_traffic_manager_profile.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
`

var testAccAzureRMTrafficManagerEndpoint_priority = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_traffic_manager_profile" "test" {
    name = "acctesttmp%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    traffic_routing_method = "Priority"

    dns_config {
        relative_name = "acctesttmp%d"
        ttl = 30
    }

    monitor_config {
        protocol = "https"
        port = 443
        path = "/"
    }
}

resource "azurerm_traffic_manager_endpoint" "testExternal" {
    name = "acctestend-external%d"
    type = "externalEndpoints"
    target = "terraform.io"
    priority = 1
    profile_name = "${azurerm_traffic_manager_profile.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_traffic_manager_endpoint" "testExternalNew" {
    name = "acctestend-external%d-2"
    type = "externalEndpoints"
    target = "www.terraform.io"
    priority = 2
    profile_name = "${azurerm_traffic_manager_profile.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
`

var testAccAzureRMTrafficManagerEndpoint_updatePriority = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_traffic_manager_profile" "test" {
    name = "acctesttmp%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    traffic_routing_method = "Priority"

    dns_config {
        relative_name = "acctesttmp%d"
        ttl = 30
    }

    monitor_config {
        protocol = "https"
        port = 443
        path = "/"
    }
}

resource "azurerm_traffic_manager_endpoint" "testExternal" {
    name = "acctestend-external%d"
    type = "externalEndpoints"
    target = "terraform.io"
    priority = 3
    profile_name = "${azurerm_traffic_manager_profile.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_traffic_manager_endpoint" "testExternalNew" {
    name = "acctestend-external%d-2"
    type = "externalEndpoints"
    target = "www.terraform.io"
    priority = 2
    profile_name = "${azurerm_traffic_manager_profile.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
`

var testAccAzureRMTrafficManagerEndpoint_nestedEndpoints = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_traffic_manager_profile" "parent" {
    name = "acctesttmpparent%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    traffic_routing_method = "Priority"

    dns_config {
        relative_name = "acctestparent%d"
        ttl = 30
    }

    monitor_config {
        protocol = "https"
        port = 443
        path = "/"
    }
}

resource "azurerm_traffic_manager_profile" "child" {
    name = "acctesttmpchild%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    traffic_routing_method = "Priority"

    dns_config {
        relative_name = "acctesttmpchild%d"
        ttl = 30
    }

    monitor_config {
        protocol = "https"
        port = 443
        path = "/"
    }
}

resource "azurerm_traffic_manager_endpoint" "nested" {
    name = "acctestend-parent%d"
    type = "nestedEndpoints"
    target_resource_id = "${azurerm_traffic_manager_profile.child.id}"
    priority = 1
    profile_name = "${azurerm_traffic_manager_profile.parent.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    min_child_endpoints = 1
}

resource "azurerm_traffic_manager_endpoint" "externalChild" {
    name = "acctestend-child%d"
    type = "externalEndpoints"
    target = "terraform.io"
    priority = 1
    profile_name = "${azurerm_traffic_manager_profile.child.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
`
