package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSCloudwatchLogDestination_importBasic(t *testing.T) {
	resourceName := "aws_cloudwatch_log_destination.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudwatchLogDestinationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudwatchLogDestinationConfig(),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
