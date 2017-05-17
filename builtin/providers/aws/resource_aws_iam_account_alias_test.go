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

func TestAccAWSIAMAccountAlias(t *testing.T) {
	testCases := map[string]map[string]func(t *testing.T){
		"Basic": {
			"basic": testAccAWSIAMAccountAlias_basic_with_datasource,
		},
		"Import": {
			"import": testAccAWSIAMAccountAlias_importBasic,
		},
	}

	for group, m := range testCases {
		m := m
		t.Run(group, func(t *testing.T) {
			for name, tc := range m {
				tc := tc
				t.Run(name, func(t *testing.T) {
					tc(t)
				})
			}
		})
	}
}

func testAccAWSIAMAccountAlias_basic_with_datasource(t *testing.T) {
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
			{
				Config: testAccAWSIAMAccountAliasConfig_with_datasource(rstring),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSIAMAccountAliasExists("aws_iam_account_alias.test", &account_alias),
					testAccCheckAWSIAMAccountAliasDataExists("data.aws_iam_account_alias.current", &account_alias),
				),
				// We expect a non-empty plan due to the way data sources and depends_on
				// work, or don't work. See https://github.com/hashicorp/terraform/issues/11139#issuecomment-275121893
				// We accept this limitation and feel this test is OK because of the
				// explicity check above
				ExpectNonEmptyPlan: true,
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

func testAccCheckAWSIAMAccountAliasDataExists(n string, a *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.Attributes["account_alias"] != *a {
			return fmt.Errorf("Data Source account_alias didn't match, expected (%s), got (%s)", *a, rs.Primary.Attributes["account_alias"])
		}

		return nil
	}
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

func testAccAWSIAMAccountAliasConfig_with_datasource(rstring string) string {
	return fmt.Sprintf(`
resource "aws_iam_account_alias" "test" {
  account_alias = "terraform-%s-alias"
}

data "aws_iam_account_alias" "current" {
  depends_on = ["aws_iam_account_alias.test"]
}`, rstring)
}

func testAccAWSIAMAccountAliasConfig(rstring string) string {
	return fmt.Sprintf(`
resource "aws_iam_account_alias" "test" {
  account_alias = "terraform-%s-alias"
}`, rstring)
}
