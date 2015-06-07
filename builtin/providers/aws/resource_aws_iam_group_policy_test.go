package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIAMGroupPolicy_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMGroupPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccIAMGroupPolicyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMGroupPolicy(
						"aws_iam_group.group",
						"aws_iam_group_policy.foo",
					),
				),
			},
			resource.TestStep{
				Config: testAccIAMGroupPolicyConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMGroupPolicy(
						"aws_iam_group.group",
						"aws_iam_group_policy.bar",
					),
				),
			},
		},
	})
}

func testAccCheckIAMGroupPolicyDestroy(s *terraform.State) error {
	if len(s.RootModule().Resources) > 0 {
		return fmt.Errorf("Expected all resources to be gone, but found: %#v", s.RootModule().Resources)
	}

	return nil
}

func testAccCheckIAMGroupPolicy(
	iamGroupResource string,
	iamGroupPolicyResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[iamGroupResource]
		if !ok {
			return fmt.Errorf("Not Found: %s", iamGroupResource)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		policy, ok := s.RootModule().Resources[iamGroupPolicyResource]
		if !ok {
			return fmt.Errorf("Not Found: %s", iamGroupPolicyResource)
		}

		iamconn := testAccProvider.Meta().(*AWSClient).iamconn
		group, name := resourceAwsIamGroupPolicyParseId(policy.Primary.ID)
		_, err := iamconn.GetGroupPolicy(&iam.GetGroupPolicyInput{
			GroupName:  aws.String(group),
			PolicyName: aws.String(name),
		})

		if err != nil {
			return err
		}

		return nil
	}
}

const testAccIAMGroupPolicyConfig = `
resource "aws_iam_group" "group" {
	name = "test_group"
	path = "/"
}

resource "aws_iam_group_policy" "foo" {
	name = "foo_policy"
	group = "${aws_iam_group.group.name}"
	policy = "{\"Version\":\"2012-10-17\",\"Statement\":{\"Effect\":\"Allow\",\"Action\":\"*\",\"Resource\":\"*\"}}"
}
`

const testAccIAMGroupPolicyConfigUpdate = `
resource "aws_iam_group" "group" {
	name = "test_group"
	path = "/"
}

resource "aws_iam_group_policy" "foo" {
	name = "foo_policy"
	group = "${aws_iam_group.group.name}"
	policy = "{\"Version\":\"2012-10-17\",\"Statement\":{\"Effect\":\"Allow\",\"Action\":\"*\",\"Resource\":\"*\"}}"
}

resource "aws_iam_group_policy" "bar" {
	name = "bar_policy"
	group = "${aws_iam_group.group.name}"
	policy = "{\"Version\":\"2012-10-17\",\"Statement\":{\"Effect\":\"Allow\",\"Action\":\"*\",\"Resource\":\"*\"}}"
}
`
