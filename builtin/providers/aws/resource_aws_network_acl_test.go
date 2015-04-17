package aws

import (
	"fmt"
	"testing"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSNetworkAcl_EgressAndIngressRules(t *testing.T) {
	var networkAcl ec2.NetworkACL

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
						"aws_network_acl.bar", "ingress.3409203205.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "ingress.3409203205.rule_no", "1"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "ingress.3409203205.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "ingress.3409203205.to_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "ingress.3409203205.action", "allow"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "ingress.3409203205.cidr_block", "10.3.10.3/18"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "egress.2579689292.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "egress.2579689292.rule_no", "2"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "egress.2579689292.from_port", "443"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "egress.2579689292.to_port", "443"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "egress.2579689292.cidr_block", "10.3.2.3/18"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.bar", "egress.2579689292.action", "allow"),
				),
			},
		},
	})
}

func TestAccAWSNetworkAcl_OnlyIngressRules(t *testing.T) {
	var networkAcl ec2.NetworkACL

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSNetworkAclIngressConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists("aws_network_acl.foos", &networkAcl),
					// testAccCheckSubnetAssociation("aws_network_acl.foos", "aws_subnet.blob"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.2750166237.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.2750166237.rule_no", "2"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.2750166237.from_port", "443"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.2750166237.to_port", "443"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.2750166237.action", "deny"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.2750166237.cidr_block", "10.2.2.3/18"),
				),
			},
		},
	})
}

func TestAccAWSNetworkAcl_OnlyIngressRulesChange(t *testing.T) {
	var networkAcl ec2.NetworkACL

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSNetworkAclIngressConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists("aws_network_acl.foos", &networkAcl),
					testIngressRuleLength(&networkAcl, 2),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.37211640.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.37211640.rule_no", "1"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.37211640.from_port", "0"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.37211640.to_port", "22"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.37211640.action", "deny"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.37211640.cidr_block", "10.2.2.3/18"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.2750166237.from_port", "443"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.2750166237.rule_no", "2"),
				),
			},
			resource.TestStep{
				Config: testAccAWSNetworkAclIngressConfigChange,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists("aws_network_acl.foos", &networkAcl),
					testIngressRuleLength(&networkAcl, 1),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.37211640.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.37211640.rule_no", "1"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.37211640.from_port", "0"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.37211640.to_port", "22"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.37211640.action", "deny"),
					resource.TestCheckResourceAttr(
						"aws_network_acl.foos", "ingress.37211640.cidr_block", "10.2.2.3/18"),
				),
			},
		},
	})
}

func TestAccAWSNetworkAcl_OnlyEgressRules(t *testing.T) {
	var networkAcl ec2.NetworkACL

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSNetworkAclEgressConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists("aws_network_acl.bond", &networkAcl),
					testAccCheckTagsSDK(&networkAcl.Tags, "foo", "bar"),
				),
			},
		},
	})
}

func TestAccAWSNetworkAcl_SubnetChange(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSNetworkAclSubnetConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetIsAssociatedWithAcl("aws_network_acl.bar", "aws_subnet.old"),
				),
			},
			resource.TestStep{
				Config: testAccAWSNetworkAclSubnetConfigChange,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetIsNotAssociatedWithAcl("aws_network_acl.bar", "aws_subnet.old"),
					testAccCheckSubnetIsAssociatedWithAcl("aws_network_acl.bar", "aws_subnet.new"),
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
		resp, err := conn.DescribeNetworkACLs(&ec2.DescribeNetworkACLsInput{
			NetworkACLIDs: []*string{aws.String(rs.Primary.ID)},
		})
		if err == nil {
			if len(resp.NetworkACLs) > 0 && *resp.NetworkACLs[0].NetworkACLID == rs.Primary.ID {
				return fmt.Errorf("Network Acl (%s) still exists.", rs.Primary.ID)
			}

			return nil
		}

		ec2err, ok := err.(aws.APIError)
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

func testAccCheckAWSNetworkAclExists(n string, networkAcl *ec2.NetworkACL) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Security Group is set")
		}
		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		resp, err := conn.DescribeNetworkACLs(&ec2.DescribeNetworkACLsInput{
			NetworkACLIDs: []*string{aws.String(rs.Primary.ID)},
		})
		if err != nil {
			return err
		}

		if len(resp.NetworkACLs) > 0 && *resp.NetworkACLs[0].NetworkACLID == rs.Primary.ID {
			*networkAcl = *resp.NetworkACLs[0]
			return nil
		}

		return fmt.Errorf("Network Acls not found")
	}
}

func testIngressRuleLength(networkAcl *ec2.NetworkACL, length int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var ingressEntries []*ec2.NetworkACLEntry
		for _, e := range networkAcl.Entries {
			if *e.Egress == false {
				ingressEntries = append(ingressEntries, e)
			}
		}
		// There is always a default rule (ALL Traffic ... DENY)
		// so we have to increase the lenght by 1
		if len(ingressEntries) != length+1 {
			return fmt.Errorf("Invalid number of ingress entries found; count = %d", len(ingressEntries))
		}
		return nil
	}
}

func testAccCheckSubnetIsAssociatedWithAcl(acl string, sub string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		networkAcl := s.RootModule().Resources[acl]
		subnet := s.RootModule().Resources[sub]

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := conn.DescribeNetworkACLs(&ec2.DescribeNetworkACLsInput{
			NetworkACLIDs: []*string{aws.String(networkAcl.Primary.ID)},
			Filters: []*ec2.Filter{
				&ec2.Filter{
					Name:   aws.String("association.subnet-id"),
					Values: []*string{aws.String(subnet.Primary.ID)},
				},
			},
		})
		if err != nil {
			return err
		}
		if len(resp.NetworkACLs) > 0 {
			return nil
		}

		// r, _ := conn.NetworkACLs([]string{}, ec2.NewFilter())
		// fmt.Printf("\n\nall acls\n %#v\n\n", r.NetworkAcls)
		// conn.NetworkAcls([]string{}, filter)

		return fmt.Errorf("Network Acl %s is not associated with subnet %s", acl, sub)
	}
}

func testAccCheckSubnetIsNotAssociatedWithAcl(acl string, subnet string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		networkAcl := s.RootModule().Resources[acl]
		subnet := s.RootModule().Resources[subnet]

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := conn.DescribeNetworkACLs(&ec2.DescribeNetworkACLsInput{
			NetworkACLIDs: []*string{aws.String(networkAcl.Primary.ID)},
			Filters: []*ec2.Filter{
				&ec2.Filter{
					Name:   aws.String("association.subnet-id"),
					Values: []*string{aws.String(subnet.Primary.ID)},
				},
			},
		})

		if err != nil {
			return err
		}
		if len(resp.NetworkACLs) > 0 {
			return fmt.Errorf("Network Acl %s is still associated with subnet %s", acl, subnet)
		}
		return nil
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
		rule_no = 1
		action = "deny"
		cidr_block =  "10.2.2.3/18"
		from_port = 0
		to_port = 22
	}
	ingress = {
		protocol = "tcp"
		rule_no = 2
		action = "deny"
		cidr_block =  "10.2.2.3/18"
		from_port = 443
		to_port = 443
	}
	subnet_id = "${aws_subnet.blob.id}"
}
`
const testAccAWSNetworkAclIngressConfigChange = `
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
		rule_no = 1
		action = "deny"
		cidr_block =  "10.2.2.3/18"
		from_port = 0
		to_port = 22
	}
	subnet_id = "${aws_subnet.blob.id}"
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

	tags {
		foo = "bar"
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
const testAccAWSNetworkAclSubnetConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}
resource "aws_subnet" "old" {
	cidr_block = "10.1.111.0/24"
	vpc_id = "${aws_vpc.foo.id}"
	map_public_ip_on_launch = true
}
resource "aws_subnet" "new" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
	map_public_ip_on_launch = true
}
resource "aws_network_acl" "roll" {
	vpc_id = "${aws_vpc.foo.id}"
	subnet_id = "${aws_subnet.new.id}"
}
resource "aws_network_acl" "bar" {
	vpc_id = "${aws_vpc.foo.id}"
	subnet_id = "${aws_subnet.old.id}"
}
`

const testAccAWSNetworkAclSubnetConfigChange = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}
resource "aws_subnet" "old" {
	cidr_block = "10.1.111.0/24"
	vpc_id = "${aws_vpc.foo.id}"
	map_public_ip_on_launch = true
}
resource "aws_subnet" "new" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
	map_public_ip_on_launch = true
}
resource "aws_network_acl" "bar" {
	vpc_id = "${aws_vpc.foo.id}"
	subnet_id = "${aws_subnet.new.id}"
}
`
