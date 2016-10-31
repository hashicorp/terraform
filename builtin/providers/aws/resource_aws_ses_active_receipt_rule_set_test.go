package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSESActiveReceiptRuleSet_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSESActiveReceiptRuleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSESActiveReceiptRuleSetConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsSESActiveReceiptRuleSetExists("aws_ses_active_receipt_rule_set.test"),
				),
			},
		},
	})
}

func testAccCheckSESActiveReceiptRuleSetDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).sesConn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ses_active_receipt_rule_set" {
			continue
		}

		response, err := conn.DescribeActiveReceiptRuleSet(&ses.DescribeActiveReceiptRuleSetInput{})
		if err != nil {
			return err
		}

		if response.Metadata != nil && *response.Metadata.Name == "test-receipt-rule" {
			return fmt.Errorf("Active receipt rule set still exists")
		}

	}

	return nil

}

func testAccCheckAwsSESActiveReceiptRuleSetExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("SES Active Receipt Rule Set not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("SES Active Receipt Rule Set name not set")
		}

		conn := testAccProvider.Meta().(*AWSClient).sesConn

		response, err := conn.DescribeActiveReceiptRuleSet(&ses.DescribeActiveReceiptRuleSetInput{})
		if err != nil {
			return err
		}

		if *response.Metadata.Name != "test-receipt-rule" {
			return fmt.Errorf("The active receipt rule set (%s) was not set to test-receipt-rule", *response.Metadata.Name)
		}

		return nil
	}
}

const testAccAWSSESActiveReceiptRuleSetConfig = `
resource "aws_ses_receipt_rule_set" "test" {
    rule_set_name = "test-receipt-rule"
}

resource "aws_ses_active_receipt_rule_set" "test" {
    rule_set_name = "${aws_ses_receipt_rule_set.test.rule_set_name}"
}
`
