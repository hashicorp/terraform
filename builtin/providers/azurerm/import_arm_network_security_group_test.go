package azurerm

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureRMNetworkSecurityGroup_importBasic(t *testing.T) {
	resourceName := "azurerm_network_security_group.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMNetworkSecurityGroup_basic,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
