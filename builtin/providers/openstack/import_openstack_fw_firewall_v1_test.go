package openstack

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOpenStackFWFirewallV1_importBasic(t *testing.T) {
	resourceName := "openstack_fw_firewall_v1.accept_test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFWFirewallV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testFirewallConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
