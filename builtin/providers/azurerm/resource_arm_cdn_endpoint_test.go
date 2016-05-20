package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMCdnEndpoint_basic(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMCdnEndpoint_basic, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMCdnEndpointDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMCdnEndpointExists("azurerm_cdn_endpoint.test"),
				),
			},
		},
	})
}

func TestAccAzureRMCdnEndpoints_withTags(t *testing.T) {

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMCdnEndpoint_withTags, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMCdnEndpoint_withTagsUpdate, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMCdnEndpointDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMCdnEndpointExists("azurerm_cdn_endpoint.test"),
					resource.TestCheckResourceAttr(
						"azurerm_cdn_endpoint.test", "tags.#", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_cdn_endpoint.test", "tags.environment", "Production"),
					resource.TestCheckResourceAttr(
						"azurerm_cdn_endpoint.test", "tags.cost_center", "MSFT"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMCdnEndpointExists("azurerm_cdn_endpoint.test"),
					resource.TestCheckResourceAttr(
						"azurerm_cdn_endpoint.test", "tags.#", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_cdn_endpoint.test", "tags.environment", "staging"),
				),
			},
		},
	})
}

func testCheckAzureRMCdnEndpointExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		profileName := rs.Primary.Attributes["profile_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for cdn endpoint: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).cdnEndpointsClient

		resp, err := conn.Get(name, profileName, resourceGroup)
		if err != nil {
			return fmt.Errorf("Bad: Get on cdnEndpointsClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: CDN Endpoint %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMCdnEndpointDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).cdnEndpointsClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_cdn_endpoint" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]
		profileName := rs.Primary.Attributes["profile_name"]

		resp, err := conn.Get(name, profileName, resourceGroup)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("CDN Endpoint still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

var testAccAzureRMCdnEndpoint_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}
resource "azurerm_cdn_profile" "test" {
    name = "acctestcdnprof%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    sku = "Standard"
}

resource "azurerm_cdn_endpoint" "test" {
    name = "acctestcdnend%d"
    profile_name = "${azurerm_cdn_profile.test.name}"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    origin {
	name = "acceptanceTestCdnOrigin1"
	host_name = "www.example.com"
	https_port = 443
	http_port = 80
    }
}
`

var testAccAzureRMCdnEndpoint_withTags = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}
resource "azurerm_cdn_profile" "test" {
    name = "acctestcdnprof%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    sku = "Standard"
}

resource "azurerm_cdn_endpoint" "test" {
    name = "acctestcdnend%d"
    profile_name = "${azurerm_cdn_profile.test.name}"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    origin {
	name = "acceptanceTestCdnOrigin2"
	host_name = "www.example.com"
	https_port = 443
	http_port = 80
    }

    tags {
	environment = "Production"
	cost_center = "MSFT"
    }
}
`

var testAccAzureRMCdnEndpoint_withTagsUpdate = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}
resource "azurerm_cdn_profile" "test" {
    name = "acctestcdnprof%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    sku = "Standard"
}

resource "azurerm_cdn_endpoint" "test" {
    name = "acctestcdnend%d"
    profile_name = "${azurerm_cdn_profile.test.name}"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    origin {
	name = "acceptanceTestCdnOrigin2"
	host_name = "www.example.com"
	https_port = 443
	http_port = 80
    }

    tags {
	environment = "staging"
    }
}
`
