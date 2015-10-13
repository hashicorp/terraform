package aws

import (
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSVPCPeeringConnection_basic(t *testing.T) {
	var connection ec2.VpcPeeringConnection

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if os.Getenv("AWS_ACCOUNT_ID") == "" {
				t.Fatal("AWS_ACCOUNT_ID must be set")
			}
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSVpcPeeringConnectionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcPeeringConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSVpcPeeringConnectionExists("aws_vpc_peering_connection.foo", &connection),
				),
			},
		},
	})
}

func TestAccAWSVPCPeeringConnection_tags(t *testing.T) {
	var connection ec2.VpcPeeringConnection

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcPeeringConfigTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSVpcPeeringConnectionExists("aws_vpc_peering_connection.foo", &connection),
					testAccCheckTags(&connection.Tags, "foo", "bar"),
				),
			},
		},
	})
}

func testAccCheckAWSVpcPeeringConnectionDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_vpc_peering_connection" {
			continue
		}

		describe, err := conn.DescribeVpcPeeringConnections(
			&ec2.DescribeVpcPeeringConnectionsInput{
				VpcPeeringConnectionIds: []*string{aws.String(rs.Primary.ID)},
			})

		if err == nil {
			if len(describe.VpcPeeringConnections) != 0 {
				return fmt.Errorf("vpc peering connection still exists")
			}
		}
	}

	return nil
}

func testAccCheckAWSVpcPeeringConnectionExists(n string, connection *ec2.VpcPeeringConnection) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No vpc peering connection id is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := conn.DescribeVpcPeeringConnections(
			&ec2.DescribeVpcPeeringConnectionsInput{
				VpcPeeringConnectionIds: []*string{aws.String(rs.Primary.ID)},
			})
		if err != nil {
			return err
		}
		if len(resp.VpcPeeringConnections) == 0 {
			return fmt.Errorf("VPC peering connection not found")
		}

		*connection = *resp.VpcPeeringConnections[0]

		return nil
	}
}

const testAccVpcPeeringConfig = `
resource "aws_vpc" "foo" {
		cidr_block = "10.0.0.0/16"
}

resource "aws_vpc" "bar" {
		cidr_block = "10.1.0.0/16"
}

resource "aws_vpc_peering_connection" "foo" {
		vpc_id = "${aws_vpc.foo.id}"
		peer_vpc_id = "${aws_vpc.bar.id}"
		auto_accept = true
}
`

const testAccVpcPeeringConfigTags = `
resource "aws_vpc" "foo" {
		cidr_block = "10.0.0.0/16"
}

resource "aws_vpc" "bar" {
		cidr_block = "10.1.0.0/16"
}

resource "aws_vpc_peering_connection" "foo" {
		vpc_id = "${aws_vpc.foo.id}"
		peer_vpc_id = "${aws_vpc.bar.id}"
		tags {
			foo = "bar"
		}
}
`
