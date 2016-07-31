package aws

import (
	"fmt"
	"log"
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

		IDRefreshName:   "aws_vpc_peering_connection.foo",
		IDRefreshIgnore: []string{"auto_accept"},

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

func TestAccAWSVPCPeeringConnection_plan(t *testing.T) {
	var connection ec2.VpcPeeringConnection

	// reach out and DELETE the VPC Peering connection outside of Terraform
	testDestroy := func(*terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		log.Printf("[DEBUG] Test deleting VPC Peering connection")
		_, err := conn.DeleteVpcPeeringConnection(
			&ec2.DeleteVpcPeeringConnectionInput{
				VpcPeeringConnectionId: connection.VpcPeeringConnectionId,
			})
		if err != nil {
			return err
		}
		return nil
	}

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
					testDestroy,
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSVPCPeeringConnection_tags(t *testing.T) {
	var connection ec2.VpcPeeringConnection
	peerId := os.Getenv("TF_PEER_ID")
	if peerId == "" {
		t.Skip("Error: TestAccAWSVPCPeeringConnection_tags requires a peer id to be set")
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },

		IDRefreshName:   "aws_vpc_peering_connection.foo",
		IDRefreshIgnore: []string{"auto_accept"},

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccVpcPeeringConfigTags, peerId),
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

		if err != nil {
			return err
		}

		var pc *ec2.VpcPeeringConnection
		for _, c := range describe.VpcPeeringConnections {
			if rs.Primary.ID == *c.VpcPeeringConnectionId {
				pc = c
			}
		}

		if pc == nil {
			// not found
			return nil
		}

		if pc.Status != nil {
			if *pc.Status.Code == "deleted" {
				return nil
			}
			return fmt.Errorf("Found vpc peering connection in unexpected state: %s", pc)
		}

		// return error here; we've found the vpc_peering object we want, however
		// it's not in an expected state
		return fmt.Errorf("Fall through error for testAccCheckAWSVpcPeeringConnectionDestroy")
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
		tags {
			Name = "TestAccAWSVPCPeeringConnection_basic"
		}
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
		tags {
			Name = "TestAccAWSVPCPeeringConnection_tags"
		}
}

resource "aws_vpc" "bar" {
		cidr_block = "10.1.0.0/16"
}

resource "aws_vpc_peering_connection" "foo" {
		vpc_id = "${aws_vpc.foo.id}"
		peer_vpc_id = "${aws_vpc.bar.id}"
		peer_owner_id = "%s"
		tags {
			foo = "bar"
		}
}
`
