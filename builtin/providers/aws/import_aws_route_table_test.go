package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSRouteTable_importBasic(t *testing.T) {
	checkFn := func(s []*terraform.InstanceState) error {
		// Expect 2: group, 1 rules
		if len(s) != 2 {
			return fmt.Errorf("bad states: %#v", s)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRouteTableDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRouteTableConfig,
			},

			{
				ResourceName:     "aws_route_table.foo",
				ImportState:      true,
				ImportStateCheck: checkFn,
			},
		},
	})
}

func TestAccAWSRouteTable_complex(t *testing.T) {
	checkFn := func(s []*terraform.InstanceState) error {
		// Expect 3: group, 2 rules
		if len(s) != 3 {
			return fmt.Errorf("bad states: %#v", s)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRouteTableDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRouteTableConfig_complexImport,
			},

			{
				ResourceName:     "aws_route_table.mod",
				ImportState:      true,
				ImportStateCheck: checkFn,
			},
		},
	})
}

const testAccRouteTableConfig_complexImport = `
resource "aws_vpc" "default" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true

  tags {
    Name = "tf-rt-import-test"
  }
}

resource "aws_subnet" "tf_test_subnet" {
  vpc_id                  = "${aws_vpc.default.id}"
  cidr_block              = "10.0.0.0/24"
  map_public_ip_on_launch = true

  tags {
    Name = "tf-rt-import-test"
  }
}

resource "aws_eip" "nat" {
  vpc                       = true
  associate_with_private_ip = "10.0.0.10"
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.default.id}"

  tags {
    Name = "tf-rt-import-test"
  }
}

variable "private_subnet_cidrs" {
  default = "10.0.0.0/24"
}

resource "aws_nat_gateway" "nat" {
  count         = "${length(split(",", var.private_subnet_cidrs))}"
  allocation_id = "${element(aws_eip.nat.*.id, count.index)}"
  subnet_id     = "${aws_subnet.tf_test_subnet.id}"
}

resource "aws_route_table" "mod" {
  count  = "${length(split(",", var.private_subnet_cidrs))}"
  vpc_id = "${aws_vpc.default.id}"

  tags {
    Name = "tf-rt-import-test"
  }

  depends_on = ["aws_internet_gateway.ogw", "aws_internet_gateway.gw"]
}

resource "aws_route" "mod-1" {
  route_table_id         = "${aws_route_table.mod.id}"
  destination_cidr_block = "0.0.0.0/0"
  nat_gateway_id         = "${element(aws_nat_gateway.nat.*.id, count.index)}"
}

resource "aws_route" "mod" {
  route_table_id            = "${aws_route_table.mod.id}"
  destination_cidr_block    = "10.181.0.0/16"
  vpc_peering_connection_id = "${aws_vpc_peering_connection.foo.id}"
}

resource "aws_vpc_endpoint" "s3" {
  vpc_id          = "${aws_vpc.default.id}"
  service_name    = "com.amazonaws.us-west-2.s3"
  route_table_ids = ["${aws_route_table.mod.*.id}"]
}

### vpc bar

resource "aws_vpc" "bar" {
  cidr_block = "10.1.0.0/16"

  tags {
    Name = "tf-rt-import-test"
  }
}

resource "aws_internet_gateway" "ogw" {
  vpc_id = "${aws_vpc.bar.id}"

  tags {
    Name = "tf-rt-import-test"
  }
}

### vpc peer connection

resource "aws_vpc_peering_connection" "foo" {
  vpc_id        = "${aws_vpc.default.id}"
  peer_vpc_id   = "${aws_vpc.bar.id}"
  peer_owner_id = "187416307283"

  tags {
    Name = "tf-rt-import-test"
  }

  auto_accept = true
}
`
