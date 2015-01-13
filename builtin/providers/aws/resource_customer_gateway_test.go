package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func TestAccAWSCustomerGateway(t *testing.T) {
	var customerGateway ec2.CustomerGateway

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCustomerGatewayDestroy,
		Steps: []resource.TestStep {
			resource.TestStep{
				Config: testAccCustomerGatewayConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomerGatewayExists("aws_customer_gateway.foo", &customerGateway),
					resource.TestCheckResourceAttr("aws_customer_gateway.foo", "bgp_asn", "65000"),
					testAccCheckTags(&customerGateway.Tags, "bar", "baz"),
				),
			},
		},
	})
}

func TestAccAWSCustomerGateway_delete(t *testing.T) {
	var customerGateway ec2.CustomerGateway

	testDeleted := func(r string) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			_, ok := s.RootModule().Resources[r]
			if ok {
				return fmt.Errorf("Customer Gateway %q should have been deleted", r)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCustomerGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep {
				Config: testAccCustomerGatewayConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomerGatewayExists("aws_customer_gateway.foo", &customerGateway)),
			},
			resource.TestStep{
				Config: testAccNoCustomerGatewayConfig,
				Check:  resource.ComposeTestCheckFunc(
					testDeleted("aws_customer_gateway.foo")),
			},
		},
	})
}

func testAccCheckCustomerGatewayDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_customer_gateway" {
			continue
		}

		// Try to find the resource
		resp, err := conn.DescribeCustomerGateways(
			[]string{rs.Primary.ID}, ec2.NewFilter())
		if err == nil {
			if len(resp.CustomerGateways) > 0 {
				return fmt.Errorf("still exist.")
			}

			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(*ec2.Error)
		if !ok {
			return err
		}
		if ec2err.Code != "InvalidCustomerGatewayID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckCustomerGatewayExists(n string, customerGateway *ec2.CustomerGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := conn.DescribeCustomerGateways(
			[]string{rs.Primary.ID}, ec2.NewFilter())
		if err != nil {
			return err
		}
		if len(resp.CustomerGateways) == 0 {
			return fmt.Errorf("Customer Gateway not found")
		}

		*customerGateway = resp.CustomerGateways[0]

		return nil
	}
}

const testAccCustomerGatewayConfig = `
resource "aws_customer_gateway" "foo" {
	bgp_asn = 65000
	ip_address = "182.72.16.113"
	type = "ipsec.1"
	tags {
		bar = "baz"
	}
}
`
const testAccNoCustomerGatewayConfig = ``

