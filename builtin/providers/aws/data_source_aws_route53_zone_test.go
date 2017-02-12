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
					testAccDataSourceAwsRoute53ZoneCheckPrivate("data.aws_route53_zone.by_vpc"),
					testAccDataSourceAwsRoute53ZoneCheckPrivate("data.aws_route53_zone.by_tag"),
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

func testAccDataSourceAwsRoute53ZoneCheckPrivate(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		hostedZone, ok := s.RootModule().Resources["aws_route53_zone.test_private"]
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

		if attr["name"] != "test.acc." {
			return fmt.Errorf(
				"Route53 Zone name is %s; want test.acc.",
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

resource "aws_vpc" "test" {
  cidr_block = "172.16.0.0/16"
}

resource "aws_route53_zone" "test_private" {
  name = "test.acc."
  vpc_id = "${aws_vpc.test.id}"
  tags {
    Environment = "dev"
  }
}
data "aws_route53_zone" "by_vpc" {
 name = "${aws_route53_zone.test_private.name}"
 vpc_id = "${aws_vpc.test.id}"
}

data "aws_route53_zone" "by_tag" {
 name = "${aws_route53_zone.test_private.name}"
 private_zone = true
 tags {
 	Environment = "dev"
 }
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
