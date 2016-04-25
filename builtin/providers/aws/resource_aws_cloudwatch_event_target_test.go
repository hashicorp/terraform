package aws

import (
	"fmt"
	"regexp"
	"testing"

	events "github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCloudWatchEventTarget_basic(t *testing.T) {
	var target events.Target

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchEventTargetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudWatchEventTargetConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchEventTargetExists("aws_cloudwatch_event_target.moobar", &target),
					resource.TestCheckResourceAttr("aws_cloudwatch_event_target.moobar", "rule", "tf-acc-cw-event-rule-basic"),
					resource.TestCheckResourceAttr("aws_cloudwatch_event_target.moobar", "target_id", "tf-acc-cw-target-basic"),
					resource.TestMatchResourceAttr("aws_cloudwatch_event_target.moobar", "arn",
						regexp.MustCompile(":tf-acc-moon$")),
				),
			},
			resource.TestStep{
				Config: testAccAWSCloudWatchEventTargetConfigModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchEventTargetExists("aws_cloudwatch_event_target.moobar", &target),
					resource.TestCheckResourceAttr("aws_cloudwatch_event_target.moobar", "rule", "tf-acc-cw-event-rule-basic"),
					resource.TestCheckResourceAttr("aws_cloudwatch_event_target.moobar", "target_id", "tf-acc-cw-target-modified"),
					resource.TestMatchResourceAttr("aws_cloudwatch_event_target.moobar", "arn",
						regexp.MustCompile(":tf-acc-sun$")),
				),
			},
		},
	})
}

func TestAccAWSCloudWatchEventTarget_missingTargetId(t *testing.T) {
	var target events.Target

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchEventTargetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudWatchEventTargetConfigMissingTargetId,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchEventTargetExists("aws_cloudwatch_event_target.moobar", &target),
					resource.TestCheckResourceAttr("aws_cloudwatch_event_target.moobar", "rule", "tf-acc-cw-event-rule-missing-target-id"),
					resource.TestMatchResourceAttr("aws_cloudwatch_event_target.moobar", "arn",
						regexp.MustCompile(":tf-acc-moon$")),
				),
			},
		},
	})
}

func TestAccAWSCloudWatchEventTarget_full(t *testing.T) {
	var target events.Target

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchEventTargetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudWatchEventTargetConfig_full,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchEventTargetExists("aws_cloudwatch_event_target.foobar", &target),
					resource.TestCheckResourceAttr("aws_cloudwatch_event_target.foobar", "rule", "tf-acc-cw-event-rule-full"),
					resource.TestCheckResourceAttr("aws_cloudwatch_event_target.foobar", "target_id", "tf-acc-cw-target-full"),
					resource.TestMatchResourceAttr("aws_cloudwatch_event_target.foobar", "arn",
						regexp.MustCompile("^arn:aws:kinesis:.*:stream/terraform-kinesis-test$")),
					resource.TestCheckResourceAttr("aws_cloudwatch_event_target.foobar", "input", "{ \"source\": [\"aws.cloudtrail\"] }\n"),
					resource.TestCheckResourceAttr("aws_cloudwatch_event_target.foobar", "input_path", ""),
				),
			},
		},
	})
}

func testAccCheckCloudWatchEventTargetExists(n string, rule *events.Target) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).cloudwatcheventsconn
		t, err := findEventTargetById(rs.Primary.Attributes["target_id"],
			rs.Primary.Attributes["rule"], nil, conn)
		if err != nil {
			return fmt.Errorf("Event Target not found: %s", err)
		}

		*rule = *t

		return nil
	}
}

func testAccCheckAWSCloudWatchEventTargetDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).cloudwatcheventsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_cloudwatch_event_target" {
			continue
		}

		t, err := findEventTargetById(rs.Primary.Attributes["target_id"],
			rs.Primary.Attributes["rule"], nil, conn)
		if err == nil {
			return fmt.Errorf("CloudWatch Event Target %q still exists: %s",
				rs.Primary.ID, t)
		}
	}

	return nil
}

func testAccCheckTargetIdExists(targetId string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[targetId]
		if !ok {
			return fmt.Errorf("Not found: %s", targetId)
		}

		return nil
	}
}

var testAccAWSCloudWatchEventTargetConfig = `
resource "aws_cloudwatch_event_rule" "foo" {
	name = "tf-acc-cw-event-rule-basic"
	schedule_expression = "rate(1 hour)"
}

resource "aws_cloudwatch_event_target" "moobar" {
	rule = "${aws_cloudwatch_event_rule.foo.name}"
	target_id = "tf-acc-cw-target-basic"
	arn = "${aws_sns_topic.moon.arn}"
}

resource "aws_sns_topic" "moon" {
	name = "tf-acc-moon"
}
`

var testAccAWSCloudWatchEventTargetConfigMissingTargetId = `
resource "aws_cloudwatch_event_rule" "foo" {
	name = "tf-acc-cw-event-rule-missing-target-id"
	schedule_expression = "rate(1 hour)"
}

resource "aws_cloudwatch_event_target" "moobar" {
	rule = "${aws_cloudwatch_event_rule.foo.name}"
	arn = "${aws_sns_topic.moon.arn}"
}

resource "aws_sns_topic" "moon" {
	name = "tf-acc-moon"
}
`

var testAccAWSCloudWatchEventTargetConfigModified = `
resource "aws_cloudwatch_event_rule" "foo" {
	name = "tf-acc-cw-event-rule-basic"
	schedule_expression = "rate(1 hour)"
}

resource "aws_cloudwatch_event_target" "moobar" {
	rule = "${aws_cloudwatch_event_rule.foo.name}"
	target_id = "tf-acc-cw-target-modified"
	arn = "${aws_sns_topic.sun.arn}"
}

resource "aws_sns_topic" "sun" {
	name = "tf-acc-sun"
}
`

var testAccAWSCloudWatchEventTargetConfig_full = `
resource "aws_cloudwatch_event_rule" "foo" {
    name = "tf-acc-cw-event-rule-full"
    schedule_expression = "rate(1 hour)"
    role_arn = "${aws_iam_role.role.arn}"
}

resource "aws_iam_role" "role" {
	name = "test_role"
	assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "events.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
POLICY
}

resource "aws_iam_role_policy" "test_policy" {
    name = "test_policy"
    role = "${aws_iam_role.role.id}"
    policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "kinesis:PutRecord",
        "kinesis:PutRecords"
      ],
      "Resource": [
        "*"
      ],
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_cloudwatch_event_target" "foobar" {
	rule = "${aws_cloudwatch_event_rule.foo.name}"
	target_id = "tf-acc-cw-target-full"
	input = <<INPUT
{ "source": ["aws.cloudtrail"] }
INPUT
	arn = "${aws_kinesis_stream.test_stream.arn}"
}

resource "aws_kinesis_stream" "test_stream" {
    name = "terraform-kinesis-test"
    shard_count = 1
}
`
