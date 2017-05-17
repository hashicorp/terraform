package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMTrafficManagerProfile_weighted(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMTrafficManagerProfile_weighted, ri, ri, ri)

	fqdn := fmt.Sprintf("acctesttmp%d.trafficmanager.net", ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMTrafficManagerProfileDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTrafficManagerProfileExists("azurerm_traffic_manager_profile.test"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_profile.test", "traffic_routing_method", "Weighted"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_profile.test", "fqdn", fqdn),
				),
			},
		},
	})
}

func TestAccAzureRMTrafficManagerProfile_performance(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMTrafficManagerProfile_performance, ri, ri, ri)

	fqdn := fmt.Sprintf("acctesttmp%d.trafficmanager.net", ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMTrafficManagerProfileDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTrafficManagerProfileExists("azurerm_traffic_manager_profile.test"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_profile.test", "traffic_routing_method", "Performance"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_profile.test", "fqdn", fqdn),
				),
			},
		},
	})
}

func TestAccAzureRMTrafficManagerProfile_priority(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMTrafficManagerProfile_priority, ri, ri, ri)

	fqdn := fmt.Sprintf("acctesttmp%d.trafficmanager.net", ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMTrafficManagerProfileDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTrafficManagerProfileExists("azurerm_traffic_manager_profile.test"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_profile.test", "traffic_routing_method", "Priority"),
					resource.TestCheckResourceAttr("azurerm_traffic_manager_profile.test", "fqdn", fqdn),
				),
			},
		},
	})
}

func TestAccAzureRMTrafficManagerProfile_withTags(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMTrafficManagerProfile_withTags, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMTrafficManagerProfile_withTagsUpdated, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMTrafficManagerProfileDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTrafficManagerProfileExists("azurerm_traffic_manager_profile.test"),
					resource.TestCheckResourceAttr(
						"azurerm_traffic_manager_profile.test", "tags.%", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_traffic_manager_profile.test", "tags.environment", "Production"),
					resource.TestCheckResourceAttr(
						"azurerm_traffic_manager_profile.test", "tags.cost_center", "MSFT"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMTrafficManagerProfileExists("azurerm_traffic_manager_profile.test"),
					resource.TestCheckResourceAttr(
						"azurerm_traffic_manager_profile.test", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_traffic_manager_profile.test", "tags.environment", "staging"),
				),
			},
		},
	})
}

func testCheckAzureRMTrafficManagerProfileExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for Traffic Manager Profile: %s", name)
		}

		// Ensure resource group/virtual network combination exists in API
		conn := testAccProvider.Meta().(*ArmClient).trafficManagerProfilesClient

		resp, err := conn.Get(resourceGroup, name)
		if err != nil {
			return fmt.Errorf("Bad: Get on trafficManagerProfilesClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Traffic Manager %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMTrafficManagerProfileDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).trafficManagerProfilesClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_traffic_manager_profile" {
			continue
		}

		log.Printf("[TRACE] test_profile %#v", rs)

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)
		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Traffic Manager profile sitll exists:\n%#v", resp.ProfileProperties)
		}
	}

	return nil
}

var testAccAzureRMTrafficManagerProfile_weighted = `
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
`

var testAccAzureRMTrafficManagerProfile_performance = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_traffic_manager_profile" "test" {
    name = "acctesttmp%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    traffic_routing_method = "Performance"

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
`

var testAccAzureRMTrafficManagerProfile_priority = `
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
`

var testAccAzureRMTrafficManagerProfile_withTags = `
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
    
    tags {
        environment = "Production"
        cost_center = "MSFT"
    }
}
`

var testAccAzureRMTrafficManagerProfile_withTagsUpdated = `
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
    
    tags {
        environment = "staging"
    }
}
`
