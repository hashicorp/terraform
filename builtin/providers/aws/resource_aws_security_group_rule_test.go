package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestIpPermissionIDHash(t *testing.T) {
	simple := &ec2.IPPermission{
		IPProtocol: aws.String("tcp"),
		FromPort:   aws.Int64(int64(80)),
		ToPort:     aws.Int64(int64(8000)),
		IPRanges: []*ec2.IPRange{
			&ec2.IPRange{
				CIDRIP: aws.String("10.0.0.0/8"),
			},
		},
	}

	egress := &ec2.IPPermission{
		IPProtocol: aws.String("tcp"),
		FromPort:   aws.Int64(int64(80)),
		ToPort:     aws.Int64(int64(8000)),
		IPRanges: []*ec2.IPRange{
			&ec2.IPRange{
				CIDRIP: aws.String("10.0.0.0/8"),
			},
		},
	}

	egress_all := &ec2.IPPermission{
		IPProtocol: aws.String("-1"),
		IPRanges: []*ec2.IPRange{
			&ec2.IPRange{
				CIDRIP: aws.String("10.0.0.0/8"),
			},
		},
	}

	vpc_security_group_source := &ec2.IPPermission{
		IPProtocol: aws.String("tcp"),
		FromPort:   aws.Int64(int64(80)),
		ToPort:     aws.Int64(int64(8000)),
		UserIDGroupPairs: []*ec2.UserIDGroupPair{
			&ec2.UserIDGroupPair{
				UserID:  aws.String("987654321"),
				GroupID: aws.String("sg-12345678"),
			},
			&ec2.UserIDGroupPair{
				UserID:  aws.String("123456789"),
				GroupID: aws.String("sg-987654321"),
			},
			&ec2.UserIDGroupPair{
				UserID:  aws.String("123456789"),
				GroupID: aws.String("sg-12345678"),
			},
		},
	}

	security_group_source := &ec2.IPPermission{
		IPProtocol: aws.String("tcp"),
		FromPort:   aws.Int64(int64(80)),
		ToPort:     aws.Int64(int64(8000)),
		UserIDGroupPairs: []*ec2.UserIDGroupPair{
			&ec2.UserIDGroupPair{
				UserID:    aws.String("987654321"),
				GroupName: aws.String("my-security-group"),
			},
			&ec2.UserIDGroupPair{
				UserID:    aws.String("123456789"),
				GroupName: aws.String("my-security-group"),
			},
			&ec2.UserIDGroupPair{
				UserID:    aws.String("123456789"),
				GroupName: aws.String("my-other-security-group"),
			},
		},
	}

	// hardcoded hashes, to detect future change
	cases := []struct {
		Input  *ec2.IPPermission
		Type   string
		Output string
	}{
		{simple, "ingress", "sg-82613597"},
		{egress, "egress", "sg-363054720"},
		{egress_all, "egress", "sg-2766285362"},
		{vpc_security_group_source, "egress", "sg-2661404947"},
		{security_group_source, "egress", "sg-1841245863"},
	}

	for _, tc := range cases {
		actual := ipPermissionIDHash(tc.Type, tc.Input)
		if actual != tc.Output {
			t.Errorf("input: %s - %s\noutput: %s", tc.Type, tc.Input, actual)
		}
	}
}

func TestAccAWSSecurityGroupRule_Ingress_VPC(t *testing.T) {
	var group ec2.SecurityGroup

	testRuleCount := func(*terraform.State) error {
		if len(group.IPPermissions) != 1 {
			return fmt.Errorf("Wrong Security Group rule count, expected %d, got %d",
				1, len(group.IPPermissions))
		}

		rule := group.IPPermissions[0]
		if *rule.FromPort != int64(80) {
			return fmt.Errorf("Wrong Security Group port setting, expected %d, got %d",
				80, int(*rule.FromPort))
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSecurityGroupRuleIngressConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupRuleAttributes(&group, "ingress"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rule.ingress_1", "from_port", "80"),
					testRuleCount,
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRule_Ingress_Classic(t *testing.T) {
	var group ec2.SecurityGroup

	testRuleCount := func(*terraform.State) error {
		if len(group.IPPermissions) != 1 {
			return fmt.Errorf("Wrong Security Group rule count, expected %d, got %d",
				1, len(group.IPPermissions))
		}

		rule := group.IPPermissions[0]
		if *rule.FromPort != int64(80) {
			return fmt.Errorf("Wrong Security Group port setting, expected %d, got %d",
				80, int(*rule.FromPort))
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSecurityGroupRuleIngressClassicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupRuleAttributes(&group, "ingress"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rule.ingress_1", "from_port", "80"),
					testRuleCount,
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRule_MultiIngress(t *testing.T) {
	var group ec2.SecurityGroup

	testMultiRuleCount := func(*terraform.State) error {
		if len(group.IPPermissions) != 2 {
			return fmt.Errorf("Wrong Security Group rule count, expected %d, got %d",
				2, len(group.IPPermissions))
		}

		var rule *ec2.IPPermission
		for _, r := range group.IPPermissions {
			if *r.FromPort == int64(80) {
				rule = r
			}
		}

		if *rule.ToPort != int64(8000) {
			return fmt.Errorf("Wrong Security Group port 2 setting, expected %d, got %d",
				8000, int(*rule.ToPort))
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSecurityGroupRuleConfigMultiIngress,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.web", &group),
					testMultiRuleCount,
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRule_Egress(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSecurityGroupRuleEgressConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupRuleAttributes(&group, "egress"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRule_SelfReference(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSecurityGroupRuleConfigSelfReference,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.web", &group),
				),
			},
		},
	})
}

func testAccCheckAWSSecurityGroupRuleDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_security_group" {
			continue
		}

		// Retrieve our group
		req := &ec2.DescribeSecurityGroupsInput{
			GroupIDs: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeSecurityGroups(req)
		if err == nil {
			if len(resp.SecurityGroups) > 0 && *resp.SecurityGroups[0].GroupID == rs.Primary.ID {
				return fmt.Errorf("Security Group (%s) still exists.", rs.Primary.ID)
			}

			return nil
		}

		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		// Confirm error code is what we want
		if ec2err.Code() != "InvalidGroup.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSSecurityGroupRuleExists(n string, group *ec2.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Security Group is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		req := &ec2.DescribeSecurityGroupsInput{
			GroupIDs: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeSecurityGroups(req)
		if err != nil {
			return err
		}

		if len(resp.SecurityGroups) > 0 && *resp.SecurityGroups[0].GroupID == rs.Primary.ID {
			*group = *resp.SecurityGroups[0]
			return nil
		}

		return fmt.Errorf("Security Group not found")
	}
}

func testAccCheckAWSSecurityGroupRuleAttributes(group *ec2.SecurityGroup, ruleType string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		p := &ec2.IPPermission{
			FromPort:   aws.Int64(80),
			ToPort:     aws.Int64(8000),
			IPProtocol: aws.String("tcp"),
			IPRanges:   []*ec2.IPRange{&ec2.IPRange{CIDRIP: aws.String("10.0.0.0/8")}},
		}
		var rules []*ec2.IPPermission
		if ruleType == "ingress" {
			rules = group.IPPermissions
		} else {
			rules = group.IPPermissionsEgress
		}

		if len(rules) == 0 {
			return fmt.Errorf("No IPPerms")
		}

		// Compare our ingress
		if !reflect.DeepEqual(rules[0], p) {
			return fmt.Errorf(
				"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
				rules[0],
				p)
		}

		return nil
	}
}

const testAccAWSSecurityGroupRuleIngressConfig = `
resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"

        tags {
                Name = "tf-acc-test"
        }
}

resource "aws_security_group_rule" "ingress_1" {
  type = "ingress"
  protocol = "tcp"
  from_port = 80
  to_port = 8000
  cidr_blocks = ["10.0.0.0/8"]

  security_group_id = "${aws_security_group.web.id}"
}
`

const testAccAWSSecurityGroupRuleIngressClassicConfig = `
provider "aws" {
        region = "us-east-1"
}

resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"

        tags {
                Name = "tf-acc-test"
        }
}

resource "aws_security_group_rule" "ingress_1" {
  type = "ingress"
  protocol = "tcp"
  from_port = 80
  to_port = 8000
  cidr_blocks = ["10.0.0.0/8"]

  security_group_id = "${aws_security_group.web.id}"
}
`

const testAccAWSSecurityGroupRuleEgressConfig = `
resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"

        tags {
                Name = "tf-acc-test"
        }
}

resource "aws_security_group_rule" "egress_1" {
  type = "egress"
  protocol = "tcp"
  from_port = 80
  to_port = 8000
  cidr_blocks = ["10.0.0.0/8"]

  security_group_id = "${aws_security_group.web.id}"
}
`

const testAccAWSSecurityGroupRuleConfigMultiIngress = `
resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example_2"
  description = "Used in the terraform acceptance tests"
}

resource "aws_security_group" "worker" {
  name = "terraform_acceptance_test_example_worker"
  description = "Used in the terraform acceptance tests"
}


resource "aws_security_group_rule" "ingress_1" {
  type = "ingress"
  protocol = "tcp"
  from_port = 22
  to_port = 22
  cidr_blocks = ["10.0.0.0/8"]

  security_group_id = "${aws_security_group.web.id}"
}

resource "aws_security_group_rule" "ingress_2" {
  type = "ingress"
  protocol = "tcp"
  from_port = 80
  to_port = 8000
        self = true

  security_group_id = "${aws_security_group.web.id}"
}
`

// check for GH-1985 regression
const testAccAWSSecurityGroupRuleConfigSelfReference = `
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
  tags {
    Name = "sg-self-test"
  }
}

resource "aws_security_group" "web" {
  name = "main"
  vpc_id = "${aws_vpc.main.id}"
  tags {
    Name = "sg-self-test"
  }
}

resource "aws_security_group_rule" "self" {
  type = "ingress"
  protocol = "-1"
  from_port = 0
  to_port = 0
  self = true
  security_group_id = "${aws_security_group.web.id}"
}
`
