package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIamAccountAlias_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsIamAccountAliasConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsIamAccountAlias("data.aws_iam_account_alias.current"),
				),
			},
		},
	})
}

func testAccCheckAwsIamAccountAlias(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find Account Alias resource: %s", n)
		}

		if rs.Primary.Attributes["account_alias"] == "" {
			return fmt.Errorf("Missing Account Alias")
		}

		return nil
	}
}

const testAccCheckAwsIamAccountAliasConfig_basic = `
data "aws_iam_account_alias" "current" { }
`
