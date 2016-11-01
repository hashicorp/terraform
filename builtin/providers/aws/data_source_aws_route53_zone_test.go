package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsRoute53Zone(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsRoute53ZoneConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsRoute53ZoneCheck("data.aws_route53_zone.by_zone_id"),
					testAccDataSourceAwsRoute53ZoneCheck("data.aws_route53_zone.by_name"),
				),
			},
		},
	})
}

func testAccDataSourceAwsRoute53ZoneCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		hostedZone, ok := s.RootModule().Resources["aws_route53_zone.test"]
		if !ok {
			return fmt.Errorf("can't find aws_hosted_zone.test in state")
		}
		attr := rs.Primary.Attributes
		if attr["id"] != hostedZone.Primary.Attributes["id"] {
			return fmt.Errorf(
				"id is %s; want %s",
				attr["id"],
				hostedZone.Primary.Attributes["id"],
			)
		}

		if attr["name"] != "terraformtestacchz.com." {
			return fmt.Errorf(
				"Route53 Zone name is %s; want terraformtestacchz.com.",
				attr["name"],
			)
		}

		return nil
	}
}

const testAccDataSourceAwsRoute53ZoneConfig = `

provider "aws" {
  region = "us-east-2"
}
resource "aws_route53_zone" "test" {
  name = "terraformtestacchz.com."
}
data "aws_route53_zone" "by_zone_id" {
  zone_id = "${aws_route53_zone.test.zone_id}"
}

data "aws_route53_zone" "by_name" {
  name = "${data.aws_route53_zone.by_zone_id.name}"
}


`
