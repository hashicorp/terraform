package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSSfnActivity_importBasic(t *testing.T) {
	resourceName := "aws_sfn_activity.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSfnActivityDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSfnActivityBasicConfig(resourceName),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
