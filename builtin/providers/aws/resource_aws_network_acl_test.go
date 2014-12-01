package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
	// "github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	// "github.com/hashicorp/terraform/helper/schema"
)

func TestAccAWSNetworkAclsWithEgressAndIngressRulesSneha(t *testing.T) {
	var networkAcl ec2.NetworkAcl

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSNetworkAclEgressNIngressConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists("aws_network_acl.bar", &networkAcl),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "ingress.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "ingress.0.rule_no", "1"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "ingress.0.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "ingress.0.to_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "ingress.0.action", "allow"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "ingress.0.cidr_block", "10.3.10.3/18"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "egress.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "egress.0.rule_no", "2"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "egress.0.from_port", "443"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "egress.0.to_port", "443"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "egress.0.cidr_block", "10.3.2.3/18"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "egress.0.action", "allow"),
				),
			},
		},
	})
}

func TestAccAWSNetworkAclsOnlyIngressRulesSneha(t *testing.T) {
	var networkAcl ec2.NetworkAcl

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSNetworkAclIngressConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists("aws_network_acl.foos", &networkAcl),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.0.rule_no", "2"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.0.from_port", "0"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.0.to_port", "22"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.0.action", "deny"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.0.cidr_block", "10.2.2.3/18"),
				),
			},
		},
	})
}

func TestAccAWSNetworkAclsOnlyEgressRulesSneha(t *testing.T) {
	var networkAcl ec2.NetworkAcl

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSNetworkAclEgressConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists("aws_network_acl.bond", &networkAcl),
				),
			},
		},
	})
}

func testAccCheckAWSNetworkAclDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_network" {
			continue
		}

		// Retrieve the network acl
		resp, err := conn.NetworkAcls([]string{rs.Primary.ID}, ec2.NewFilter())
		if err == nil {
			if len(resp.NetworkAcls) > 0 && resp.NetworkAcls[0].NetworkAclId == rs.Primary.ID {
				return fmt.Errorf("Network Acl (%s) still exists.", rs.Primary.ID)
			}

			return nil
		}

		ec2err, ok := err.(*ec2.Error)
		if !ok {
			return err
		}
		// Confirm error code is what we want
		if ec2err.Code != "InvalidNetworkAclID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSNetworkAclExists(n string, networkAcl *ec2.NetworkAcl) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Security Group is set")
		}
		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		resp, err := conn.NetworkAcls([]string{rs.Primary.ID}, nil)
		if err != nil {
			return err
		}

		if len(resp.NetworkAcls) > 0 && resp.NetworkAcls[0].NetworkAclId == rs.Primary.ID {
			*networkAcl = resp.NetworkAcls[0]
			return nil
		}

		return fmt.Errorf("Network Acls not found")
	}
}

const testAccAWSNetworkAclIngressConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}
resource "aws_subnet" "blob" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
	map_public_ip_on_launch = true
}
resource "aws_network_acl" "foos" {
	vpc_id = "${aws_vpc.foo.id}"
	ingress = {
		protocol = "tcp"
		rule_no = 2
		action = "deny"
		cidr_block =  "10.2.2.3/18"
		from_port = 0
		to_port = 22
	}
}
`

const testAccAWSNetworkAclEgressConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.2.0.0/16"
}
resource "aws_subnet" "blob" {
	cidr_block = "10.2.0.0/24"
	vpc_id = "${aws_vpc.foo.id}"
	map_public_ip_on_launch = true
}
resource "aws_network_acl" "bond" {
	vpc_id = "${aws_vpc.foo.id}"
	egress = {
		protocol = "tcp"
		rule_no = 2
		action = "allow"
		cidr_block =  "10.2.2.3/18"
		from_port = 443
		to_port = 443
	}

	egress = {
		protocol = "tcp"
		rule_no = 1
		action = "allow"
		cidr_block =  "10.2.10.3/18"
		from_port = 80
		to_port = 80
	}

	egress = {
		protocol = "tcp"
		rule_no = 3
		action = "allow"
		cidr_block =  "10.2.10.3/18"
		from_port = 22
		to_port = 22
	}
}
`

const testAccAWSNetworkAclEgressNIngressConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.3.0.0/16"
}
resource "aws_subnet" "blob" {
	cidr_block = "10.3.0.0/24"
	vpc_id = "${aws_vpc.foo.id}"
	map_public_ip_on_launch = true
}
resource "aws_network_acl" "bar" {
	vpc_id = "${aws_vpc.foo.id}"
	egress = {
		protocol = "tcp"
		rule_no = 2
		action = "allow"
		cidr_block =  "10.3.2.3/18"
		from_port = 443
		to_port = 443
	}

	ingress = {
		protocol = "tcp"
		rule_no = 1
		action = "allow"
		cidr_block =  "10.3.10.3/18"
		from_port = 80
		to_port = 80
	}
}
`
