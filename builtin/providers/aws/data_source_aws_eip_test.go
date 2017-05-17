package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsEip(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsEipConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsEipCheck("data.aws_eip.by_id"),
					testAccDataSourceAwsEipCheck("data.aws_eip.by_public_ip"),
				),
			},
		},
	})
}

func testAccDataSourceAwsEipCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		eipRs, ok := s.RootModule().Resources["aws_eip.test"]
		if !ok {
			return fmt.Errorf("can't find aws_eip.test in state")
		}

		attr := rs.Primary.Attributes

		if attr["id"] != eipRs.Primary.Attributes["id"] {
			return fmt.Errorf(
				"id is %s; want %s",
				attr["id"],
				eipRs.Primary.Attributes["id"],
			)
		}

		if attr["public_ip"] != eipRs.Primary.Attributes["public_ip"] {
			return fmt.Errorf(
				"public_ip is %s; want %s",
				attr["public_ip"],
				eipRs.Primary.Attributes["public_ip"],
			)
		}

		return nil
	}
}

const testAccDataSourceAwsEipConfig = `
provider "aws" {
  region = "us-west-2"
}

resource "aws_eip" "wrong1" {}
resource "aws_eip" "test" {}
resource "aws_eip" "wrong2" {}

data "aws_eip" "by_id" {
  id = "${aws_eip.test.id}"
}

data "aws_eip" "by_public_ip" {
  public_ip = "${aws_eip.test.public_ip}"
}
`
