package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSEFSMountTarget_importBasic(t *testing.T) {
	resourceName := "aws_efs_mount_target.alpha"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEfsMountTargetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEFSMountTargetConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
