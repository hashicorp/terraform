package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceAwsSubnetIDs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsSubnetIDsConfig,
			},
			{
				Config: testAccDataSourceAwsSubnetIDsConfigWithDataSource,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_subnet_ids.selected", "ids.#", "1"),
				),
			},
		},
	})
}

const testAccDataSourceAwsSubnetIDsConfigWithDataSource = `
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
}
`
const testAccDataSourceAwsSubnetIDsConfig = `
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
`
