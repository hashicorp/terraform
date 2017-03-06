// make testacc TEST=./builtin/providers/aws/ TESTARGS='-run=TestAccDataSourceAwsVpcPeeringConnection_'
package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsVpcPeeringConnection_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsVpcPeeringConnectionConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsVpcPeeringConnectionCheck("data.aws_vpc_peering_connection.test_by_id"),
					testAccDataSourceAwsVpcPeeringConnectionCheck("data.aws_vpc_peering_connection.test_by_requester_vpc_id"),
					testAccDataSourceAwsVpcPeeringConnectionCheck("data.aws_vpc_peering_connection.test_by_accepter_vpc_id"),
					testAccDataSourceAwsVpcPeeringConnectionCheck("data.aws_vpc_peering_connection.test_by_requester_cidr_block"),
					testAccDataSourceAwsVpcPeeringConnectionCheck("data.aws_vpc_peering_connection.test_by_accepter_cidr_block"),
					testAccDataSourceAwsVpcPeeringConnectionCheck("data.aws_vpc_peering_connection.test_by_owner_ids"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccDataSourceAwsVpcPeeringConnectionCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		pcxRs, ok := s.RootModule().Resources["aws_vpc_peering_connection.test"]
		if !ok {
			return fmt.Errorf("can't find aws_vpc_peering_connection.test in state")
		}

		attr := rs.Primary.Attributes

		if attr["id"] != pcxRs.Primary.Attributes["id"] {
			return fmt.Errorf(
				"id is %s; want %s",
				attr["id"],
				pcxRs.Primary.Attributes["id"],
			)
		}

		return nil
	}
}

const testAccDataSourceAwsVpcPeeringConnectionConfig = `
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags {
	  Name = "terraform-testacc-vpc-peering-connection-data-source-foo"
  }
}

resource "aws_vpc" "bar" {
  cidr_block = "10.2.0.0/16"

  tags {
	  Name = "terraform-testacc-vpc-peering-connection-data-source-bar"
  }
}

resource "aws_vpc_peering_connection" "test" {
	vpc_id = "${aws_vpc.foo.id}"
	peer_vpc_id = "${aws_vpc.bar.id}"
	auto_accept = true

    tags {
      Name = "terraform-testacc-vpc-peering-connection-data-source-foo-to-bar"
    }
}

data "aws_caller_identity" "current" {}

data "aws_vpc_peering_connection" "test_by_id" {
	id = "${aws_vpc_peering_connection.test.id}"
}

data "aws_vpc_peering_connection" "test_by_requester_vpc_id" {
	vpc_id = "${aws_vpc.foo.id}"

	depends_on = ["aws_vpc_peering_connection.test"]
}

data "aws_vpc_peering_connection" "test_by_accepter_vpc_id" {
	peer_vpc_id = "${aws_vpc.bar.id}"

	depends_on = ["aws_vpc_peering_connection.test"]
}

data "aws_vpc_peering_connection" "test_by_requester_cidr_block" {
	cidr_block = "10.1.0.0/16"
	status = "active"

	depends_on = ["aws_vpc_peering_connection.test"]
}

data "aws_vpc_peering_connection" "test_by_accepter_cidr_block" {
	peer_cidr_block = "10.2.0.0/16"
	status = "active"

	depends_on = ["aws_vpc_peering_connection.test"]
}

data "aws_vpc_peering_connection" "test_by_owner_ids" {
	owner_id = "${data.aws_caller_identity.current.account_id}"
	peer_owner_id = "${data.aws_caller_identity.current.account_id}"
	status = "active"

	depends_on = ["aws_vpc_peering_connection.test"]
}
`
