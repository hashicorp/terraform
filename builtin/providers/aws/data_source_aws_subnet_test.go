package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsSubnet(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsSubnetConfig(rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
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

func TestAccDataSourceAwsSubnetIpv6ByIpv6Filter(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsSubnetConfigIpv6(rInt),
			},
			{
				Config: testAccDataSourceAwsSubnetConfigIpv6WithDataSourceFilter(rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"data.aws_subnet.by_ipv6_cidr", "ipv6_cidr_block_association_id"),
					resource.TestCheckResourceAttrSet(
						"data.aws_subnet.by_ipv6_cidr", "ipv6_cidr_block"),
				),
			},
		},
	})
}

func TestAccDataSourceAwsSubnetIpv6ByIpv6CidrBlock(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsSubnetConfigIpv6(rInt),
			},
			{
				Config: testAccDataSourceAwsSubnetConfigIpv6WithDataSourceIpv6CidrBlock(rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"data.aws_subnet.by_ipv6_cidr", "ipv6_cidr_block_association_id"),
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

		return nil
	}
}

func testAccDataSourceAwsSubnetConfig(rInt int) string {
	return fmt.Sprintf(`
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
    Name = "terraform-testacc-subnet-data-source-%d"
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
`, rInt)
}

func testAccDataSourceAwsSubnetConfigIpv6(rInt int) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "172.20.0.0/16"
  assign_generated_ipv6_cidr_block = true

  tags {
    Name = "terraform-testacc-subnet-data-source-ipv6"
  }
}

resource "aws_subnet" "test" {
  vpc_id            = "${aws_vpc.test.id}"
  cidr_block        = "172.20.123.0/24"
  availability_zone = "us-west-2a"
  ipv6_cidr_block = "${cidrsubnet(aws_vpc.test.ipv6_cidr_block, 8, 1)}"

  tags {
    Name = "terraform-testacc-subnet-data-sourceipv6-%d"
  }
}
`, rInt)
}

func testAccDataSourceAwsSubnetConfigIpv6WithDataSourceFilter(rInt int) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "172.20.0.0/16"
  assign_generated_ipv6_cidr_block = true

  tags {
    Name = "terraform-testacc-subnet-data-source-ipv6"
  }
}

resource "aws_subnet" "test" {
  vpc_id            = "${aws_vpc.test.id}"
  cidr_block        = "172.20.123.0/24"
  availability_zone = "us-west-2a"
  ipv6_cidr_block = "${cidrsubnet(aws_vpc.test.ipv6_cidr_block, 8, 1)}"

  tags {
    Name = "terraform-testacc-subnet-data-sourceipv6-%d"
  }
}

data "aws_subnet" "by_ipv6_cidr" {
  filter {
    name = "ipv6-cidr-block-association.ipv6-cidr-block"
    values = ["${aws_subnet.test.ipv6_cidr_block}"]
  }
}
`, rInt)
}

func testAccDataSourceAwsSubnetConfigIpv6WithDataSourceIpv6CidrBlock(rInt int) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "172.20.0.0/16"
  assign_generated_ipv6_cidr_block = true

  tags {
    Name = "terraform-testacc-subnet-data-source-ipv6"
  }
}

resource "aws_subnet" "test" {
  vpc_id            = "${aws_vpc.test.id}"
  cidr_block        = "172.20.123.0/24"
  availability_zone = "us-west-2a"
  ipv6_cidr_block = "${cidrsubnet(aws_vpc.test.ipv6_cidr_block, 8, 1)}"

  tags {
    Name = "terraform-testacc-subnet-data-sourceipv6-%d"
  }
}

data "aws_subnet" "by_ipv6_cidr" {
  ipv6_cidr_block = "${aws_subnet.test.ipv6_cidr_block}"
}
`, rInt)
}
