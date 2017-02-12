package azurerm

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureRMNetworkSecurityRule_importBasic(t *testing.T) {
	rInt := acctest.RandInt()
	resourceName := "azurerm_network_security_rule.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMNetworkSecurityRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMNetworkSecurityRule_basic(rInt),
			},

			resource.TestStep{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"network_security_group_name"},
			},
		},
	})
}
