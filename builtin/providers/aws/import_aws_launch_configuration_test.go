package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSLaunchConfiguration_importBasic(t *testing.T) {
	resourceName := "aws_launch_configuration.bar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSLaunchConfigurationNoNameConfig,
			},

			resource.TestStep{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"associate_public_ip_address", "user_data"},
			},
		},
	})
}
