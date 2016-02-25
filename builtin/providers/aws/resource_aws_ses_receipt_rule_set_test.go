package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSESReceiptRuleSet_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSESReceiptRuleSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSESReceiptRuleSetConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsSESReceiptRuleSetExists("aws_ses_receipt_rule_set.test"),
				),
			},
		},
	})
}

func testAccCheckSESReceiptRuleSetDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).sesConn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ses_receipt_rule_set" {
			continue
		}

		params := &ses.DescribeReceiptRuleSetInput{
			RuleSetName: aws.String("just-a-test"),
		}

		_, err := conn.DescribeReceiptRuleSet(params)
		if err == nil {
			return fmt.Errorf("Receipt rule set %s still exists. Failing!", rs.Primary.ID)
		}

		// Verify the error is what we want
		_, ok := err.(awserr.Error)
		if !ok {
			return err
		}

	}

	return nil

}

func testAccCheckAwsSESReceiptRuleSetExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("SES Receipt Rule Set not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("SES Receipt Rule Set name not set")
		}

		conn := testAccProvider.Meta().(*AWSClient).sesConn

		params := &ses.DescribeReceiptRuleSetInput{
			RuleSetName: aws.String("just-a-test"),
		}

		_, err := conn.DescribeReceiptRuleSet(params)
		if err != nil {
			return err
		}

		return nil
	}
}

const testAccAWSSESReceiptRuleSetConfig = `
resource "aws_ses_receipt_rule_set" "test" {
    rule_set_name = "just-a-test"
}
`
