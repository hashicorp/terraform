package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSRolePolicyAttachment_basic(t *testing.T) {
	var out iam.ListAttachedRolePoliciesOutput
	rInt := acctest.RandInt()
	testPolicy := fmt.Sprintf("tf-acctest-%d", rInt)
	testPolicy2 := fmt.Sprintf("tf-acctest2-%d", rInt)
	testPolicy3 := fmt.Sprintf("tf-acctest3-%d", rInt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRolePolicyAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRolePolicyAttachConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRolePolicyAttachmentExists("aws_iam_role_policy_attachment.test-attach", 1, &out),
					testAccCheckAWSRolePolicyAttachmentAttributes([]string{testPolicy}, &out),
				),
			},
			{
				Config: testAccAWSRolePolicyAttachConfigUpdate(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRolePolicyAttachmentExists("aws_iam_role_policy_attachment.test-attach", 2, &out),
					testAccCheckAWSRolePolicyAttachmentAttributes([]string{testPolicy2, testPolicy3}, &out),
				),
			},
		},
	})
}
func testAccCheckAWSRolePolicyAttachmentDestroy(s *terraform.State) error {
	return nil
}

func testAccCheckAWSRolePolicyAttachmentExists(n string, c int, out *iam.ListAttachedRolePoliciesOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No policy name is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).iamconn
		role := rs.Primary.Attributes["role"]

		attachedPolicies, err := conn.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
			RoleName: aws.String(role),
		})
		if err != nil {
			return fmt.Errorf("Error: Failed to get attached policies for role %s (%s)", role, n)
		}
		if c != len(attachedPolicies.AttachedPolicies) {
			return fmt.Errorf("Error: Role (%s) has wrong number of policies attached on initial creation", n)
		}

		*out = *attachedPolicies
		return nil
	}
}
func testAccCheckAWSRolePolicyAttachmentAttributes(policies []string, out *iam.ListAttachedRolePoliciesOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		matched := 0

		for _, p := range policies {
			for _, ap := range out.AttachedPolicies {
				// *ap.PolicyArn like arn:aws:iam::111111111111:policy/test-policy
				parts := strings.Split(*ap.PolicyArn, "/")
				if len(parts) == 2 && p == parts[1] {
					matched++
				}
			}
		}
		if matched != len(policies) || matched != len(out.AttachedPolicies) {
			return fmt.Errorf("Error: Number of attached policies was incorrect: expected %d matched policies, matched %d of %d", len(policies), matched, len(out.AttachedPolicies))
		}
		return nil
	}
}

func testAccAWSRolePolicyAttachConfig(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_iam_role" "role" {
			name = "test-role-%d"
			assume_role_policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Action": "sts:AssumeRole",
			"Principal": {
				"Service": "ec2.amazonaws.com"
			},
			"Effect": "Allow",
			"Sid": ""
		}
	]
}
EOF
	}

	resource "aws_iam_policy" "policy" {
			name = "tf-acctest-%d"
			description = "A test policy"
			policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Action": [
				"iam:ChangePassword"
			],
			"Resource": "*",
			"Effect": "Allow"
		}
	]
}
EOF
	}

	resource "aws_iam_role_policy_attachment" "test-attach" {
			role = "${aws_iam_role.role.name}"
			policy_arn = "${aws_iam_policy.policy.arn}"
	}`, rInt, rInt)
}

func testAccAWSRolePolicyAttachConfigUpdate(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_iam_role" "role" {
			name = "test-role-%d"
			assume_role_policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Action": "sts:AssumeRole",
			"Principal": {
				"Service": "ec2.amazonaws.com"
			},
			"Effect": "Allow",
			"Sid": ""
		}
	]
}
EOF
	}

	resource "aws_iam_policy" "policy" {
			name = "tf-acctest-%d"
			description = "A test policy"
			policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Action": [
				"iam:ChangePassword"
			],
			"Resource": "*",
			"Effect": "Allow"
		}
	]
}
EOF
	}

	resource "aws_iam_policy" "policy2" {
			name = "tf-acctest2-%d"
			description = "A test policy"
			policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Action": [
				"iam:ChangePassword"
			],
			"Resource": "*",
			"Effect": "Allow"
		}
	]
}
EOF
	}

	resource "aws_iam_policy" "policy3" {
			name = "tf-acctest3-%d"
			description = "A test policy"
			policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Action": [
				"iam:ChangePassword"
			],
			"Resource": "*",
			"Effect": "Allow"
		}
	]
}
EOF
	}

	resource "aws_iam_role_policy_attachment" "test-attach" {
			role = "${aws_iam_role.role.name}"
			policy_arn = "${aws_iam_policy.policy2.arn}"
	}

	resource "aws_iam_role_policy_attachment" "test-attach2" {
			role = "${aws_iam_role.role.name}"
			policy_arn = "${aws_iam_policy.policy3.arn}"
	}`, rInt, rInt, rInt, rInt)
}
