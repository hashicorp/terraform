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
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_network_acl_rule" {
			continue
		}

		rule_number := rs.Primary.Attributes["rule_number"].(int)
		egress := rs.Primary.Attributes["egress"].(bool)

		req := &ec2.DescribeNetworkAclsInput{
			NetworkAclIds: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeNetworkAcls(req)
		if err == nil {
			if len(resp.NetworkAcls) > 0 && *resp.NetworkAcls[0].NetworkAclId == rs.Primary.ID {
				networkAcl := resp.NetworkAcls[0]
				if networkAcl.Entries != nil {
					for _, i := range networkAcl.Entries {
						if *i.RuleNumber == int64(rule_number) && *i.Egress == egress {
							return fmt.Errorf("Network ACL Rule (%s) still exists.", rs.Primary.ID)
						}
					}
				}
			}
		}

		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		// Confirm error code is what we want
		if ec2err.Code() != "InvalidNetworkAclEntry.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSNetworkAclRuleExists(n string, networkAcl *ec2.NetworkAcl) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Security Group is set")
		}

		return nil
	}

	return nil
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
