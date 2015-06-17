package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

func TestAccAWSPolicyAttach_basic(t *testing.T) {
	var policy iam.GetPolicyOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSPolicyAttachmentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSPolicyAttachConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSPolicyAttachmentExists(),
					testAccCheckAWSPolicyAttachmentAttributes(),
				),
			},
			resource.TestStep{
				Config: testAccAWSPolicyAttachConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSPolicyAttachmentExists(),
					testAccCheckAWSPolicyAttachmentAttributes(),
				),
			},
		},
	})
}

func testAccCheckAWSPolicyAttachmentDestroy(s *terraform.State) error {

	return nil
}

func testAccCheckAWSPolicyAttachmentExists(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).iamconn

	return nil
}
func testAccCheckAWSPolicyAttachmentAttributes() error {

	return nil
}

const testAccAWSPolicyAttachConfig = `
resource "aws_iam_user" "user" {
    name = "test-user"
}
resource "aws_iam_role" "role" {
    name = "test-role"
}
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

resource "aws_iam_policy_attach" "test-attach" {
    name = "test-attachment"
    users = ["${aws_iam_user.user.name}"]
    roles = ["${aws_iam_role.role.name}"]
    groups = ["${aws_iam_group.group.name}"]
    policy_arn = "${aws_iam_policy.policy.arn}"
}
`

const testAccAWSPolicyAttachConfigUpdate = `
resource "aws_iam_user" "user" {
    name = "test-user"
}
resource "aws_iam_user" "user2" {
    name = "test-user2"
}
resource "aws_iam_user" "user3" {
    name = "test-user3"
}
resource "aws_iam_role" "role" {
    name = "test-role"
}
resource "aws_iam_role" "role2" {
    name = "test-role2"
}
resource "aws_iam_role" "role3" {
    name = "test-role3"
}
resource "aws_iam_group" "group" {
    name = "test-group"
}
resource "aws_iam_group" "group2" {
    name = "test-group2"
}
resource "aws_iam_group" "group3" {
    name = "test-group3"
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

resource "aws_iam_policy_attach" "test-attach" {
    name = "test-attachment"
    users = [
        "${aws_iam_user.user2.name}",
        "${aws_iam_user.user3.name}"
    ]
    roles = [
        "${aws_iam_role.role2.name}",
        "${aws_iam_role.role3.name}"
    ]
    groups = [
        "${aws_iam_group.group2.name}",
        "${aws_iam_group.group3.name}"
    ]
    policy_arn = "${aws_iam_policy.policy.arn}"
}
`
