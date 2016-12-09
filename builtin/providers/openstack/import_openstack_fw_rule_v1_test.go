package openstack

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOpenStackFWRuleV1_importBasic(t *testing.T) {
	resourceName := "openstack_fw_rule_v1.accept_test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFWRuleV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testFirewallRuleConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
