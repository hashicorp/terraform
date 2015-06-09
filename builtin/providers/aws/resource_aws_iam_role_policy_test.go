package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIAMRolePolicy_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMRolePolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccIAMRolePolicyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMRolePolicy(
						"aws_iam_role.role",
						"aws_iam_role_policy.foo",
					),
				),
			},
			resource.TestStep{
				Config: testAccIAMRolePolicyConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMRolePolicy(
						"aws_iam_role.role",
						"aws_iam_role_policy.bar",
					),
				),
			},
		},
	})
}

func testAccCheckIAMRolePolicyDestroy(s *terraform.State) error {
	if len(s.RootModule().Resources) > 0 {
		return fmt.Errorf("Expected all resources to be gone, but found: %#v", s.RootModule().Resources)
	}

	return nil
}

func testAccCheckIAMRolePolicy(
	iamRoleResource string,
	iamRolePolicyResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[iamRoleResource]
		if !ok {
			return fmt.Errorf("Not Found: %s", iamRoleResource)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		policy, ok := s.RootModule().Resources[iamRolePolicyResource]
		if !ok {
			return fmt.Errorf("Not Found: %s", iamRolePolicyResource)
		}

		iamconn := testAccProvider.Meta().(*AWSClient).iamconn
		role, name := resourceAwsIamRolePolicyParseId(policy.Primary.ID)
		_, err := iamconn.GetRolePolicy(&iam.GetRolePolicyInput{
			RoleName:   aws.String(role),
			PolicyName: aws.String(name),
		})

		if err != nil {
			return err
		}

		return nil
	}
}

const testAccIAMRolePolicyConfig = `
resource "aws_iam_role" "role" {
	name = "test_role"
	path = "/"
	assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Action\":\"sts:AssumeRole\",\"Principal\":{\"Service\":\"ec2.amazonaws.com\"},\"Effect\":\"Allow\",\"Sid\":\"\"}]}"
}

resource "aws_iam_role_policy" "foo" {
	name = "foo_policy"
	role = "${aws_iam_role.role.name}"
	policy = "{\"Version\":\"2012-10-17\",\"Statement\":{\"Effect\":\"Allow\",\"Action\":\"*\",\"Resource\":\"*\"}}"
}
`

const testAccIAMRolePolicyConfigUpdate = `
resource "aws_iam_role" "role" {
	name = "test_role"
	path = "/"
	assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Action\":\"sts:AssumeRole\",\"Principal\":{\"Service\":\"ec2.amazonaws.com\"},\"Effect\":\"Allow\",\"Sid\":\"\"}]}"
}

resource "aws_iam_role_policy" "foo" {
	name = "foo_policy"
	role = "${aws_iam_role.role.name}"
	policy = "{\"Version\":\"2012-10-17\",\"Statement\":{\"Effect\":\"Allow\",\"Action\":\"*\",\"Resource\":\"*\"}}"
}

resource "aws_iam_role_policy" "bar" {
	name = "bar_policy"
	role = "${aws_iam_role.role.name}"
	policy = "{\"Version\":\"2012-10-17\",\"Statement\":{\"Effect\":\"Allow\",\"Action\":\"*\",\"Resource\":\"*\"}}"
}
`
