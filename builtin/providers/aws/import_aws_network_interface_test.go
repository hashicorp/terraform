package aws

import (
	"testing"

	"github.com/r3labs/terraform/helper/resource"
)

func TestAccAWSENI_importBasic(t *testing.T) {
	resourceName := "aws_network_interface.bar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSENIDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSENIConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
