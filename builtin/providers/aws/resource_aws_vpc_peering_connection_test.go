package aws

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSVPCPeeringConnection_basic(t *testing.T) {
	var connection ec2.VpcPeeringConnection

	resource.Test(t, resource.TestCase{
		PreCheck:        func() { testAccPreCheck(t) },
		IDRefreshName:   "aws_vpc_peering_connection.foo",
		IDRefreshIgnore: []string{"auto_accept"},

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSVpcPeeringConnectionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcPeeringConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSVpcPeeringConnectionExists(
						"aws_vpc_peering_connection.foo",
						&connection),
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
		log.Printf("[DEBUG] Test deleting the VPC Peering Connection.")
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
		PreCheck:        func() { testAccPreCheck(t) },
		IDRefreshIgnore: []string{"auto_accept"},
		Providers:       testAccProviders,
		CheckDestroy:    testAccCheckAWSVpcPeeringConnectionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcPeeringConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSVpcPeeringConnectionExists(
						"aws_vpc_peering_connection.foo",
						&connection),
					testDestroy,
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSVPCPeeringConnection_tags(t *testing.T) {
	var connection ec2.VpcPeeringConnection

	resource.Test(t, resource.TestCase{
		PreCheck:        func() { testAccPreCheck(t) },
		IDRefreshName:   "aws_vpc_peering_connection.foo",
		IDRefreshIgnore: []string{"auto_accept"},

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcPeeringConfigTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSVpcPeeringConnectionExists(
						"aws_vpc_peering_connection.foo",
						&connection),
					testAccCheckTags(&connection.Tags, "foo", "bar"),
				),
			},
		},
	})
}

func TestAccAWSVPCPeeringConnection_options(t *testing.T) {
	var connection ec2.VpcPeeringConnection

	testAccepterChange := func(*terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		log.Printf("[DEBUG] Test change to the VPC Peering Connection Options.")

		_, err := conn.ModifyVpcPeeringConnectionOptions(
			&ec2.ModifyVpcPeeringConnectionOptionsInput{
				VpcPeeringConnectionId: connection.VpcPeeringConnectionId,
				AccepterPeeringConnectionOptions: &ec2.PeeringConnectionOptionsRequest{
					AllowDnsResolutionFromRemoteVpc: aws.Bool(false),
				},
			})
		if err != nil {
			return err
		}
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:        func() { testAccPreCheck(t) },
		IDRefreshName:   "aws_vpc_peering_connection.foo",
		IDRefreshIgnore: []string{"auto_accept"},

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSVpcPeeringConnectionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcPeeringConfigOptions,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSVpcPeeringConnectionExists(
						"aws_vpc_peering_connection.foo",
						&connection),
					resource.TestCheckResourceAttr(
						"aws_vpc_peering_connection.foo",
						"accepter.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_vpc_peering_connection.foo",
						"accepter.1102046665.allow_remote_vpc_dns_resolution", "true"),
					testAccCheckAWSVpcPeeringConnectionOptions(
						"aws_vpc_peering_connection.foo", "accepter",
						&ec2.VpcPeeringConnectionOptionsDescription{
							AllowDnsResolutionFromRemoteVpc:            aws.Bool(true),
							AllowEgressFromLocalClassicLinkToRemoteVpc: aws.Bool(false),
							AllowEgressFromLocalVpcToRemoteClassicLink: aws.Bool(false),
						}),
					resource.TestCheckResourceAttr(
						"aws_vpc_peering_connection.foo",
						"requester.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_vpc_peering_connection.foo",
						"requester.41753983.allow_classic_link_to_remote_vpc", "true"),
					resource.TestCheckResourceAttr(
						"aws_vpc_peering_connection.foo",
						"requester.41753983.allow_vpc_to_remote_classic_link", "true"),
					testAccCheckAWSVpcPeeringConnectionOptions(
						"aws_vpc_peering_connection.foo", "requester",
						&ec2.VpcPeeringConnectionOptionsDescription{
							AllowDnsResolutionFromRemoteVpc:            aws.Bool(false),
							AllowEgressFromLocalClassicLinkToRemoteVpc: aws.Bool(true),
							AllowEgressFromLocalVpcToRemoteClassicLink: aws.Bool(true),
						},
					),
					testAccepterChange,
				),
				ExpectNonEmptyPlan: true,
			},
			resource.TestStep{
				Config: testAccVpcPeeringConfigOptions,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSVpcPeeringConnectionExists(
						"aws_vpc_peering_connection.foo",
						&connection),
					resource.TestCheckResourceAttr(
						"aws_vpc_peering_connection.foo",
						"accepter.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_vpc_peering_connection.foo",
						"accepter.1102046665.allow_remote_vpc_dns_resolution", "true"),
					testAccCheckAWSVpcPeeringConnectionOptions(
						"aws_vpc_peering_connection.foo", "accepter",
						&ec2.VpcPeeringConnectionOptionsDescription{
							AllowDnsResolutionFromRemoteVpc:            aws.Bool(true),
							AllowEgressFromLocalClassicLinkToRemoteVpc: aws.Bool(false),
							AllowEgressFromLocalVpcToRemoteClassicLink: aws.Bool(false),
						},
					),
				),
			},
		},
	})
}

func TestAccAWSVPCPeeringConnection_failedState(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:        func() { testAccPreCheck(t) },
		IDRefreshIgnore: []string{"auto_accept"},
		Providers:       testAccProviders,
		CheckDestroy:    testAccCheckAWSVpcPeeringConnectionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config:      testAccVpcPeeringConfigFailedState,
				ExpectError: regexp.MustCompile(`.*Error waiting.*\(pcx-\w+\).*incorrect.*VPC-ID.*`),
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
			return fmt.Errorf("Found the VPC Peering Connection in an unexpected state: %s", pc)
		}

		// return error here; we've found the vpc_peering object we want, however
		// it's not in an expected state
		return fmt.Errorf("Fall through error for testAccCheckAWSVpcPeeringConnectionDestroy.")
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
			return fmt.Errorf("No VPC Peering Connection ID is set.")
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
			return fmt.Errorf("VPC Peering Connection could not be found")
		}

		*connection = *resp.VpcPeeringConnections[0]

		return nil
	}
}

func testAccCheckAWSVpcPeeringConnectionOptions(n, block string, options *ec2.VpcPeeringConnectionOptionsDescription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VPC Peering Connection ID is set.")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := conn.DescribeVpcPeeringConnections(
			&ec2.DescribeVpcPeeringConnectionsInput{
				VpcPeeringConnectionIds: []*string{aws.String(rs.Primary.ID)},
			})
		if err != nil {
			return err
		}

		pc := resp.VpcPeeringConnections[0]

		o := pc.AccepterVpcInfo
		if block == "requester" {
			o = pc.RequesterVpcInfo
		}

		if !reflect.DeepEqual(o.PeeringOptions, options) {
			return fmt.Errorf("Expected the VPC Peering Connection Options to be %#v, got %#v",
				options, o.PeeringOptions)
		}

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
	auto_accept = true
	tags {
		foo = "bar"
	}
}
`

const testAccVpcPeeringConfigOptions = `
resource "aws_vpc" "foo" {
	cidr_block = "10.0.0.0/16"
	tags {
		Name = "TestAccAWSVPCPeeringConnection_options"
	}
}

resource "aws_vpc" "bar" {
	cidr_block = "10.1.0.0/16"
	enable_dns_hostnames = true
}

resource "aws_vpc_peering_connection" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
	peer_vpc_id = "${aws_vpc.bar.id}"
	auto_accept = true

	accepter {
		allow_remote_vpc_dns_resolution = true
	}

	requester {
		allow_vpc_to_remote_classic_link = true
		allow_classic_link_to_remote_vpc = true
	}
}
`

const testAccVpcPeeringConfigFailedState = `
resource "aws_vpc" "foo" {
	cidr_block = "10.0.0.0/16"
	tags {
		Name = "TestAccAWSVPCPeeringConnection_failedState"
	}
}

resource "aws_vpc" "bar" {
	cidr_block = "10.0.0.0/16"
}

resource "aws_vpc_peering_connection" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
	peer_vpc_id = "${aws_vpc.bar.id}"
}
`
