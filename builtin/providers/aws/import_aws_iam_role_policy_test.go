package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

var testAccAwsIamRolePolicyConfig = `
resource "aws_iam_role" "role" {
	name = "tf_test_role_test"
	path = "/"
	assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Sid\":\"\",\"Effect\":\"Allow\",\"Principal\":{\"Service\":\"ec2.amazonaws.com\"},\"Action\":\"sts:AssumeRole\"}]}"
}

resource "aws_iam_role_policy" "foo" {
	name = "tf_test_policy_test"
	role = "${aws_iam_role.role.name}"
	policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": {
    "Effect": "Allow",
    "Action": "*",
    "Resource": "*"
  }
}
EOF
}
`

func TestAccAWSIAMRolePolicy_importBasic(t *testing.T) {
	resourceName := "aws_iam_role_policy.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMRolePolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsIamRolePolicyConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
