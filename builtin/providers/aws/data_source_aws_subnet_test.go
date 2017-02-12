package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsSubnet(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsSubnetConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsSubnetCheck("data.aws_subnet.by_id"),
					testAccDataSourceAwsSubnetCheck("data.aws_subnet.by_cidr"),
					testAccDataSourceAwsSubnetCheck("data.aws_subnet.by_tag"),
					testAccDataSourceAwsSubnetCheck("data.aws_subnet.by_vpc"),
					testAccDataSourceAwsSubnetCheck("data.aws_subnet.by_filter"),
				),
			},
		},
	})
}

func testAccDataSourceAwsSubnetCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		vpcRs, ok := s.RootModule().Resources["aws_vpc.test"]
		if !ok {
			return fmt.Errorf("can't find aws_vpc.test in state")
		}
		subnetRs, ok := s.RootModule().Resources["aws_subnet.test"]
		if !ok {
			return fmt.Errorf("can't find aws_subnet.test in state")
		}

		attr := rs.Primary.Attributes

		if attr["id"] != subnetRs.Primary.Attributes["id"] {
			return fmt.Errorf(
				"id is %s; want %s",
				attr["id"],
				subnetRs.Primary.Attributes["id"],
			)
		}

		if attr["vpc_id"] != vpcRs.Primary.Attributes["id"] {
			return fmt.Errorf(
				"vpc_id is %s; want %s",
				attr["vpc_id"],
				vpcRs.Primary.Attributes["id"],
			)
		}

		if attr["cidr_block"] != "172.16.123.0/24" {
			return fmt.Errorf("bad cidr_block %s", attr["cidr_block"])
		}
		if attr["availability_zone"] != "us-west-2a" {
			return fmt.Errorf("bad availability_zone %s", attr["availability_zone"])
		}
		if attr["tags.Name"] != "terraform-testacc-subnet-data-source" {
			return fmt.Errorf("bad Name tag %s", attr["tags.Name"])
		}

		return nil
	}
}

const testAccDataSourceAwsSubnetConfig = `
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "test" {
  cidr_block = "172.16.0.0/16"

  tags {
    Name = "terraform-testacc-subnet-data-source"
  }
}

resource "aws_subnet" "test" {
  vpc_id            = "${aws_vpc.test.id}"
  cidr_block        = "172.16.123.0/24"
  availability_zone = "us-west-2a"

  tags {
    Name = "terraform-testacc-subnet-data-source"
  }
}

data "aws_subnet" "by_id" {
  id = "${aws_subnet.test.id}"
}

data "aws_subnet" "by_cidr" {
  cidr_block = "${aws_subnet.test.cidr_block}"
}

data "aws_subnet" "by_tag" {
  tags {
    Name = "${aws_subnet.test.tags["Name"]}"
  }
}

data "aws_subnet" "by_vpc" {
  vpc_id = "${aws_subnet.test.vpc_id}"
}

data "aws_subnet" "by_filter" {
  filter {
    name = "vpc-id"
    values = ["${aws_subnet.test.vpc_id}"]
  }
}
`
