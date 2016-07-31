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

func TestAccAWSInternetGateway_basic(t *testing.T) {
	var v, v2 ec2.InternetGateway

	testNotEqual := func(*terraform.State) error {
		if len(v.Attachments) == 0 {
			return fmt.Errorf("IG A is not attached")
		}
		if len(v2.Attachments) == 0 {
			return fmt.Errorf("IG B is not attached")
		}

		id1 := v.Attachments[0].VpcId
		id2 := v2.Attachments[0].VpcId
		if id1 == id2 {
			return fmt.Errorf("Both attachment IDs are the same")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_internet_gateway.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckInternetGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInternetGatewayConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInternetGatewayExists(
						"aws_internet_gateway.foo", &v),
				),
			},

			resource.TestStep{
				Config: testAccInternetGatewayConfigChangeVPC,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInternetGatewayExists(
						"aws_internet_gateway.foo", &v2),
					testNotEqual,
				),
			},
		},
	})
}

func TestAccAWSInternetGateway_delete(t *testing.T) {
	var ig ec2.InternetGateway

	testDeleted := func(r string) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			_, ok := s.RootModule().Resources[r]
			if ok {
				return fmt.Errorf("Internet Gateway %q should have been deleted", r)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_internet_gateway.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckInternetGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInternetGatewayConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInternetGatewayExists("aws_internet_gateway.foo", &ig)),
			},
			resource.TestStep{
				Config: testAccNoInternetGatewayConfig,
				Check:  resource.ComposeTestCheckFunc(testDeleted("aws_internet_gateway.foo")),
			},
		},
	})
}

func TestAccAWSInternetGateway_tags(t *testing.T) {
	var v ec2.InternetGateway

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_internet_gateway.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckInternetGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckInternetGatewayConfigTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInternetGatewayExists("aws_internet_gateway.foo", &v),
					testAccCheckTags(&v.Tags, "foo", "bar"),
				),
			},

			resource.TestStep{
				Config: testAccCheckInternetGatewayConfigTagsUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInternetGatewayExists("aws_internet_gateway.foo", &v),
					testAccCheckTags(&v.Tags, "foo", ""),
					testAccCheckTags(&v.Tags, "bar", "baz"),
				),
			},
		},
	})
}

func testAccCheckInternetGatewayDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_internet_gateway" {
			continue
		}

		// Try to find the resource
		resp, err := conn.DescribeInternetGateways(&ec2.DescribeInternetGatewaysInput{
			InternetGatewayIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err == nil {
			if len(resp.InternetGateways) > 0 {
				return fmt.Errorf("still exists")
			}

			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidInternetGatewayID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckInternetGatewayExists(n string, ig *ec2.InternetGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := conn.DescribeInternetGateways(&ec2.DescribeInternetGatewaysInput{
			InternetGatewayIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err != nil {
			return err
		}
		if len(resp.InternetGateways) == 0 {
			return fmt.Errorf("InternetGateway not found")
		}

		*ig = *resp.InternetGateways[0]

		return nil
	}
}

const testAccNoInternetGatewayConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}
`

const testAccInternetGatewayConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_internet_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}
`

const testAccInternetGatewayConfigChangeVPC = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_vpc" "bar" {
	cidr_block = "10.2.0.0/16"
}

resource "aws_internet_gateway" "foo" {
	vpc_id = "${aws_vpc.bar.id}"
}
`

const testAccCheckInternetGatewayConfigTags = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_internet_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		foo = "bar"
	}
}
`

const testAccCheckInternetGatewayConfigTagsUpdate = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_internet_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		bar = "baz"
	}
}
`
