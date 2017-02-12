package rabbitmq

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccUser_importBasic(t *testing.T) {
	resourceName := "rabbitmq_user.test"
	var user string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccUserCheckDestroy(user),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccUserConfig_basic,
				Check: testAccUserCheck(
					resourceName, &user,
				),
			},

			resource.TestStep{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}
