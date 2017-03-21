package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIAMAccountAlias_basic(t *testing.T) {
	var account_alias string

	rstring := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSIAMAccountAliasDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSIAMAccountAliasConfig(rstring),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSIAMAccountAliasExists("aws_iam_account_alias.test", &account_alias),
				),
			},
		},
	})
}

func testAccCheckAWSIAMAccountAliasDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).iamconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iam_account_alias" {
			continue
		}

		params := &iam.ListAccountAliasesInput{}

		resp, err := conn.ListAccountAliases(params)

		if err != nil || resp == nil {
			return nil
		}

		if len(resp.AccountAliases) > 0 {
			return fmt.Errorf("Bad: Account alias still exists: %q", rs.Primary.ID)
		}
	}

	return nil

}

func testAccCheckAWSIAMAccountAliasExists(n string, a *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).iamconn
		params := &iam.ListAccountAliasesInput{}

		resp, err := conn.ListAccountAliases(params)

		if err != nil || resp == nil {
			return nil
		}

		if len(resp.AccountAliases) == 0 {
			return fmt.Errorf("Bad: Account alias %q does not exist", rs.Primary.ID)
		}

		*a = aws.StringValue(resp.AccountAliases[0])

		return nil
	}
}

func testAccAWSIAMAccountAliasConfig(rstring string) string {
	return fmt.Sprintf(`
resource "aws_iam_account_alias" "test" {
  account_alias = "terraform-%s-alias"
}
`, rstring)
}
