package openstack

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccLBV1VIP_importBasic(t *testing.T) {
	resourceName := "openstack_lb_vip_v1.vip_1"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBV1VIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLBV1VIP_basic,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
