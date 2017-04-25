package rabbitmq

import (
	"testing"

	"github.com/michaelklishin/rabbit-hole"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccPolicy_importBasic(t *testing.T) {
	resourceName := "rabbitmq_policy.test"
	var policy rabbithole.Policy

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPolicyCheckDestroy(&policy),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPolicyConfig_basic,
				Check: testAccPolicyCheck(
					resourceName, &policy,
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
