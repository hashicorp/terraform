package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSESReceiptFilter_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSESReceiptFilterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSESReceiptFilterConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsSESReceiptFilterExists("aws_ses_receipt_filter.test"),
				),
			},
		},
	})
}

func testAccCheckSESReceiptFilterDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).sesConn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ses_receipt_filter" {
			continue
		}

		response, err := conn.ListReceiptFilters(&ses.ListReceiptFiltersInput{})
		if err != nil {
			return err
		}

		found := false
		for _, element := range response.Filters {
			if *element.Name == "block-some-ip" {
				found = true
			}
		}

		if found {
			return fmt.Errorf("The receipt filter still exists")
		}

	}

	return nil

}

func testAccCheckAwsSESReceiptFilterExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("SES receipt filter not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("SES receipt filter ID not set")
		}

		conn := testAccProvider.Meta().(*AWSClient).sesConn

		response, err := conn.ListReceiptFilters(&ses.ListReceiptFiltersInput{})
		if err != nil {
			return err
		}

		found := false
		for _, element := range response.Filters {
			if *element.Name == "block-some-ip" {
				found = true
			}
		}

		if !found {
			return fmt.Errorf("The receipt filter was not created")
		}

		return nil
	}
}

const testAccAWSSESReceiptFilterConfig = `
resource "aws_ses_receipt_filter" "test" {
    name = "block-some-ip"
    cidr = "10.10.10.10"
    policy = "Block"
}
`
