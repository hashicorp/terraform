// make testacc TEST=./builtin/providers/aws/ TESTARGS='-run=TestAccAwsVPCPeeringConnectionAccepter_'
package aws

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAwsVPCPeeringConnectionAccepter_sameAccount(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAwsVPCPeeringConnectionAccepterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config:      testAccAwsVPCPeeringConnectionAccepterSameAccountConfig,
				ExpectError: regexp.MustCompile(`aws_vpc_peering_connection_accepter can only adopt into management cross-account VPC peering connections`),
			},
		},
	})
}

func testAccAwsVPCPeeringConnectionAccepterDestroy(s *terraform.State) error {
	// We don't destroy the underlying VPC Peering Connection.
	return nil
}

const testAccAwsVPCPeeringConnectionAccepterSameAccountConfig = `
provider "aws" {
    region = "us-west-2"
    // Requester's credentials.
}

provider "aws" {
    alias = "peer"
    region = "us-west-2"
    // Accepter's credentials.
}

resource "aws_vpc" "main" {
    cidr_block = "10.0.0.0/16"
}

resource "aws_vpc" "peer" {
    provider = "aws.peer"
    cidr_block = "10.1.0.0/16"
}

data "aws_caller_identity" "peer" {
    provider = "aws.peer"
}

// Requester's side of the connection.
resource "aws_vpc_peering_connection" "peer" {
    vpc_id = "${aws_vpc.main.id}"
    peer_vpc_id = "${aws_vpc.peer.id}"
    peer_owner_id = "${data.aws_caller_identity.peer.account_id}"
    auto_accept = false

    tags {
      Side = "Requester"
    }
}

// Accepter's side of the connection.
resource "aws_vpc_peering_connection_accepter" "peer" {
    provider = "aws.peer"
    vpc_peering_connection_id = "${aws_vpc_peering_connection.peer.id}"
    auto_accept = true

    tags {
       Side = "Accepter"
    }
}
`
