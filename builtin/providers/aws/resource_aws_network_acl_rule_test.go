package aws

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSNetworkAclRule_basic(t *testing.T) {
	var networkAcl ec2.NetworkAcl

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSNetworkAclRuleBasicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclRuleExists("aws_network_acl_rule.bar", &networkAcl),
				),
			},
		},
	})
}

func testAccCheckAWSNetworkAclRuleDestroy(s *terraform.State) error {

	for _, rs := range s.RootModule().Resources {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		if rs.Type != "aws_network_acl_rule" {
			continue
		}

		req := &ec2.DescribeNetworkAclsInput{
			NetworkAclIds: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeNetworkAcls(req)
		if err == nil {
			if len(resp.NetworkAcls) > 0 && *resp.NetworkAcls[0].NetworkAclId == rs.Primary.ID {
				networkAcl := resp.NetworkAcls[0]
				if networkAcl.Entries != nil {
					return fmt.Errorf("Network ACL Entries still exist")
				}
			}
		}

		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidNetworkAclID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSNetworkAclRuleExists(n string, networkAcl *ec2.NetworkAcl) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Network ACL Id is set")
		}

		req := &ec2.DescribeNetworkAclsInput{
			NetworkAclIds: []*string{aws.String(rs.Primary.Attributes["network_acl_id"])},
		}
		resp, err := conn.DescribeNetworkAcls(req)
		if err != nil {
			return err
		}
		if len(resp.NetworkAcls) != 1 {
			return fmt.Errorf("Network ACL not found")
		}
		egress, err := strconv.ParseBool(rs.Primary.Attributes["egress"])
		if err != nil {
			return err
		}
		ruleNo, err := strconv.ParseInt(rs.Primary.Attributes["rule_number"], 10, 64)
		if err != nil {
			return err
		}
		for _, e := range resp.NetworkAcls[0].Entries {
			if *e.RuleNumber == ruleNo && *e.Egress == egress {
				return nil
			}
		}
		return fmt.Errorf("Entry not found: %s", resp.NetworkAcls[0])
	}
}

const testAccAWSNetworkAclRuleBasicConfig = `
provider "aws" {
  region = "us-east-1"
}
resource "aws_vpc" "foo" {
	cidr_block = "10.3.0.0/16"
}
resource "aws_network_acl" "bar" {
	vpc_id = "${aws_vpc.foo.id}"
}
resource "aws_network_acl_rule" "bar" {
	network_acl_id = "${aws_network_acl.bar.id}"
	rule_number = 200
	egress = false
	protocol = "tcp"
	rule_action = "allow"
	cidr_block = "0.0.0.0/0"
	from_port = 22
	to_port = 22
}
`
