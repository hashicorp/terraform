package openstack

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccLBV1Pool_importBasic(t *testing.T) {
	resourceName := "openstack_lb_pool_v1.pool_1"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBV1PoolDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLBV1Pool_basic,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
