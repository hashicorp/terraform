package openstack

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOpenStackFWPolicyV1_importBasic(t *testing.T) {
	resourceName := "openstack_fw_policy_v1.accept_test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFWPolicyV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testFirewallPolicyConfigAddRules,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
