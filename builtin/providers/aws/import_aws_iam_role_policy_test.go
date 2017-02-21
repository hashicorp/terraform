package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func testAccAwsIamRolePolicyConfig(suffix string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "role_%[1]s" {
	name = "tf_test_role_test_%[1]s"
	path = "/"
	assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Sid\":\"\",\"Effect\":\"Allow\",\"Principal\":{\"Service\":\"ec2.amazonaws.com\"},\"Action\":\"sts:AssumeRole\"}]}"
}

resource "aws_iam_role_policy" "foo_%[1]s" {
	name = "tf_test_policy_test_%[1]s"
	role = "${aws_iam_role.role_%[1]s.name}"
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
`, suffix)
}

func TestAccAWSIAMRolePolicy_importBasic(t *testing.T) {
	suffix := randomString(10)
	resourceName := fmt.Sprintf("aws_iam_role_policy.foo_%s", suffix)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMRolePolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsIamRolePolicyConfig(suffix),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
