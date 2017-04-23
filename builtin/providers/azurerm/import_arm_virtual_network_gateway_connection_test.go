package azurerm

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccAzureRMVirtualNetworkGatewayConnection_importSiteToSite(t *testing.T) {
	resourceName := "azurerm_virtual_network_gateway_connection.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualNetworkGatewayConnection_sitetosite, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualNetworkGatewayConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
