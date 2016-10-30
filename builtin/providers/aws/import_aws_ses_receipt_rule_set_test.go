package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSSESReceiptRuleSet_importBasic(t *testing.T) {
	resourceName := "aws_ses_receipt_rule_set.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSESReceiptRuleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSESReceiptRuleSetConfig,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
