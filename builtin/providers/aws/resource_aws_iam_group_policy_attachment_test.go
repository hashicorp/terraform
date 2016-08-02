package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIamGroupPolicyAttachment_basic(t *testing.T) {
	var out iam.ListAttachedGroupPoliciesOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGroupPolicyAttachmentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSGroupPolicyAttachConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGroupPolicyAttachmentExists("aws_iam_group_policy_attachment.test-attach", 1, &out),
					testAccCheckAWSGroupPolicyAttachmentAttributes([]string{"test-policy"}, &out),
				),
			},
			resource.TestStep{
				Config: testAccAWSGroupPolicyAttachConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGroupPolicyAttachmentExists("aws_iam_group_policy_attachment.test-attach", 2, &out),
					testAccCheckAWSGroupPolicyAttachmentAttributes([]string{"test-policy2", "test-policy3"}, &out),
				),
			},
		},
	})
}
func testAccCheckAWSGroupPolicyAttachmentDestroy(s *terraform.State) error {
	return nil
}

func testAccCheckAWSGroupPolicyAttachmentExists(n string, c int, out *iam.ListAttachedGroupPoliciesOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No policy name is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).iamconn
		group := rs.Primary.Attributes["group"]

		attachedPolicies, err := conn.ListAttachedGroupPolicies(&iam.ListAttachedGroupPoliciesInput{
			GroupName: aws.String(group),
		})
		if err != nil {
			return fmt.Errorf("Error: Failed to get attached policies for group %s (%s)", group, n)
		}
		if c != len(attachedPolicies.AttachedPolicies) {
			return fmt.Errorf("Error: Group (%s) has wrong number of policies attached on initial creation", n)
		}

		*out = *attachedPolicies
		return nil
	}
}
func testAccCheckAWSGroupPolicyAttachmentAttributes(policies []string, out *iam.ListAttachedGroupPoliciesOutput) resource.TestCheckFunc {
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

const testAccAWSGroupPolicyAttachConfig = `
resource "aws_iam_group" "group" {
    name = "test-group"
}

resource "aws_iam_policy" "policy" {
    name = "test-policy"
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

resource "aws_iam_group_policy_attachment" "test-attach" {
    group = "${aws_iam_group.group.name}"
    policy_arn = "${aws_iam_policy.policy.arn}"
}
`

const testAccAWSGroupPolicyAttachConfigUpdate = `
resource "aws_iam_group" "group" {
    name = "test-group"
}

resource "aws_iam_policy" "policy" {
    name = "test-policy"
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
    name = "test-policy2"
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
    name = "test-policy3"
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

resource "aws_iam_group_policy_attachment" "test-attach" {
    group = "${aws_iam_group.group.name}"
    policy_arn = "${aws_iam_policy.policy2.arn}"
}

resource "aws_iam_group_policy_attachment" "test-attach2" {
    group = "${aws_iam_group.group.name}"
    policy_arn = "${aws_iam_policy.policy3.arn}"
}
`
