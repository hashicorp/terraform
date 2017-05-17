package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsRoute53Zone(t *testing.T) {
	rInt := acctest.RandInt()
	publicResourceName := "aws_route53_zone.test"
	publicDomain := fmt.Sprintf("terraformtestacchz-%d.com.", rInt)
	privateResourceName := "aws_route53_zone.test_private"
	privateDomain := fmt.Sprintf("test.acc-%d.", rInt)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsRoute53ZoneConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsRoute53ZoneCheck(
						publicResourceName, "data.aws_route53_zone.by_zone_id", publicDomain),
					testAccDataSourceAwsRoute53ZoneCheck(
						publicResourceName, "data.aws_route53_zone.by_name", publicDomain),
					testAccDataSourceAwsRoute53ZoneCheck(
						privateResourceName, "data.aws_route53_zone.by_vpc", privateDomain),
					testAccDataSourceAwsRoute53ZoneCheck(
						privateResourceName, "data.aws_route53_zone.by_tag", privateDomain),
				),
			},
		},
	})
}

// rsName for the name of the created resource
// dsName for the name of the created data source
// zName for the name of the domain
func testAccDataSourceAwsRoute53ZoneCheck(rsName, dsName, zName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rsName]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", rsName)
		}

		hostedZone, ok := s.RootModule().Resources[dsName]
		if !ok {
			return fmt.Errorf("can't find zone %q in state", dsName)
		}

		attr := rs.Primary.Attributes
		if attr["id"] != hostedZone.Primary.Attributes["id"] {
			return fmt.Errorf(
				"id is %s; want %s",
				attr["id"],
				hostedZone.Primary.Attributes["id"],
			)
		}

		if attr["name"] != zName {
			return fmt.Errorf("Route53 Zone name is %q; want %q", attr["name"], zName)
		}

		return nil
	}
}

func testAccDataSourceAwsRoute53ZoneConfig(rInt int) string {
	return fmt.Sprintf(`
	provider "aws" {
		region = "us-east-1"
	}

	resource "aws_vpc" "test" {
		cidr_block = "172.16.0.0/16"
		tags {
			Name = "testAccDataSourceAwsRoute53ZoneConfig"
		}
	}

	resource "aws_route53_zone" "test_private" {
		name = "test.acc-%d."
		vpc_id = "${aws_vpc.test.id}"
		tags {
			Environment = "dev-%d"
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
		 Environment = "dev-%d"
	 }
	}

	resource "aws_route53_zone" "test" {
		name = "terraformtestacchz-%d.com."
	}

	data "aws_route53_zone" "by_zone_id" {
		zone_id = "${aws_route53_zone.test.zone_id}"
	}

	data "aws_route53_zone" "by_name" {
		name = "${data.aws_route53_zone.by_zone_id.name}"
	}`, rInt, rInt, rInt, rInt)
}
