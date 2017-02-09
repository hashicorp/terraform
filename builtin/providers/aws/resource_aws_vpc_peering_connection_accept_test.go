package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"os"
	"testing"
)

func TestAccAWSVPCPeeringConnectionAccept_basic(t *testing.T) {
	var connection ec2.VpcPeeringConnection

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if os.Getenv("AWS_ACCOUNT_ID") == "" {
				t.Fatal("AWS_ACCOUNT_ID must be set")
			}
		},

		IDRefreshName:   "aws_vpc_peering_connection.foo",
		IDRefreshIgnore: []string{"auto_accept"},

		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcPeeringAcceptConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSVpcPeeringConnectionExists("aws_vpc_peering_connection.foo", &connection),
					testAccCheckAWSVpcPeeringConnectionAccepted(&connection),
				),
			},
		},
	})
}

func testAccCheckAWSVpcPeeringConnectionAccepted(conn *ec2.VpcPeeringConnection) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if conn.Status == nil {
			return fmt.Errorf("No vpc peering connection status")
		}
		if *conn.Status.Code != ec2.VpcPeeringConnectionStateReasonCodeActive {
			return fmt.Errorf("Vpc peering connection not accepted: %s", conn.Status.Code)
		}
		return nil
	}
}

const testAccVpcPeeringAcceptConfig = `
resource "aws_vpc" "foo" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_vpc" "bar" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_vpc_peering_connection" "foo" {
  vpc_id = "${aws_vpc.foo.id}"
  peer_vpc_id = "${aws_vpc.bar.id}"
  auto_accept = false
}

resource "aws_vpc_peering_connection_accept" "foo" {
  peering_connection_id = "${aws_vpc_peering_connection.foo.id}"
}
`
