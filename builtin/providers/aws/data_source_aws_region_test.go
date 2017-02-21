package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsRegion(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsRegionConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsRegionCheck("data.aws_region.by_name_current", "us-west-2", "true"),
					testAccDataSourceAwsRegionCheck("data.aws_region.by_name_other", "us-west-1", "false"),
					testAccDataSourceAwsRegionCheck("data.aws_region.by_current", "us-west-2", "true"),
				),
			},
		},
	})
}

func testAccDataSourceAwsRegionCheck(name, region, current string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		attr := rs.Primary.Attributes

		if attr["name"] != region {
			return fmt.Errorf("bad name %s", attr["name"])
		}
		if attr["current"] != current {
			return fmt.Errorf("bad current %s; want %s", attr["current"], current)
		}

		return nil
	}
}

const testAccDataSourceAwsRegionConfig = `
provider "aws" {
  region = "us-west-2"
}

data "aws_region" "by_name_current" {
  name = "us-west-2"
}

data "aws_region" "by_name_other" {
  name = "us-west-1"
}

data "aws_region" "by_current" {
  current = true
}
`
