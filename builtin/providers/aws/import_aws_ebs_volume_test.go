package aws

import (
	"testing"

	"github.com/r3labs/terraform/helper/resource"
)

func TestAccAWSEBSVolume_importBasic(t *testing.T) {
	resourceName := "aws_ebs_volume.test"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsEbsVolumeConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
