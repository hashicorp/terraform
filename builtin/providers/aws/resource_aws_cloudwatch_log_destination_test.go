package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCloudwatchLogDestination_basic(t *testing.T) {
	var destination cloudwatchlogs.Destination

	rstring := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudwatchLogDestinationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudwatchLogDestinationConfig(rstring),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCloudwatchLogDestinationExists("aws_cloudwatch_log_destination.test", &destination),
				),
			},
		},
	})
}

func testAccCheckAWSCloudwatchLogDestinationDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).cloudwatchlogsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_cloudwatch_log_destination" {
			continue
		}
		_, exists, err := lookupCloudWatchLogDestination(conn, rs.Primary.ID, nil)
		if err != nil {
			return nil
		}

		if exists {
			return fmt.Errorf("Bad: Destination still exists: %q", rs.Primary.ID)
		}
	}

	return nil

}

func testAccCheckAWSCloudwatchLogDestinationExists(n string, d *cloudwatchlogs.Destination) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).cloudwatchlogsconn
		destination, exists, err := lookupCloudWatchLogDestination(conn, rs.Primary.ID, nil)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("Bad: Destination %q does not exist", rs.Primary.ID)
		}

		*d = *destination

		return nil
	}
}

func testAccAWSCloudwatchLogDestinationConfig(rstring string) string {
	return fmt.Sprintf(`
resource "aws_kinesis_stream" "test" {
  name = "RootAccess_%s"
  shard_count = 1
}

data "aws_region" "current" {
  current = true
}

data "aws_iam_policy_document" "role" {
  statement {
    effect = "Allow"
    principals = {
      type = "Service"
      identifiers = [
        "logs.${data.aws_region.current.name}.amazonaws.com"
      ]
    }
    actions = [
      "sts:AssumeRole",
    ]
  }
}

resource "aws_iam_role" "test" {
  name = "CWLtoKinesisRole_%s"
  assume_role_policy = "${data.aws_iam_policy_document.role.json}"
}

data "aws_iam_policy_document" "policy" {
  statement {
    effect = "Allow"
    actions = [
      "kinesis:PutRecord",
    ]
    resources = [
      "${aws_kinesis_stream.test.arn}"
    ]
  }
  statement {
    effect = "Allow"
    actions = [
      "iam:PassRole"
    ]
    resources = [
      "${aws_iam_role.test.arn}"
    ]
  }
}

resource "aws_iam_role_policy" "test" {
  name = "Permissions-Policy-For-CWL_%s"
  role = "${aws_iam_role.test.id}"
  policy = "${data.aws_iam_policy_document.policy.json}"
}

resource "aws_cloudwatch_log_destination" "test" {
  name = "testDestination_%s"
  target_arn = "${aws_kinesis_stream.test.arn}"
  role_arn = "${aws_iam_role.test.arn}"
  depends_on = ["aws_iam_role_policy.test"]
}

data "aws_iam_policy_document" "access" {
  statement {
    effect = "Allow"
    principals = {
      type = "AWS"
      identifiers = [
        "000000000000"
      ]
    }
    actions = [
      "logs:PutSubscriptionFilter"
    ]
    resources = [
      "${aws_cloudwatch_log_destination.test.arn}"
    ]
  }
}

resource "aws_cloudwatch_log_destination_policy" "test" {
  destination_name = "${aws_cloudwatch_log_destination.test.name}"
  access_policy = "${data.aws_iam_policy_document.access.json}"
}
`, rstring, rstring, rstring, rstring)
}
