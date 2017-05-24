package aws

import (
	"testing"

	"github.com/r3labs/terraform/helper/acctest"
	"github.com/r3labs/terraform/helper/resource"
)

func TestAccAWSIAMGroup_importBasic(t *testing.T) {
	rInt := acctest.RandInt()
	resourceName := "aws_iam_group.group"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGroupConfig(rInt),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
