package openstack

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccLBV1Monitor_importBasic(t *testing.T) {
	resourceName := "openstack_lb_monitor_v1.monitor_1"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBV1MonitorDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLBV1Monitor_basic,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
