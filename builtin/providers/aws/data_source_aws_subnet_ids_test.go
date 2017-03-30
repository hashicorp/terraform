package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsSubnetIDs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsSubnetIDsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsSubnetIDsCheck("data.aws_subnet_ids.selected"),
				),
			},
		},
	})
}

func testAccDataSourceAwsSubnetIDsCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		vpcRs, ok := s.RootModule().Resources["aws_vpc.test"]
		if !ok {
			return fmt.Errorf("can't find aws_vpc.test in state")
		}
		_, ok = s.RootModule().Resources["aws_subnet.test"]
		if !ok {
			return fmt.Errorf("can't find aws_subnet.test in state")
		}

		attr := rs.Primary.Attributes

		if rs.Primary.ID != vpcRs.Primary.ID {
			return fmt.Errorf("ID of this resource should be the vpc id")
		}

		return nil
	}
}

const testAccDataSourceAwsSubnetIDsConfig = `
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "test" {
  cidr_block = "172.16.0.0/16"

  tags {
    Name = "terraform-testacc-subnet-ids-data-source"
  }
}

resource "aws_subnet" "test" {
  vpc_id            = "${aws_vpc.test.id}"
  cidr_block        = "172.16.123.0/24"
  availability_zone = "us-west-2a"

  tags {
    Name = "terraform-testacc-subnet-ids-data-source"
  }
}

data "aws_subnet_ids" "selected" {
  vpc_id = "${aws_vpc.test.id}"
  depends_on = ["aws_subnet.test"]
}
`
