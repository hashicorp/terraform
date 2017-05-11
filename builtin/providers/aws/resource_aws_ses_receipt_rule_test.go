package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSESReceiptRule_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSESReceiptRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSESReceiptRuleBasicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsSESReceiptRuleExists("aws_ses_receipt_rule.basic"),
				),
			},
		},
	})
}

func TestAccAWSSESReceiptRule_order(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSESReceiptRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSESReceiptRuleOrderConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsSESReceiptRuleOrder("aws_ses_receipt_rule.second"),
				),
			},
		},
	})
}

func TestAccAWSSESReceiptRule_actions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSESReceiptRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSESReceiptRuleActionsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsSESReceiptRuleActions("aws_ses_receipt_rule.actions"),
				),
			},
		},
	})
}

func testAccCheckSESReceiptRuleDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).sesConn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ses_receipt_rule" {
			continue
		}

		params := &ses.DescribeReceiptRuleInput{
			RuleName:    aws.String(rs.Primary.Attributes["name"]),
			RuleSetName: aws.String(rs.Primary.Attributes["rule_set_name"]),
		}

		_, err := conn.DescribeReceiptRule(params)
		if err == nil {
			return fmt.Errorf("Receipt rule %s still exists. Failing!", rs.Primary.ID)
		}

		// Verify the error is what we want
		_, ok := err.(awserr.Error)
		if !ok {
			return err
		}

	}

	return nil

}

func testAccCheckAwsSESReceiptRuleExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("SES Receipt Rule not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("SES Receipt Rule name not set")
		}

		conn := testAccProvider.Meta().(*AWSClient).sesConn

		params := &ses.DescribeReceiptRuleInput{
			RuleName:    aws.String("basic"),
			RuleSetName: aws.String(fmt.Sprintf("test-me-%d", srrsRandomInt)),
		}

		response, err := conn.DescribeReceiptRule(params)
		if err != nil {
			return err
		}

		if !*response.Rule.Enabled {
			return fmt.Errorf("Enabled (%v) was not set to true", *response.Rule.Enabled)
		}

		if !reflect.DeepEqual(response.Rule.Recipients, []*string{aws.String("test@example.com")}) {
			return fmt.Errorf("Recipients (%v) was not set to [test@example.com]", response.Rule.Recipients)
		}

		if !*response.Rule.ScanEnabled {
			return fmt.Errorf("ScanEnabled (%v) was not set to true", *response.Rule.ScanEnabled)
		}

		if *response.Rule.TlsPolicy != "Require" {
			return fmt.Errorf("TLS Policy (%s) was not set to Require", *response.Rule.TlsPolicy)
		}

		return nil
	}
}

func testAccCheckAwsSESReceiptRuleOrder(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("SES Receipt Rule not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("SES Receipt Rule name not set")
		}

		conn := testAccProvider.Meta().(*AWSClient).sesConn

		params := &ses.DescribeReceiptRuleSetInput{
			RuleSetName: aws.String(fmt.Sprintf("test-me-%d", srrsRandomInt)),
		}

		response, err := conn.DescribeReceiptRuleSet(params)
		if err != nil {
			return err
		}

		if len(response.Rules) != 2 {
			return fmt.Errorf("Number of rules (%d) was not equal to 2", len(response.Rules))
		} else if *response.Rules[0].Name != "first" || *response.Rules[1].Name != "second" {
			return fmt.Errorf("Order of rules (%v) was incorrect", response.Rules)
		}

		return nil
	}
}

func testAccCheckAwsSESReceiptRuleActions(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("SES Receipt Rule not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("SES Receipt Rule name not set")
		}

		conn := testAccProvider.Meta().(*AWSClient).sesConn

		params := &ses.DescribeReceiptRuleInput{
			RuleName:    aws.String("actions4"),
			RuleSetName: aws.String(fmt.Sprintf("test-me-%d", srrsRandomInt)),
		}

		response, err := conn.DescribeReceiptRule(params)
		if err != nil {
			return err
		}

		actions := response.Rule.Actions

		if len(actions) != 3 {
			return fmt.Errorf("Number of rules (%d) was not equal to 3", len(actions))
		}

		addHeaderAction := actions[0].AddHeaderAction
		if *addHeaderAction.HeaderName != "Another-Header" {
			return fmt.Errorf("Header Name (%s) was not equal to Another-Header", *addHeaderAction.HeaderName)
		}

		if *addHeaderAction.HeaderValue != "First" {
			return fmt.Errorf("Header Value (%s) was not equal to First", *addHeaderAction.HeaderValue)
		}

		secondAddHeaderAction := actions[1].AddHeaderAction
		if *secondAddHeaderAction.HeaderName != "Added-By" {
			return fmt.Errorf("Header Name (%s) was not equal to Added-By", *secondAddHeaderAction.HeaderName)
		}

		if *secondAddHeaderAction.HeaderValue != "Terraform" {
			return fmt.Errorf("Header Value (%s) was not equal to Terraform", *secondAddHeaderAction.HeaderValue)
		}

		stopAction := actions[2].StopAction
		if *stopAction.Scope != "RuleSet" {
			return fmt.Errorf("Scope (%s) was not equal to RuleSet", *stopAction.Scope)
		}

		return nil
	}
}

var srrsRandomInt = acctest.RandInt()
var testAccAWSSESReceiptRuleBasicConfig = fmt.Sprintf(`
resource "aws_ses_receipt_rule_set" "test" {
    rule_set_name = "test-me-%d"
}

resource "aws_ses_receipt_rule" "basic" {
    name = "basic"
    rule_set_name = "${aws_ses_receipt_rule_set.test.rule_set_name}"
    recipients = ["test@example.com"]
    enabled = true
    scan_enabled = true
    tls_policy = "Require"
}
`, srrsRandomInt)

var testAccAWSSESReceiptRuleOrderConfig = fmt.Sprintf(`
resource "aws_ses_receipt_rule_set" "test" {
    rule_set_name = "test-me-%d"
}

resource "aws_ses_receipt_rule" "second" {
    name = "second"
    rule_set_name = "${aws_ses_receipt_rule_set.test.rule_set_name}"
    after = "${aws_ses_receipt_rule.first.name}"
}

resource "aws_ses_receipt_rule" "first" {
    name = "first"
    rule_set_name = "${aws_ses_receipt_rule_set.test.rule_set_name}"
}
`, srrsRandomInt)

var testAccAWSSESReceiptRuleActionsConfig = fmt.Sprintf(`
resource "aws_s3_bucket" "emails" {
    bucket = "ses-terraform-emails"
}

resource "aws_ses_receipt_rule_set" "test" {
    rule_set_name = "test-me-%d"
}

resource "aws_ses_receipt_rule" "actions" {
    name = "actions4"
    rule_set_name = "${aws_ses_receipt_rule_set.test.rule_set_name}"

    add_header_action {
			header_name = "Added-By"
			header_value = "Terraform"
			position = 2
    }

    add_header_action {
			header_name = "Another-Header"
			header_value = "First"
			position = 1
    }

    stop_action {
			scope = "RuleSet"
			position = 3
    }
}
`, srrsRandomInt)
