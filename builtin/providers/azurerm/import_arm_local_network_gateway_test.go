package azurerm

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureRMLocalNetworkGateway_importBasic(t *testing.T) {
	resourceName := "azurerm_local_network_gateway.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLocalNetworkGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMLocalNetworkGatewayConfig_basic,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
