package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSIAMGroup_importBasic(t *testing.T) {
	resourceName := "aws_iam_group.group"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSGroupConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
