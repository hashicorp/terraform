package rabbitmq

import (
	"testing"

	"github.com/michaelklishin/rabbit-hole"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccBinding_importBasic(t *testing.T) {
	resourceName := "rabbitmq_binding.test"
	var bindingInfo rabbithole.BindingInfo

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccBindingCheckDestroy(bindingInfo),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBindingConfig_basic,
				Check: testAccBindingCheck(
					resourceName, &bindingInfo,
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
