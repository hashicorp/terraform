package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

var defaultEgressAcl = &ec2.NetworkAclEntry{
	CidrBlock:  aws.String("0.0.0.0/0"),
	Egress:     aws.Bool(true),
	Protocol:   aws.String("-1"),
	RuleAction: aws.String("allow"),
	RuleNumber: aws.Int64(100),
}
var defaultIngressAcl = &ec2.NetworkAclEntry{
	CidrBlock:  aws.String("0.0.0.0/0"),
	Egress:     aws.Bool(false),
	Protocol:   aws.String("-1"),
	RuleAction: aws.String("allow"),
	RuleNumber: aws.Int64(100),
}

func TestAccAWSDefaultNetworkAcl_basic(t *testing.T) {
	var networkAcl ec2.NetworkAcl

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDefaultNetworkAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDefaultNetworkConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccGetAWSDefaultNetworkAcl("aws_default_network_acl.default", &networkAcl),
					testAccCheckAWSDefaultACLAttributes(&networkAcl, []*ec2.NetworkAclEntry{}, 0, 2),
				),
			},
		},
	})
}

func TestAccAWSDefaultNetworkAcl_basicIpv6Vpc(t *testing.T) {
	var networkAcl ec2.NetworkAcl

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDefaultNetworkAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDefaultNetworkConfig_basicIpv6Vpc,
				Check: resource.ComposeTestCheckFunc(
					testAccGetAWSDefaultNetworkAcl("aws_default_network_acl.default", &networkAcl),
					testAccCheckAWSDefaultACLAttributes(&networkAcl, []*ec2.NetworkAclEntry{}, 0, 4),
				),
			},
		},
	})
}

func TestAccAWSDefaultNetworkAcl_deny_ingress(t *testing.T) {
	// TestAccAWSDefaultNetworkAcl_deny_ingress will deny all Ingress rules, but
	// not Egress. We then expect there to be 3 rules, 2 AWS defaults and 1
	// additional Egress.
	var networkAcl ec2.NetworkAcl

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDefaultNetworkAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDefaultNetworkConfig_deny_ingress,
				Check: resource.ComposeTestCheckFunc(
					testAccGetAWSDefaultNetworkAcl("aws_default_network_acl.default", &networkAcl),
					testAccCheckAWSDefaultACLAttributes(&networkAcl, []*ec2.NetworkAclEntry{defaultEgressAcl}, 0, 2),
				),
			},
		},
	})
}

func TestAccAWSDefaultNetworkAcl_SubnetRemoval(t *testing.T) {
	var networkAcl ec2.NetworkAcl

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDefaultNetworkAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDefaultNetworkConfig_Subnets,
				Check: resource.ComposeTestCheckFunc(
					testAccGetAWSDefaultNetworkAcl("aws_default_network_acl.default", &networkAcl),
					testAccCheckAWSDefaultACLAttributes(&networkAcl, []*ec2.NetworkAclEntry{}, 2, 2),
				),
			},

			// Here the Subnets have been removed from the Default Network ACL Config,
			// but have not been reassigned. The result is that the Subnets are still
			// there, and we have a non-empty plan
			resource.TestStep{
				Config: testAccAWSDefaultNetworkConfig_Subnets_remove,
				Check: resource.ComposeTestCheckFunc(
					testAccGetAWSDefaultNetworkAcl("aws_default_network_acl.default", &networkAcl),
					testAccCheckAWSDefaultACLAttributes(&networkAcl, []*ec2.NetworkAclEntry{}, 2, 2),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSDefaultNetworkAcl_SubnetReassign(t *testing.T) {
	var networkAcl ec2.NetworkAcl

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDefaultNetworkAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDefaultNetworkConfig_Subnets,
				Check: resource.ComposeTestCheckFunc(
					testAccGetAWSDefaultNetworkAcl("aws_default_network_acl.default", &networkAcl),
					testAccCheckAWSDefaultACLAttributes(&networkAcl, []*ec2.NetworkAclEntry{}, 2, 2),
				),
			},

			// Here we've reassigned the subnets to a different ACL.
			// Without any otherwise association between the `aws_network_acl` and
			// `aws_default_network_acl` resources, we cannot guarantee that the
			// reassignment of the two subnets to the `aws_network_acl` will happen
			// before the update/read on the `aws_default_network_acl` resource.
			// Because of this, there could be a non-empty plan if a READ is done on
			// the default before the reassignment occurs on the other resource.
			//
			// For the sake of testing, here we introduce a depends_on attribute from
			// the default resource to the other acl resource, to ensure the latter's
			// update occurs first, and the former's READ will correctly read zero
			// subnets
			resource.TestStep{
				Config: testAccAWSDefaultNetworkConfig_Subnets_move,
				Check: resource.ComposeTestCheckFunc(
					testAccGetAWSDefaultNetworkAcl("aws_default_network_acl.default", &networkAcl),
					testAccCheckAWSDefaultACLAttributes(&networkAcl, []*ec2.NetworkAclEntry{}, 0, 2),
				),
			},
		},
	})
}

func testAccCheckAWSDefaultNetworkAclDestroy(s *terraform.State) error {
	// We can't destroy this resource; it comes and goes with the VPC itself.
	return nil
}

func testAccCheckAWSDefaultACLAttributes(acl *ec2.NetworkAcl, rules []*ec2.NetworkAclEntry, subnetCount int, hiddenRuleCount int) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		aclEntriesCount := len(acl.Entries)
		ruleCount := len(rules)

		// Default ACL has hidden rules we can't do anything about
		ruleCount = ruleCount + hiddenRuleCount

		if ruleCount != aclEntriesCount {
			return fmt.Errorf("Expected (%d) Rules, got (%d)", ruleCount, aclEntriesCount)
		}

		if len(acl.Associations) != subnetCount {
			return fmt.Errorf("Expected (%d) Subnets, got (%d)", subnetCount, len(acl.Associations))
		}

		return nil
	}
}

func testAccGetAWSDefaultNetworkAcl(n string, networkAcl *ec2.NetworkAcl) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Network ACL is set")
		}
		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		resp, err := conn.DescribeNetworkAcls(&ec2.DescribeNetworkAclsInput{
			NetworkAclIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err != nil {
			return err
		}

		if len(resp.NetworkAcls) > 0 && *resp.NetworkAcls[0].NetworkAclId == rs.Primary.ID {
			*networkAcl = *resp.NetworkAcls[0]
			return nil
		}

		return fmt.Errorf("Network Acls not found")
	}
}

const testAccAWSDefaultNetworkConfig_basic = `
resource "aws_vpc" "tftestvpc" {
  cidr_block = "10.1.0.0/16"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_basic"
  }
}

resource "aws_default_network_acl" "default" {
  default_network_acl_id = "${aws_vpc.tftestvpc.default_network_acl_id}"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_basic"
  }
}
`

const testAccAWSDefaultNetworkConfig_basicDefaultRules = `
resource "aws_vpc" "tftestvpc" {
  cidr_block = "10.1.0.0/16"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_basic"
  }
}

resource "aws_default_network_acl" "default" {
  default_network_acl_id = "${aws_vpc.tftestvpc.default_network_acl_id}"

  ingress {
    protocol   = -1
    rule_no    = 100
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }

  egress {
    protocol   = -1
    rule_no    = 100
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_basic"
  }
}
`

const testAccAWSDefaultNetworkConfig_deny = `
resource "aws_vpc" "tftestvpc" {
  cidr_block = "10.1.0.0/16"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_basic"
  }
}

resource "aws_default_network_acl" "default" {
  default_network_acl_id = "${aws_vpc.tftestvpc.default_network_acl_id}"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_basic"
  }
}
`

const testAccAWSDefaultNetworkConfig_deny_ingress = `
resource "aws_vpc" "tftestvpc" {
  cidr_block = "10.1.0.0/16"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_basic"
  }
}

resource "aws_default_network_acl" "default" {
  default_network_acl_id = "${aws_vpc.tftestvpc.default_network_acl_id}"

  egress {
    protocol   = -1
    rule_no    = 100
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_basic"
  }
}
`

const testAccAWSDefaultNetworkConfig_Subnets = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_SubnetRemoval"
  }
}

resource "aws_subnet" "one" {
  cidr_block = "10.1.111.0/24"
  vpc_id     = "${aws_vpc.foo.id}"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_SubnetRemoval"
  }
}

resource "aws_subnet" "two" {
  cidr_block = "10.1.1.0/24"
  vpc_id     = "${aws_vpc.foo.id}"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_SubnetRemoval"
  }
}

resource "aws_network_acl" "bar" {
  vpc_id = "${aws_vpc.foo.id}"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_SubnetRemoval"
  }
}

resource "aws_default_network_acl" "default" {
  default_network_acl_id = "${aws_vpc.foo.default_network_acl_id}"

  subnet_ids = ["${aws_subnet.one.id}", "${aws_subnet.two.id}"]

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_SubnetRemoval"
  }
}
`

const testAccAWSDefaultNetworkConfig_Subnets_remove = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_SubnetRemoval"
  }
}

resource "aws_subnet" "one" {
  cidr_block = "10.1.111.0/24"
  vpc_id     = "${aws_vpc.foo.id}"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_SubnetRemoval"
  }
}

resource "aws_subnet" "two" {
  cidr_block = "10.1.1.0/24"
  vpc_id     = "${aws_vpc.foo.id}"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_SubnetRemoval"
  }
}

resource "aws_network_acl" "bar" {
  vpc_id = "${aws_vpc.foo.id}"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_SubnetRemoval"
  }
}

resource "aws_default_network_acl" "default" {
  default_network_acl_id = "${aws_vpc.foo.default_network_acl_id}"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_SubnetRemoval"
  }
}
`

const testAccAWSDefaultNetworkConfig_Subnets_move = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_SubnetRemoval"
  }
}

resource "aws_subnet" "one" {
  cidr_block = "10.1.111.0/24"
  vpc_id     = "${aws_vpc.foo.id}"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_SubnetRemoval"
  }
}

resource "aws_subnet" "two" {
  cidr_block = "10.1.1.0/24"
  vpc_id     = "${aws_vpc.foo.id}"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_SubnetRemoval"
  }
}

resource "aws_network_acl" "bar" {
  vpc_id = "${aws_vpc.foo.id}"

  subnet_ids = ["${aws_subnet.one.id}", "${aws_subnet.two.id}"]

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_SubnetRemoval"
  }
}

resource "aws_default_network_acl" "default" {
  default_network_acl_id = "${aws_vpc.foo.default_network_acl_id}"

  depends_on = ["aws_network_acl.bar"]

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_SubnetRemoval"
  }
}
`

const testAccAWSDefaultNetworkConfig_basicIpv6Vpc = `
provider "aws" {
  region = "us-east-2"
}

resource "aws_vpc" "tftestvpc" {
	cidr_block = "10.1.0.0/16"
	assign_generated_ipv6_cidr_block = true

	tags {
		Name = "TestAccAWSDefaultNetworkAcl_basicIpv6Vpc"
	}
}

resource "aws_default_network_acl" "default" {
  default_network_acl_id = "${aws_vpc.tftestvpc.default_network_acl_id}"

  tags {
    Name = "TestAccAWSDefaultNetworkAcl_basicIpv6Vpc"
  }
}
`
