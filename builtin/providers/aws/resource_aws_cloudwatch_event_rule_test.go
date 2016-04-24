package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	events "github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCloudWatchEventRule_basic(t *testing.T) {
	var rule events.DescribeRuleOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchEventRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudWatchEventRuleConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchEventRuleExists("aws_cloudwatch_event_rule.foo", &rule),
					resource.TestCheckResourceAttr("aws_cloudwatch_event_rule.foo", "name", "tf-acc-cw-event-rule"),
				),
			},
			resource.TestStep{
				Config: testAccAWSCloudWatchEventRuleConfigModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchEventRuleExists("aws_cloudwatch_event_rule.foo", &rule),
					resource.TestCheckResourceAttr("aws_cloudwatch_event_rule.foo", "name", "tf-acc-cw-event-rule-mod"),
				),
			},
		},
	})
}

func TestAccAWSCloudWatchEventRule_full(t *testing.T) {
	var rule events.DescribeRuleOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchEventRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudWatchEventRuleConfig_full,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchEventRuleExists("aws_cloudwatch_event_rule.moobar", &rule),
					resource.TestCheckResourceAttr("aws_cloudwatch_event_rule.moobar", "name", "tf-acc-cw-event-rule-full"),
					resource.TestCheckResourceAttr("aws_cloudwatch_event_rule.moobar", "schedule_expression", "rate(5 minutes)"),
					resource.TestCheckResourceAttr("aws_cloudwatch_event_rule.moobar", "event_pattern", "{\"source\":[\"aws.ec2\"]}"),
					resource.TestCheckResourceAttr("aws_cloudwatch_event_rule.moobar", "description", "He's not dead, he's just resting!"),
					resource.TestCheckResourceAttr("aws_cloudwatch_event_rule.moobar", "role_arn", ""),
					testAccCheckCloudWatchEventRuleEnabled("aws_cloudwatch_event_rule.moobar", "DISABLED", &rule),
				),
			},
		},
	})
}

func TestAccAWSCloudWatchEventRule_enable(t *testing.T) {
	var rule events.DescribeRuleOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchEventRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudWatchEventRuleConfigEnabled,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchEventRuleExists("aws_cloudwatch_event_rule.moo", &rule),
					testAccCheckCloudWatchEventRuleEnabled("aws_cloudwatch_event_rule.moo", "ENABLED", &rule),
				),
			},
			resource.TestStep{
				Config: testAccAWSCloudWatchEventRuleConfigDisabled,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchEventRuleExists("aws_cloudwatch_event_rule.moo", &rule),
					testAccCheckCloudWatchEventRuleEnabled("aws_cloudwatch_event_rule.moo", "DISABLED", &rule),
				),
			},
			resource.TestStep{
				Config: testAccAWSCloudWatchEventRuleConfigEnabled,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchEventRuleExists("aws_cloudwatch_event_rule.moo", &rule),
					testAccCheckCloudWatchEventRuleEnabled("aws_cloudwatch_event_rule.moo", "ENABLED", &rule),
				),
			},
		},
	})
}

func testAccCheckCloudWatchEventRuleExists(n string, rule *events.DescribeRuleOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).cloudwatcheventsconn
		params := events.DescribeRuleInput{
			Name: aws.String(rs.Primary.ID),
		}
		resp, err := conn.DescribeRule(&params)
		if err != nil {
			return err
		}
		if resp == nil {
			return fmt.Errorf("Rule not found")
		}

		*rule = *resp

		return nil
	}
}

func testAccCheckCloudWatchEventRuleEnabled(n string, desired string, rule *events.DescribeRuleOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).cloudwatcheventsconn
		params := events.DescribeRuleInput{
			Name: aws.String(rs.Primary.ID),
		}
		resp, err := conn.DescribeRule(&params)

		if err != nil {
			return err
		}
		if *resp.State != desired {
			return fmt.Errorf("Expected state %q, given %q", desired, *resp.State)
		}

		return nil
	}
}

func testAccCheckAWSCloudWatchEventRuleDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).cloudwatcheventsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_cloudwatch_event_rule" {
			continue
		}

		params := events.DescribeRuleInput{
			Name: aws.String(rs.Primary.ID),
		}

		resp, err := conn.DescribeRule(&params)

		if err == nil {
			return fmt.Errorf("CloudWatch Event Rule %q still exists: %s",
				rs.Primary.ID, resp)
		}
	}

	return nil
}

var testAccAWSCloudWatchEventRuleConfig = `
resource "aws_cloudwatch_event_rule" "foo" {
    name = "tf-acc-cw-event-rule"
    schedule_expression = "rate(1 hour)"
}
`

var testAccAWSCloudWatchEventRuleConfigEnabled = `
resource "aws_cloudwatch_event_rule" "moo" {
    name = "tf-acc-cw-event-rule-state"
    schedule_expression = "rate(1 hour)"
}
`
var testAccAWSCloudWatchEventRuleConfigDisabled = `
resource "aws_cloudwatch_event_rule" "moo" {
    name = "tf-acc-cw-event-rule-state"
    schedule_expression = "rate(1 hour)"
    is_enabled = false
}
`

var testAccAWSCloudWatchEventRuleConfigModified = `
resource "aws_cloudwatch_event_rule" "foo" {
    name = "tf-acc-cw-event-rule-mod"
    schedule_expression = "rate(1 hour)"
}
`

var testAccAWSCloudWatchEventRuleConfig_full = `
resource "aws_cloudwatch_event_rule" "moobar" {
    name = "tf-acc-cw-event-rule-full"
    schedule_expression = "rate(5 minutes)"
	event_pattern = <<PATTERN
{ "source": ["aws.ec2"] }
PATTERN
	description = "He's not dead, he's just resting!"
	is_enabled = false
}
`

// TODO: Figure out example with IAM Role
