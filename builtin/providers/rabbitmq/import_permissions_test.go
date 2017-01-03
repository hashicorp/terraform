package rabbitmq

import (
	"testing"

	"github.com/michaelklishin/rabbit-hole"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccPermissions_importBasic(t *testing.T) {
	resourceName := "rabbitmq_permissions.test"
	var permissionInfo rabbithole.PermissionInfo

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPermissionsCheckDestroy(&permissionInfo),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPermissionsConfig_basic,
				Check: testAccPermissionsCheck(
					resourceName, &permissionInfo,
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
