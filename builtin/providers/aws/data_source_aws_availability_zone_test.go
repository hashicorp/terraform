package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsAvailabilityZone(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsAvailabilityZoneConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsAvailabilityZoneCheck("data.aws_availability_zone.by_name"),
				),
			},
		},
	})
}

func testAccDataSourceAwsAvailabilityZoneCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		attr := rs.Primary.Attributes

		if attr["name"] != "us-west-2a" {
			return fmt.Errorf("bad name %s", attr["name"])
		}
		if attr["name_suffix"] != "a" {
			return fmt.Errorf("bad name_suffix %s", attr["name_suffix"])
		}
		if attr["region"] != "us-west-2" {
			return fmt.Errorf("bad region %s", attr["region"])
		}

		return nil
	}
}

const testAccDataSourceAwsAvailabilityZoneConfig = `
provider "aws" {
  region = "us-west-2"
}

data "aws_availability_zone" "by_name" {
  name = "us-west-2a"
}
`
