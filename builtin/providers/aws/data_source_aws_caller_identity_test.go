package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCallerIdentity_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsCallerIdentityConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsCallerIdentityAccountId("data.aws_caller_identity.current"),
				),
			},
		},
	})
}

func testAccCheckAwsCallerIdentityAccountId(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find AccountID resource: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Account Id resource ID not set.")
		}

		expectedAccountId := testAccProvider.Meta().(*AWSClient).accountid
		expectedResource := testAccProvider.Meta().(*AWSClient).resource
		if rs.Primary.Attributes["account_id"] != expectedAccountId {
			return fmt.Errorf("Incorrect Account ID: expected %q, got %q", expectedAccountId, rs.Primary.Attributes["account_id"])
		}

		if rs.Primary.Attributes["resource"] == expectedResource {
			return fmt.Errorf("Incorrect resource: expected %q, got %q", expectedResource, rs.Primary.Attributes["resource"])
		}

		return nil
	}
}

const testAccCheckAwsCallerIdentityConfig_basic = `
data "aws_caller_identity" "current" { }
`
