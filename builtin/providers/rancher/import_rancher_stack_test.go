package rancher

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccRancherStack_importBasic(t *testing.T) {
	resourceName := "rancher_stack.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRancherStackDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRancherStackConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
