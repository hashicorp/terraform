package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIAMUserPolicy_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMUserPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccIAMUserPolicyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMUserPolicy(
						"aws_iam_user.user",
						"aws_iam_user_policy.foo",
					),
				),
			},
			resource.TestStep{
				Config: testAccIAMUserPolicyConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMUserPolicy(
						"aws_iam_user.user",
						"aws_iam_user_policy.bar",
					),
				),
			},
		},
	})
}

func testAccCheckIAMUserPolicyDestroy(s *terraform.State) error {
	if len(s.RootModule().Resources) > 0 {
		return fmt.Errorf("Expected all resources to be gone, but found: %#v", s.RootModule().Resources)
	}

	return nil
}

func testAccCheckIAMUserPolicy(
	iamUserResource string,
	iamUserPolicyResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[iamUserResource]
		if !ok {
			return fmt.Errorf("Not Found: %s", iamUserResource)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		policy, ok := s.RootModule().Resources[iamUserPolicyResource]
		if !ok {
			return fmt.Errorf("Not Found: %s", iamUserPolicyResource)
		}

		iamconn := testAccProvider.Meta().(*AWSClient).iamconn
		username, name := resourceAwsIamUserPolicyParseId(policy.Primary.ID)
		_, err := iamconn.GetUserPolicy(&iam.GetUserPolicyInput{
			UserName:   aws.String(username),
			PolicyName: aws.String(name),
		})

		if err != nil {
			return err
		}

		return nil
	}
}

const testAccIAMUserPolicyConfig = `
resource "aws_iam_user" "user" {
	name = "test_user"
	path = "/"
}

resource "aws_iam_user_policy" "foo" {
	name = "foo_policy"
	user = "${aws_iam_user.user.name}"
	policy = "{\"Version\":\"2012-10-17\",\"Statement\":{\"Effect\":\"Allow\",\"Action\":\"*\",\"Resource\":\"*\"}}"
}
`

const testAccIAMUserPolicyConfigUpdate = `
resource "aws_iam_user" "user" {
	name = "test_user"
	path = "/"
}

resource "aws_iam_user_policy" "foo" {
	name = "foo_policy"
	user = "${aws_iam_user.user.name}"
	policy = "{\"Version\":\"2012-10-17\",\"Statement\":{\"Effect\":\"Allow\",\"Action\":\"*\",\"Resource\":\"*\"}}"
}

resource "aws_iam_user_policy" "bar" {
	name = "bar_policy"
	user = "${aws_iam_user.user.name}"
	policy = "{\"Version\":\"2012-10-17\",\"Statement\":{\"Effect\":\"Allow\",\"Action\":\"*\",\"Resource\":\"*\"}}"
}
`
