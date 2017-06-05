package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsKmsAlias(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsKmsAlias(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsKmsAliasCheck("data.aws_kms_alias.by_name"),
				),
			},
		},
	})
}

func testAccDataSourceAwsKmsAliasCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		kmsKeyRs, ok := s.RootModule().Resources["aws_kms_alias.single"]
		if !ok {
			return fmt.Errorf("can't find aws_kms_alias.single in state")
		}

		attr := rs.Primary.Attributes

		if attr["arn"] != kmsKeyRs.Primary.Attributes["arn"] {
			return fmt.Errorf(
				"arn is %s; want %s",
				attr["arn"],
				kmsKeyRs.Primary.Attributes["arn"],
			)
		}

		if attr["target_key_id"] != kmsKeyRs.Primary.Attributes["target_key_id"] {
			return fmt.Errorf(
				"target_key_id is %s; want %s",
				attr["target_key_id"],
				kmsKeyRs.Primary.Attributes["target_key_id"],
			)
		}

		return nil
	}
}

func testAccDataSourceAwsKmsAlias(rInt int) string {
	return fmt.Sprintf(`
resource "aws_kms_key" "one" {
    description = "Terraform acc test"
    deletion_window_in_days = 7
}

resource "aws_kms_alias" "single" {
    name = "alias/tf-acc-key-alias-%d"
    target_key_id = "${aws_kms_key.one.key_id}"
}

data "aws_kms_alias" "by_name" {
  name = "${aws_kms_alias.single.name}"
}`, rInt)
}
