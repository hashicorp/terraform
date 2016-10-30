package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSIAMAccountPasswordPolicy_importBasic(t *testing.T) {
	resourceName := "aws_iam_account_password_policy.default"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSIAMAccountPasswordPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSIAMAccountPasswordPolicy,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
