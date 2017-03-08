package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsVpc_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsVpcConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsVpcCheck("data.aws_vpc.by_id"),
					testAccDataSourceAwsVpcCheck("data.aws_vpc.by_cidr"),
					testAccDataSourceAwsVpcCheck("data.aws_vpc.by_tag"),
					testAccDataSourceAwsVpcCheck("data.aws_vpc.by_filter"),
				),
			},
		},
	})
}

func TestAccDataSourceAwsVpc_ipv6Associated(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsVpcConfigIpv6,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsVpcCheck("data.aws_vpc.by_id"),
					resource.TestCheckResourceAttrSet(
						"data.aws_vpc.by_id", "ipv6_association_id"),
					resource.TestCheckResourceAttrSet(
						"data.aws_vpc.by_id", "ipv6_cidr_block"),
				),
			},
		},
	})
}

func testAccDataSourceAwsVpcCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		vpcRs, ok := s.RootModule().Resources["aws_vpc.test"]
		if !ok {
			return fmt.Errorf("can't find aws_vpc.test in state")
		}

		attr := rs.Primary.Attributes

		if attr["id"] != vpcRs.Primary.Attributes["id"] {
			return fmt.Errorf(
				"id is %s; want %s",
				attr["id"],
				vpcRs.Primary.Attributes["id"],
			)
		}

		if attr["cidr_block"] != "172.16.0.0/16" {
			return fmt.Errorf("bad cidr_block %s", attr["cidr_block"])
		}
		if attr["tags.Name"] != "terraform-testacc-vpc-data-source" {
			return fmt.Errorf("bad Name tag %s", attr["tags.Name"])
		}

		return nil
	}
}

const testAccDataSourceAwsVpcConfigIpv6 = `
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "test" {
  cidr_block = "172.16.0.0/16"
  assign_generated_ipv6_cidr_block = true

  tags {
    Name = "terraform-testacc-vpc-data-source"
  }
}

data "aws_vpc" "by_id" {
  id = "${aws_vpc.test.id}"
}
`

const testAccDataSourceAwsVpcConfig = `
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "test" {
  cidr_block = "172.16.0.0/16"

  tags {
    Name = "terraform-testacc-vpc-data-source"
  }
}

data "aws_vpc" "by_id" {
  id = "${aws_vpc.test.id}"
}

data "aws_vpc" "by_cidr" {
  cidr_block = "${aws_vpc.test.cidr_block}"
}

data "aws_vpc" "by_tag" {
  tags {
    Name = "${aws_vpc.test.tags["Name"]}"
  }
}

data "aws_vpc" "by_filter" {
  filter {
    name = "cidr"
    values = ["${aws_vpc.test.cidr_block}"]
  }
}
`
