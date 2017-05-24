package rancher

import (
	"testing"

	"github.com/r3labs/terraform/helper/resource"
)

func TestAccRancherEnvironment_importBasic(t *testing.T) {
	resourceName := "rancher_environment.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRancherEnvironmentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRancherEnvironmentConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
