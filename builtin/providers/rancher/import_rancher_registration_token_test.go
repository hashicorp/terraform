package rancher

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccRancherRegistrationToken_importBasic(t *testing.T) {
	resourceName := "rancher_registration_token.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRancherRegistrationTokenDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRancherRegistrationTokenConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
