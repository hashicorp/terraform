package rabbitmq

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccVhost_importBasic(t *testing.T) {
	resourceName := "rabbitmq_vhost.test"
	var vhost string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccVhostCheckDestroy(vhost),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVhostConfig_basic,
				Check: testAccVhostCheck(
					resourceName, &vhost,
				),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
