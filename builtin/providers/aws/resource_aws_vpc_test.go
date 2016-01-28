package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSVpc_basic(t *testing.T) {
	var vpc ec2.Vpc

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("aws_vpc.foo", &vpc),
					testAccCheckVpcCidr(&vpc, "10.1.0.0/16"),
					resource.TestCheckResourceAttr(
						"aws_vpc.foo", "cidr_block", "10.1.0.0/16"),
				),
			},
		},
	})
}

func TestAccAWSVpc_dedicatedTenancy(t *testing.T) {
	var vpc ec2.Vpc

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcDedicatedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("aws_vpc.bar", &vpc),
					resource.TestCheckResourceAttr(
						"aws_vpc.bar", "instance_tenancy", "dedicated"),
				),
			},
		},
	})
}

func TestAccAWSVpc_tags(t *testing.T) {
	var vpc ec2.Vpc

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcConfigTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("aws_vpc.foo", &vpc),
					testAccCheckVpcCidr(&vpc, "10.1.0.0/16"),
					resource.TestCheckResourceAttr(
						"aws_vpc.foo", "cidr_block", "10.1.0.0/16"),
					testAccCheckTags(&vpc.Tags, "foo", "bar"),
				),
			},

			resource.TestStep{
				Config: testAccVpcConfigTagsUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("aws_vpc.foo", &vpc),
					testAccCheckTags(&vpc.Tags, "foo", ""),
					testAccCheckTags(&vpc.Tags, "bar", "baz"),
				),
			},
		},
	})
}

func TestAccAWSVpc_update(t *testing.T) {
	var vpc ec2.Vpc

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("aws_vpc.foo", &vpc),
					testAccCheckVpcCidr(&vpc, "10.1.0.0/16"),
					resource.TestCheckResourceAttr(
						"aws_vpc.foo", "cidr_block", "10.1.0.0/16"),
				),
			},
			resource.TestStep{
				Config: testAccVpcConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("aws_vpc.foo", &vpc),
					resource.TestCheckResourceAttr(
						"aws_vpc.foo", "enable_dns_hostnames", "true"),
				),
			},
		},
	})
}

func testAccCheckVpcDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_vpc" {
			continue
		}

		// Try to find the VPC
		DescribeVpcOpts := &ec2.DescribeVpcsInput{
			VpcIds: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeVpcs(DescribeVpcOpts)
		if err == nil {
			if len(resp.Vpcs) > 0 {
				return fmt.Errorf("VPCs still exist.")
			}

			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidVpcID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckVpcCidr(vpc *ec2.Vpc, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		CIDRBlock := vpc.CidrBlock
		if *CIDRBlock != expected {
			return fmt.Errorf("Bad cidr: %s", *vpc.CidrBlock)
		}

		return nil
	}
}

func testAccCheckVpcExists(n string, vpc *ec2.Vpc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VPC ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		DescribeVpcOpts := &ec2.DescribeVpcsInput{
			VpcIds: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeVpcs(DescribeVpcOpts)
		if err != nil {
			return err
		}
		if len(resp.Vpcs) == 0 {
			return fmt.Errorf("VPC not found")
		}

		*vpc = *resp.Vpcs[0]

		return nil
	}
}

// https://github.com/hashicorp/terraform/issues/1301
func TestAccAWSVpc_bothDnsOptionsSet(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcConfig_BothDnsOptions,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_vpc.bar", "enable_dns_hostnames", "true"),
					resource.TestCheckResourceAttr(
						"aws_vpc.bar", "enable_dns_support", "true"),
				),
			},
		},
	})
}

func TestAccAWSVpc_classiclinkOptionSet(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcConfig_ClassiclinkOption,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_vpc.bar", "enable_classiclink", "true"),
				),
			},
		},
	})
}

const testAccVpcConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}
`

const testAccVpcConfigUpdate = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	enable_dns_hostnames = true
}
`

const testAccVpcConfigTags = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"

	tags {
		foo = "bar"
	}
}
`

const testAccVpcConfigTagsUpdate = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"

	tags {
		bar = "baz"
	}
}
`
const testAccVpcDedicatedConfig = `
resource "aws_vpc" "bar" {
	instance_tenancy = "dedicated"

	cidr_block = "10.2.0.0/16"
}
`

const testAccVpcConfig_BothDnsOptions = `
provider "aws" {
	region = "eu-central-1"
}

resource "aws_vpc" "bar" {
	cidr_block = "10.2.0.0/16"

	enable_dns_hostnames = true
	enable_dns_support = true
}
`

const testAccVpcConfig_ClassiclinkOption = `
resource "aws_vpc" "bar" {
	cidr_block = "172.2.0.0/16"

	enable_classiclink = true
}
`
