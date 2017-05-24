package aws

import (
	"testing"

	"github.com/r3labs/terraform/helper/acctest"
	"github.com/r3labs/terraform/helper/resource"
)

func TestAccAWSIAMRole_importBasic(t *testing.T) {
	resourceName := "aws_iam_role.role"
	rName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSIAMRoleConfig(rName),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
