package aws

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCustomerGateway_basic(t *testing.T) {
	var gateway ec2.CustomerGateway
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_customer_gateway.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckCustomerGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCustomerGatewayConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomerGateway("aws_customer_gateway.foo", &gateway),
				),
			},
			resource.TestStep{
				Config: testAccCustomerGatewayConfigUpdateTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomerGateway("aws_customer_gateway.foo", &gateway),
				),
			},
			resource.TestStep{
				Config: testAccCustomerGatewayConfigForceReplace,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomerGateway("aws_customer_gateway.foo", &gateway),
				),
			},
		},
	})
}

func TestAccAWSCustomerGateway_disappears(t *testing.T) {
	var gateway ec2.CustomerGateway
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCustomerGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCustomerGatewayConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomerGateway("aws_customer_gateway.foo", &gateway),
					testAccAWSCustomerGatewayDisappears(&gateway),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccAWSCustomerGatewayDisappears(gateway *ec2.CustomerGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		opts := &ec2.DeleteCustomerGatewayInput{
			CustomerGatewayId: gateway.CustomerGatewayId,
		}
		if _, err := conn.DeleteCustomerGateway(opts); err != nil {
			return err
		}
		return resource.Retry(40*time.Minute, func() *resource.RetryError {
			opts := &ec2.DescribeCustomerGatewaysInput{
				CustomerGatewayIds: []*string{gateway.CustomerGatewayId},
			}
			resp, err := conn.DescribeCustomerGateways(opts)
			if err != nil {
				cgw, ok := err.(awserr.Error)
				if ok && cgw.Code() == "InvalidCustomerGatewayID.NotFound" {
					return nil
				}
				return resource.NonRetryableError(
					fmt.Errorf("Error retrieving Customer Gateway: %s", err))
			}
			if *resp.CustomerGateways[0].State == "deleted" {
				return nil
			}
			return resource.RetryableError(fmt.Errorf(
				"Waiting for Customer Gateway: %v", gateway.CustomerGatewayId))
		})
	}
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

			if *resp.CustomerGateways[0].State == "deleted" {
				continue
			}
		}

		return err
	}

	return nil
}

func testAccCheckCustomerGateway(gatewayResource string, cgw *ec2.CustomerGateway) resource.TestCheckFunc {
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

		resp, err := ec2conn.DescribeCustomerGateways(&ec2.DescribeCustomerGatewaysInput{
			Filters: []*ec2.Filter{gatewayFilter},
		})

		if err != nil {
			return err
		}

		respGateway := resp.CustomerGateways[0]
		*cgw = *respGateway

		return nil
	}
}

const testAccCustomerGatewayConfig = `
resource "aws_customer_gateway" "foo" {
	bgp_asn = 65000
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
	bgp_asn = 65000
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
	bgp_asn = 65000
	ip_address = "172.10.10.1"
	type = "ipsec.1"
	tags {
		Name = "foo-gateway"
		Another = "tag"
	}
}
`
