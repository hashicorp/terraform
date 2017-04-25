// make testacc TEST=./builtin/providers/aws/ TESTARGS='-run=TestAccDataSourceAwsRouteTable_'
package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsRouteTable_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsRouteTableGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsRouteTableCheck("data.aws_route_table.by_tag"),
					testAccDataSourceAwsRouteTableCheck("data.aws_route_table.by_filter"),
					testAccDataSourceAwsRouteTableCheck("data.aws_route_table.by_subnet"),
					testAccDataSourceAwsRouteTableCheck("data.aws_route_table.by_id"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccDataSourceAwsRouteTable_main(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsRouteTableMainRoute,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsRouteTableCheckMain("data.aws_route_table.by_filter"),
				),
			},
		},
	})
}

func testAccDataSourceAwsRouteTableCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]

		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		rts, ok := s.RootModule().Resources["aws_route_table.test"]
		if !ok {
			return fmt.Errorf("can't find aws_route_table.test in state")
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

		if attr["id"] != rts.Primary.Attributes["id"] {
			return fmt.Errorf(
				"id is %s; want %s",
				attr["id"],
				rts.Primary.Attributes["id"],
			)
		}

		if attr["route_table_id"] != rts.Primary.Attributes["id"] {
			return fmt.Errorf(
				"route_table_id is %s; want %s",
				attr["route_table_id"],
				rts.Primary.Attributes["id"],
			)
		}

		if attr["vpc_id"] != vpcRs.Primary.Attributes["id"] {
			return fmt.Errorf(
				"vpc_id is %s; want %s",
				attr["vpc_id"],
				vpcRs.Primary.Attributes["id"],
			)
		}

		if attr["tags.Name"] != "terraform-testacc-routetable-data-source" {
			return fmt.Errorf("bad Name tag %s", attr["tags.Name"])
		}
		if attr["associations.0.subnet_id"] != subnetRs.Primary.Attributes["id"] {
			return fmt.Errorf(
				"subnet_id is %v; want %s",
				attr["associations.0.subnet_id"],
				subnetRs.Primary.Attributes["id"],
			)
		}

		return nil
	}
}

func testAccDataSourceAwsRouteTableCheckMain(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]

		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		attr := rs.Primary.Attributes

		// Verify attributes are set
		if _, ok := attr["id"]; !ok {
			return fmt.Errorf("id not set for main route table")
		}
		if _, ok := attr["vpc_id"]; !ok {
			return fmt.Errorf("vpc_id not set for main route table")
		}
		// Verify it's actually the main route table that's returned
		if attr["associations.0.main"] != "true" {
			return fmt.Errorf("main route table not found")
		}

		return nil
	}
}

const testAccDataSourceAwsRouteTableGroupConfig = `
provider "aws" {
  region = "eu-central-1"
}
resource "aws_vpc" "test" {
  cidr_block = "172.16.0.0/16"

  tags {
    Name = "terraform-testacc-data-source"
  }
}

resource "aws_subnet" "test" {
  cidr_block = "172.16.0.0/24"
  vpc_id     = "${aws_vpc.test.id}"
  tags {
    Name = "terraform-testacc-data-source"
  }
}

resource "aws_route_table" "test" {
  vpc_id = "${aws_vpc.test.id}"
  tags {
    Name = "terraform-testacc-routetable-data-source"
  }
}

resource "aws_route_table_association" "a" {
    subnet_id = "${aws_subnet.test.id}"
    route_table_id = "${aws_route_table.test.id}"
}

data "aws_route_table" "by_filter" {
  filter {
    name = "association.route-table-association-id"
    values = ["${aws_route_table_association.a.id}"]
  }
  depends_on = ["aws_route_table_association.a"]
}

data "aws_route_table" "by_tag" {
  tags {
    Name = "${aws_route_table.test.tags["Name"]}"
  }
  depends_on = ["aws_route_table_association.a"]
}

data "aws_route_table" "by_subnet" {
  subnet_id = "${aws_subnet.test.id}"
  depends_on = ["aws_route_table_association.a"]
}

data "aws_route_table" "by_id" {
  route_table_id = "${aws_route_table.test.id}"
  depends_on = ["aws_route_table_association.a"]
}
`

// Uses us-east-2, as region only has a single main route table
const testAccDataSourceAwsRouteTableMainRoute = `
provider "aws" {
  region = "us-east-2"
}

resource "aws_vpc" "test" {
  cidr_block = "172.16.0.0/16"

  tags {
    Name = "terraform-testacc-data-source"
  }
}

data "aws_route_table" "by_filter" {
  filter {
    name = "association.main"
    values = ["true"]
  }
  filter {
    name = "vpc-id"
    values = ["${aws_vpc.test.id}"]
  }
}
`
