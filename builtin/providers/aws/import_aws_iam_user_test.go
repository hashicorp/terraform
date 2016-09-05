package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSUser_importBasic(t *testing.T) {
	resourceName := "aws_iam_user.user"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSUserDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSUserConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"force_destroy"},
			},
		},
	})
}
