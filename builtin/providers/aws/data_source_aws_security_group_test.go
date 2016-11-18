package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsSecurityGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsSecurityGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsSecurityGroupCheck("data.aws_security_group.by_id"),
					testAccDataSourceAwsSecurityGroupCheck("data.aws_security_group.by_tag"),
					testAccDataSourceAwsSecurityGroupCheck("data.aws_security_group.by_filter"),
					testAccDataSourceAwsSecurityGroupCheck("data.aws_security_group.by_name"),
				),
			},
		},
	})
}

func testAccDataSourceAwsSecurityGroupCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		SGRs, ok := s.RootModule().Resources["aws_security_group.test"]
		if !ok {
			return fmt.Errorf("can't find aws_security_group.test in state")
		}
		vpcRs, ok := s.RootModule().Resources["aws_vpc.test"]
		if !ok {
			return fmt.Errorf("can't find aws_vpc.test in state")
		}
		attr := rs.Primary.Attributes

		if attr["id"] != SGRs.Primary.Attributes["id"] {
			return fmt.Errorf(
				"id is %s; want %s",
				attr["id"],
				SGRs.Primary.Attributes["id"],
			)
		}

		if attr["vpc_id"] != vpcRs.Primary.Attributes["id"] {
			return fmt.Errorf(
				"vpc_id is %s; want %s",
				attr["vpc_id"],
				vpcRs.Primary.Attributes["id"],
			)
		}

		if attr["tags.Name"] != "terraform-testacc-security-group-data-source" {
			return fmt.Errorf("bad Name tag %s", attr["tags.Name"])
		}

		return nil
	}
}

const testAccDataSourceAwsSecurityGroupConfig = `
provider "aws" {
  region = "eu-west-1"
}
resource "aws_vpc" "test" {
  cidr_block = "172.16.0.0/16"

  tags {
    Name = "terraform-testacc-subnet-data-source"
  }
}

resource "aws_security_group" "test" {
  vpc_id = "${aws_vpc.test.id}"
  name = "security-groupe-name-test"
  tags {
    Name = "terraform-testacc-security-group-data-source"
  }
}

data "aws_security_group" "by_id" {
  id = "${aws_security_group.test.id}"
}

data "aws_security_group" "by_name" {
  name = "${aws_security_group.test.name}"
}
data "aws_security_group" "by_tag" {
  tags {
    Name = "${aws_security_group.test.tags["Name"]}"
  }
}

data "aws_security_group" "by_filter" {
  filter {
    name = "group-name"
    values = ["${aws_security_group.test.name}"]
  }
}
`
