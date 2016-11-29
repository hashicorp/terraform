package scaleway

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccScalewayIP_importBasic(t *testing.T) {
	resourceName := "scaleway_ip.base"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckScalewayIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckScalewayIPConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
