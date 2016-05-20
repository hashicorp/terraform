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

func TestAccAWSCustomerGateway_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_customer_gateway.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckCustomerGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCustomerGatewayConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomerGateway(
						"aws_customer_gateway.foo",
					),
				),
			},
			resource.TestStep{
				Config: testAccCustomerGatewayConfigUpdateTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomerGateway(
						"aws_customer_gateway.foo",
					),
				),
			},
			resource.TestStep{
				Config: testAccCustomerGatewayConfigForceReplace,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomerGateway(
						"aws_customer_gateway.foo",
					),
				),
			},
		},
	})
}

func testAccCheckCustomerGatewayDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_customer_gatewah" {
			continue
		}

		gatewayFilter := &ec2.Filter{
			Name:   aws.String("customer-gateway-id"),
			Values: []*string{aws.String(rs.Primary.ID)},
		}

		resp, err := conn.DescribeCustomerGateways(&ec2.DescribeCustomerGatewaysInput{
			Filters: []*ec2.Filter{gatewayFilter},
		})

		if ae, ok := err.(awserr.Error); ok && ae.Code() == "InvalidCustomerGatewayID.NotFound" {
			continue
		}

		if err == nil {
			if len(resp.CustomerGateways) > 0 {
				return fmt.Errorf("Customer gateway still exists: %v", resp.CustomerGateways)
			}
		}

		return err
	}

	return nil
}

func testAccCheckCustomerGateway(gatewayResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[gatewayResource]
		if !ok {
			return fmt.Errorf("Not found: %s", gatewayResource)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		gateway, ok := s.RootModule().Resources[gatewayResource]
		if !ok {
			return fmt.Errorf("Not found: %s", gatewayResource)
		}

		ec2conn := testAccProvider.Meta().(*AWSClient).ec2conn
		gatewayFilter := &ec2.Filter{
			Name:   aws.String("customer-gateway-id"),
			Values: []*string{aws.String(gateway.Primary.ID)},
		}

		_, err := ec2conn.DescribeCustomerGateways(&ec2.DescribeCustomerGatewaysInput{
			Filters: []*ec2.Filter{gatewayFilter},
		})

		if err != nil {
			return err
		}

		return nil
	}
}

const testAccCustomerGatewayConfig = `
resource "aws_customer_gateway" "foo" {
	bgp_asn = 60000
	ip_address = "172.0.0.1"
	type = "ipsec.1"
	tags {
		Name = "foo-gateway"
	}
}
`

// Add the Another: "tag" tag.
const testAccCustomerGatewayConfigUpdateTags = `
resource "aws_customer_gateway" "foo" {
	bgp_asn = 60000
	ip_address = "172.0.0.1"
	type = "ipsec.1"
	tags {
		Name = "foo-gateway"
		Another = "tag"
	}
}
`

// Change the ip_address.
const testAccCustomerGatewayConfigForceReplace = `
resource "aws_customer_gateway" "foo" {
	bgp_asn = 60000
	ip_address = "172.10.10.1"
	type = "ipsec.1"
	tags {
		Name = "foo-gateway"
		Another = "tag"
	}
}
`
