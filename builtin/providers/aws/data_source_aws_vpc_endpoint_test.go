// make testacc TEST=./builtin/providers/aws/ TESTARGS='-run=TestAccDataSourceAwsVpcEndpoint_'
package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsVpcEndpoint_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsVpcEndpointConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsVpcEndpointCheckExists("data.aws_vpc_endpoint.s3"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccDataSourceAwsVpcEndpoint_withRouteTable(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsVpcEndpointWithRouteTableConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsVpcEndpointCheckExists("data.aws_vpc_endpoint.s3"),
					resource.TestCheckResourceAttr(
						"data.aws_vpc_endpoint.s3", "route_table_ids.#", "1"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccDataSourceAwsVpcEndpointCheckExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		vpceRs, ok := s.RootModule().Resources["aws_vpc_endpoint.s3"]
		if !ok {
			return fmt.Errorf("can't find aws_vpc_endpoint.s3 in state")
		}

		attr := rs.Primary.Attributes

		if attr["id"] != vpceRs.Primary.Attributes["id"] {
			return fmt.Errorf(
				"id is %s; want %s",
				attr["id"],
				vpceRs.Primary.Attributes["id"],
			)
		}

		return nil
	}
}

const testAccDataSourceAwsVpcEndpointConfig = `
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags {
	  Name = "terraform-testacc-vpc-endpoint-data-source-foo"
  }
}

resource "aws_vpc_endpoint" "s3" {
    vpc_id = "${aws_vpc.foo.id}"
    service_name = "com.amazonaws.us-west-2.s3"
}

data "aws_vpc_endpoint" "s3" {
  vpc_id = "${aws_vpc.foo.id}"
  service_name = "com.amazonaws.us-west-2.s3"
  state = "available"

  depends_on = ["aws_vpc_endpoint.s3"]
}
`

const testAccDataSourceAwsVpcEndpointWithRouteTableConfig = `
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags {
	  Name = "terraform-testacc-vpc-endpoint-data-source-foo"
  }
}

resource "aws_route_table" "rt" {
    vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_vpc_endpoint" "s3" {
    vpc_id = "${aws_vpc.foo.id}"
    service_name = "com.amazonaws.us-west-2.s3"
	route_table_ids = ["${aws_route_table.rt.id}"]
}

data "aws_vpc_endpoint" "s3" {
  vpc_id = "${aws_vpc.foo.id}"
  service_name = "com.amazonaws.us-west-2.s3"
  state = "available"

  depends_on = ["aws_vpc_endpoint.s3"]
}
`
