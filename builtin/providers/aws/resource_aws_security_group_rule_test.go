package aws

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestIpPermissionIDHash(t *testing.T) {
	simple := &ec2.IpPermission{
		IpProtocol: aws.String("tcp"),
		FromPort:   aws.Int64(int64(80)),
		ToPort:     aws.Int64(int64(8000)),
		IpRanges: []*ec2.IpRange{
			{
				CidrIp: aws.String("10.0.0.0/8"),
			},
		},
	}

	egress := &ec2.IpPermission{
		IpProtocol: aws.String("tcp"),
		FromPort:   aws.Int64(int64(80)),
		ToPort:     aws.Int64(int64(8000)),
		IpRanges: []*ec2.IpRange{
			{
				CidrIp: aws.String("10.0.0.0/8"),
			},
		},
	}

	egress_all := &ec2.IpPermission{
		IpProtocol: aws.String("-1"),
		IpRanges: []*ec2.IpRange{
			{
				CidrIp: aws.String("10.0.0.0/8"),
			},
		},
	}

	vpc_security_group_source := &ec2.IpPermission{
		IpProtocol: aws.String("tcp"),
		FromPort:   aws.Int64(int64(80)),
		ToPort:     aws.Int64(int64(8000)),
		UserIdGroupPairs: []*ec2.UserIdGroupPair{
			{
				UserId:  aws.String("987654321"),
				GroupId: aws.String("sg-12345678"),
			},
			{
				UserId:  aws.String("123456789"),
				GroupId: aws.String("sg-987654321"),
			},
			{
				UserId:  aws.String("123456789"),
				GroupId: aws.String("sg-12345678"),
			},
		},
	}

	security_group_source := &ec2.IpPermission{
		IpProtocol: aws.String("tcp"),
		FromPort:   aws.Int64(int64(80)),
		ToPort:     aws.Int64(int64(8000)),
		UserIdGroupPairs: []*ec2.UserIdGroupPair{
			{
				UserId:    aws.String("987654321"),
				GroupName: aws.String("my-security-group"),
			},
			{
				UserId:    aws.String("123456789"),
				GroupName: aws.String("my-security-group"),
			},
			{
				UserId:    aws.String("123456789"),
				GroupName: aws.String("my-other-security-group"),
			},
		},
	}

	// hardcoded hashes, to detect future change
	cases := []struct {
		Input  *ec2.IpPermission
		Type   string
		Output string
	}{
		{simple, "ingress", "sgrule-3403497314"},
		{egress, "egress", "sgrule-1173186295"},
		{egress_all, "egress", "sgrule-766323498"},
		{vpc_security_group_source, "egress", "sgrule-351225364"},
		{security_group_source, "egress", "sgrule-2198807188"},
	}

	for _, tc := range cases {
		actual := ipPermissionIDHash("sg-12345", tc.Type, tc.Input)
		if actual != tc.Output {
			t.Errorf("input: %s - %s\noutput: %s", tc.Type, tc.Input, actual)
		}
	}
}

func TestAccAWSSecurityGroupRule_Ingress_VPC(t *testing.T) {
	var group ec2.SecurityGroup
	rInt := acctest.RandInt()

	testRuleCount := func(*terraform.State) error {
		if len(group.IpPermissions) != 1 {
			return fmt.Errorf("Wrong Security Group rule count, expected %d, got %d",
				1, len(group.IpPermissions))
		}

		rule := group.IpPermissions[0]
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
			{
				Config: testAccAWSSecurityGroupRuleIngressConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupRuleAttributes("aws_security_group_rule.ingress_1", &group, nil, "ingress"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rule.ingress_1", "from_port", "80"),
					testRuleCount,
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRule_Ingress_Protocol(t *testing.T) {
	var group ec2.SecurityGroup

	testRuleCount := func(*terraform.State) error {
		if len(group.IpPermissions) != 1 {
			return fmt.Errorf("Wrong Security Group rule count, expected %d, got %d",
				1, len(group.IpPermissions))
		}

		rule := group.IpPermissions[0]
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
			{
				Config: testAccAWSSecurityGroupRuleIngress_protocolConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupRuleAttributes("aws_security_group_rule.ingress_1", &group, nil, "ingress"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rule.ingress_1", "from_port", "80"),
					testRuleCount,
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRule_Ingress_Ipv6(t *testing.T) {
	var group ec2.SecurityGroup

	testRuleCount := func(*terraform.State) error {
		if len(group.IpPermissions) != 1 {
			return fmt.Errorf("Wrong Security Group rule count, expected %d, got %d",
				1, len(group.IpPermissions))
		}

		rule := group.IpPermissions[0]
		if *rule.FromPort != int64(80) {
			return fmt.Errorf("Wrong Security Group port setting, expected %d, got %d",
				80, int(*rule.FromPort))
		}

		ipv6Address := rule.Ipv6Ranges[0]
		if *ipv6Address.CidrIpv6 != "::/0" {
			return fmt.Errorf("Wrong Security Group IPv6 address, expected %s, got %s",
				"::/0", *ipv6Address.CidrIpv6)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRuleIngress_ipv6Config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.web", &group),
					testRuleCount,
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRule_Ingress_Classic(t *testing.T) {
	var group ec2.SecurityGroup
	rInt := acctest.RandInt()

	testRuleCount := func(*terraform.State) error {
		if len(group.IpPermissions) != 1 {
			return fmt.Errorf("Wrong Security Group rule count, expected %d, got %d",
				1, len(group.IpPermissions))
		}

		rule := group.IpPermissions[0]
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
			{
				Config: testAccAWSSecurityGroupRuleIngressClassicConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupRuleAttributes("aws_security_group_rule.ingress_1", &group, nil, "ingress"),
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
		if len(group.IpPermissions) != 2 {
			return fmt.Errorf("Wrong Security Group rule count, expected %d, got %d",
				2, len(group.IpPermissions))
		}

		var rule *ec2.IpPermission
		for _, r := range group.IpPermissions {
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
			{
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
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRuleEgressConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupRuleAttributes("aws_security_group_rule.egress_1", &group, nil, "egress"),
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
			{
				Config: testAccAWSSecurityGroupRuleConfigSelfReference,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.web", &group),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRule_ExpectInvalidTypeError(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAWSSecurityGroupRuleExpectInvalidType(rInt),
				ExpectError: regexp.MustCompile(`\\"type\\" contains an invalid Security Group Rule type \\"foobar\\"`),
			},
		},
	})
}

func TestAccAWSSecurityGroupRule_ExpectInvalidCIDR(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAWSSecurityGroupRuleInvalidIPv4CIDR(rInt),
				ExpectError: regexp.MustCompile("invalid CIDR address: 1.2.3.4/33"),
			},
			{
				Config:      testAccAWSSecurityGroupRuleInvalidIPv6CIDR(rInt),
				ExpectError: regexp.MustCompile("invalid CIDR address: ::/244"),
			},
		},
	})
}

// testing partial match implementation
func TestAccAWSSecurityGroupRule_PartialMatching_basic(t *testing.T) {
	var group ec2.SecurityGroup
	rInt := acctest.RandInt()

	p := ec2.IpPermission{
		FromPort:   aws.Int64(80),
		ToPort:     aws.Int64(80),
		IpProtocol: aws.String("tcp"),
		IpRanges: []*ec2.IpRange{
			{CidrIp: aws.String("10.0.2.0/24")},
			{CidrIp: aws.String("10.0.3.0/24")},
			{CidrIp: aws.String("10.0.4.0/24")},
		},
	}

	o := ec2.IpPermission{
		FromPort:   aws.Int64(80),
		ToPort:     aws.Int64(80),
		IpProtocol: aws.String("tcp"),
		IpRanges: []*ec2.IpRange{
			{CidrIp: aws.String("10.0.5.0/24")},
		},
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulePartialMatching(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupRuleAttributes("aws_security_group_rule.ingress", &group, &p, "ingress"),
					testAccCheckAWSSecurityGroupRuleAttributes("aws_security_group_rule.other", &group, &o, "ingress"),
					testAccCheckAWSSecurityGroupRuleAttributes("aws_security_group_rule.nat_ingress", &group, &o, "ingress"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRule_PartialMatching_Source(t *testing.T) {
	var group ec2.SecurityGroup
	var nat ec2.SecurityGroup
	var p ec2.IpPermission
	rInt := acctest.RandInt()

	// This function creates the expected IPPermission with the group id from an
	// external security group, needed because Security Group IDs are generated on
	// AWS side and can't be known ahead of time.
	setupSG := func(*terraform.State) error {
		if nat.GroupId == nil {
			return fmt.Errorf("Error: nat group has nil GroupID")
		}

		p = ec2.IpPermission{
			FromPort:   aws.Int64(80),
			ToPort:     aws.Int64(80),
			IpProtocol: aws.String("tcp"),
			UserIdGroupPairs: []*ec2.UserIdGroupPair{
				{GroupId: nat.GroupId},
			},
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulePartialMatching_Source(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.nat", &nat),
					setupSG,
					testAccCheckAWSSecurityGroupRuleAttributes("aws_security_group_rule.source_ingress", &group, &p, "ingress"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRule_Issue5310(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRuleIssue5310,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.issue_5310", &group),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRule_Race(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRuleRace,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.race", &group),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRule_SelfSource(t *testing.T) {
	var group ec2.SecurityGroup
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRuleSelfInSource(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.web", &group),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRule_PrefixListEgress(t *testing.T) {
	var group ec2.SecurityGroup
	var endpoint ec2.VpcEndpoint
	var p ec2.IpPermission

	// This function creates the expected IPPermission with the prefix list ID from
	// the VPC Endpoint created in the test
	setupSG := func(*terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		prefixListInput := &ec2.DescribePrefixListsInput{
			Filters: []*ec2.Filter{
				{Name: aws.String("prefix-list-name"), Values: []*string{endpoint.ServiceName}},
			},
		}

		log.Printf("[DEBUG] Reading VPC Endpoint prefix list: %s", prefixListInput)
		prefixListsOutput, err := conn.DescribePrefixLists(prefixListInput)

		if err != nil {
			_, ok := err.(awserr.Error)
			if !ok {
				return fmt.Errorf("Error reading VPC Endpoint prefix list: %s", err.Error())
			}
		}

		if len(prefixListsOutput.PrefixLists) != 1 {
			return fmt.Errorf("There are multiple prefix lists associated with the service name '%s'. Unexpected", prefixListsOutput)
		}

		p = ec2.IpPermission{
			IpProtocol: aws.String("-1"),
			PrefixListIds: []*ec2.PrefixListId{
				{PrefixListId: prefixListsOutput.PrefixLists[0].PrefixListId},
			},
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulePrefixListEgressConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRuleExists("aws_security_group.egress", &group),
					// lookup info on the VPC Endpoint created, to populate the expected
					// IP Perm
					testAccCheckVpcEndpointExists("aws_vpc_endpoint.s3-us-west-2", &endpoint),
					setupSG,
					testAccCheckAWSSecurityGroupRuleAttributes("aws_security_group_rule.egress_1", &group, &p, "egress"),
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
			GroupIds: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeSecurityGroups(req)
		if err == nil {
			if len(resp.SecurityGroups) > 0 && *resp.SecurityGroups[0].GroupId == rs.Primary.ID {
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
			GroupIds: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeSecurityGroups(req)
		if err != nil {
			return err
		}

		if len(resp.SecurityGroups) > 0 && *resp.SecurityGroups[0].GroupId == rs.Primary.ID {
			*group = *resp.SecurityGroups[0]
			return nil
		}

		return fmt.Errorf("Security Group not found")
	}
}

func testAccCheckAWSSecurityGroupRuleAttributes(n string, group *ec2.SecurityGroup, p *ec2.IpPermission, ruleType string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Security Group Rule Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Security Group Rule is set")
		}

		if p == nil {
			p = &ec2.IpPermission{
				FromPort:   aws.Int64(80),
				ToPort:     aws.Int64(8000),
				IpProtocol: aws.String("tcp"),
				IpRanges:   []*ec2.IpRange{{CidrIp: aws.String("10.0.0.0/8")}},
			}
		}

		var matchingRule *ec2.IpPermission
		var rules []*ec2.IpPermission
		if ruleType == "ingress" {
			rules = group.IpPermissions
		} else {
			rules = group.IpPermissionsEgress
		}

		if len(rules) == 0 {
			return fmt.Errorf("No IPPerms")
		}

		for _, r := range rules {
			if r.ToPort != nil && *p.ToPort != *r.ToPort {
				continue
			}

			if r.FromPort != nil && *p.FromPort != *r.FromPort {
				continue
			}

			if r.IpProtocol != nil && *p.IpProtocol != *r.IpProtocol {
				continue
			}

			remaining := len(p.IpRanges)
			for _, ip := range p.IpRanges {
				for _, rip := range r.IpRanges {
					if *ip.CidrIp == *rip.CidrIp {
						remaining--
					}
				}
			}

			if remaining > 0 {
				continue
			}

			remaining = len(p.UserIdGroupPairs)
			for _, ip := range p.UserIdGroupPairs {
				for _, rip := range r.UserIdGroupPairs {
					if *ip.GroupId == *rip.GroupId {
						remaining--
					}
				}
			}

			if remaining > 0 {
				continue
			}

			remaining = len(p.PrefixListIds)
			for _, pip := range p.PrefixListIds {
				for _, rpip := range r.PrefixListIds {
					if *pip.PrefixListId == *rpip.PrefixListId {
						remaining--
					}
				}
			}

			if remaining > 0 {
				continue
			}

			matchingRule = r
		}

		if matchingRule != nil {
			log.Printf("[DEBUG] Matching rule found : %s", matchingRule)
			return nil
		}

		return fmt.Errorf("Error here\n\tlooking for %s, wasn't found in %s", p, rules)
	}
}

func testAccAWSSecurityGroupRuleIngressConfig(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_security_group" "web" {
		name = "terraform_test_%d"
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
	}`, rInt)
}

const testAccAWSSecurityGroupRuleIngress_ipv6Config = `
resource "aws_vpc" "tftest" {
  cidr_block = "10.0.0.0/16"

  tags {
    Name = "tf-testing"
  }
}

resource "aws_security_group" "web" {
  vpc_id = "${aws_vpc.tftest.id}"

  tags {
    Name = "tf-acc-test"
  }
}

resource "aws_security_group_rule" "ingress_1" {
  type        = "ingress"
  protocol    = "6"
  from_port   = 80
  to_port     = 8000
  ipv6_cidr_blocks = ["::/0"]

  security_group_id = "${aws_security_group.web.id}"
}
`

const testAccAWSSecurityGroupRuleIngress_protocolConfig = `
resource "aws_vpc" "tftest" {
  cidr_block = "10.0.0.0/16"

  tags {
    Name = "tf-testing"
  }
}

resource "aws_security_group" "web" {
  vpc_id = "${aws_vpc.tftest.id}"

  tags {
    Name = "tf-acc-test"
  }
}

resource "aws_security_group_rule" "ingress_1" {
  type        = "ingress"
  protocol    = "6"
  from_port   = 80
  to_port     = 8000
  cidr_blocks = ["10.0.0.0/8"]

  security_group_id = "${aws_security_group.web.id}"
}

`

const testAccAWSSecurityGroupRuleIssue5310 = `
provider "aws" {
        region = "us-east-1"
}

resource "aws_security_group" "issue_5310" {
    name = "terraform-test-issue_5310"
    description = "SG for test of issue 5310"
}

resource "aws_security_group_rule" "issue_5310" {
    type = "ingress"
    from_port = 0
    to_port = 65535
    protocol = "tcp"
    security_group_id = "${aws_security_group.issue_5310.id}"
    self = true
}
`

func testAccAWSSecurityGroupRuleIngressClassicConfig(rInt int) string {
	return fmt.Sprintf(`
	provider "aws" {
					region = "us-east-1"
	}

	resource "aws_security_group" "web" {
		name = "terraform_test_%d"
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
	}`, rInt)
}

func testAccAWSSecurityGroupRuleEgressConfig(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_security_group" "web" {
		name = "terraform_test_%d"
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
	}`, rInt)
}

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

func testAccAWSSecurityGroupRulePartialMatching(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_vpc" "default" {
		cidr_block = "10.0.0.0/16"
		tags {
			Name = "tf-sg-rule-bug"
		}
	}

	resource "aws_security_group" "web" {
			name = "tf-other-%d"
			vpc_id = "${aws_vpc.default.id}"
			tags {
					Name        = "tf-other-sg"
			}
	}

	resource "aws_security_group" "nat" {
			name = "tf-nat-%d"
			vpc_id = "${aws_vpc.default.id}"
			tags {
					Name        = "tf-nat-sg"
			}
	}

	resource "aws_security_group_rule" "ingress" {
			type        = "ingress"
			from_port   = 80
			to_port     = 80
			protocol    = "tcp"
			cidr_blocks = ["10.0.2.0/24", "10.0.3.0/24", "10.0.4.0/24"]

		 security_group_id = "${aws_security_group.web.id}"
	}

	resource "aws_security_group_rule" "other" {
			type        = "ingress"
			from_port   = 80
			to_port     = 80
			protocol    = "tcp"
			cidr_blocks = ["10.0.5.0/24"]

		 security_group_id = "${aws_security_group.web.id}"
	}

	// same a above, but different group, to guard against bad hashing
	resource "aws_security_group_rule" "nat_ingress" {
			type        = "ingress"
			from_port   = 80
			to_port     = 80
			protocol    = "tcp"
			cidr_blocks = ["10.0.2.0/24", "10.0.3.0/24", "10.0.4.0/24"]

		 security_group_id = "${aws_security_group.nat.id}"
	}`, rInt, rInt)
}

func testAccAWSSecurityGroupRulePartialMatching_Source(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_vpc" "default" {
		cidr_block = "10.0.0.0/16"
		tags {
			Name = "tf-sg-rule-bug"
		}
	}

	resource "aws_security_group" "web" {
			name = "tf-other-%d"
			vpc_id = "${aws_vpc.default.id}"
			tags {
					Name        = "tf-other-sg"
			}
	}

	resource "aws_security_group" "nat" {
			name = "tf-nat-%d"
			vpc_id = "${aws_vpc.default.id}"
			tags {
					Name        = "tf-nat-sg"
			}
	}

	resource "aws_security_group_rule" "source_ingress" {
			type        = "ingress"
			from_port   = 80
			to_port     = 80
			protocol    = "tcp"

									source_security_group_id = "${aws_security_group.nat.id}"
		 security_group_id = "${aws_security_group.web.id}"
	}

	resource "aws_security_group_rule" "other_ingress" {
			type        = "ingress"
			from_port   = 80
			to_port     = 80
			protocol    = "tcp"
			cidr_blocks = ["10.0.2.0/24", "10.0.3.0/24", "10.0.4.0/24"]

		 security_group_id = "${aws_security_group.web.id}"
	}`, rInt, rInt)
}

var testAccAWSSecurityGroupRuleRace = func() string {
	var b bytes.Buffer
	iterations := 50
	b.WriteString(fmt.Sprintf(`
		resource "aws_vpc" "default" {
			cidr_block = "10.0.0.0/16"
			tags { Name = "tf-sg-rule-race" }
		}

		resource "aws_security_group" "race" {
			name   = "tf-sg-rule-race-group-%d"
			vpc_id = "${aws_vpc.default.id}"
		}
	`, acctest.RandInt()))
	for i := 1; i < iterations; i++ {
		b.WriteString(fmt.Sprintf(`
			resource "aws_security_group_rule" "ingress%d" {
				security_group_id = "${aws_security_group.race.id}"
				type              = "ingress"
				from_port         = %d
				to_port           = %d
				protocol          = "tcp"
				cidr_blocks       = ["10.0.0.%d/32"]
			}

			resource "aws_security_group_rule" "egress%d" {
				security_group_id = "${aws_security_group.race.id}"
				type              = "egress"
				from_port         = %d
				to_port           = %d
				protocol          = "tcp"
				cidr_blocks       = ["10.0.0.%d/32"]
			}
		`, i, i, i, i, i, i, i, i))
	}
	return b.String()
}()

const testAccAWSSecurityGroupRulePrefixListEgressConfig = `

resource "aws_vpc" "tf_sg_prefix_list_egress_test" {
    cidr_block = "10.0.0.0/16"
    tags {
            Name = "tf_sg_prefix_list_egress_test"
    }
}

resource "aws_route_table" "default" {
    vpc_id = "${aws_vpc.tf_sg_prefix_list_egress_test.id}"
}

resource "aws_vpc_endpoint" "s3-us-west-2" {
  	vpc_id = "${aws_vpc.tf_sg_prefix_list_egress_test.id}"
  	service_name = "com.amazonaws.us-west-2.s3"
  	route_table_ids = ["${aws_route_table.default.id}"]
  	policy = <<POLICY
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid":"AllowAll",
			"Effect":"Allow",
			"Principal":"*",
			"Action":"*",
			"Resource":"*"
		}
	]
}
POLICY
}

resource "aws_security_group" "egress" {
    name = "terraform_acceptance_test_prefix_list_egress"
    description = "Used in the terraform acceptance tests"
    vpc_id = "${aws_vpc.tf_sg_prefix_list_egress_test.id}"
}

resource "aws_security_group_rule" "egress_1" {
  type = "egress"
  protocol = "-1"
  from_port = 0
  to_port = 0
  prefix_list_ids = ["${aws_vpc_endpoint.s3-us-west-2.prefix_list_id}"]
	security_group_id = "${aws_security_group.egress.id}"
}
`

func testAccAWSSecurityGroupRuleSelfInSource(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_vpc" "foo" {
		cidr_block = "10.1.0.0/16"

		tags {
			Name = "tf_sg_rule_self_group"
		}
	}

	resource "aws_security_group" "web" {
		name        = "allow_all-%d"
		description = "Allow all inbound traffic"
		vpc_id      = "${aws_vpc.foo.id}"
	}

	resource "aws_security_group_rule" "allow_self" {
		type                     = "ingress"
		from_port                = 0
		to_port                  = 0
		protocol                 = "-1"
		security_group_id        = "${aws_security_group.web.id}"
		source_security_group_id = "${aws_security_group.web.id}"
	}`, rInt)
}

func testAccAWSSecurityGroupRuleExpectInvalidType(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_vpc" "foo" {
		cidr_block = "10.1.0.0/16"

		tags {
			Name = "tf_sg_rule_self_group"
		}
	}

	resource "aws_security_group" "web" {
		name        = "allow_all-%d"
		description = "Allow all inbound traffic"
		vpc_id      = "${aws_vpc.foo.id}"
	}

	resource "aws_security_group_rule" "allow_self" {
		type                     = "foobar"
		from_port                = 0
		to_port                  = 0
		protocol                 = "-1"
		security_group_id        = "${aws_security_group.web.id}"
		source_security_group_id = "${aws_security_group.web.id}"
	}`, rInt)
}

func testAccAWSSecurityGroupRuleInvalidIPv4CIDR(rInt int) string {
	return fmt.Sprintf(`
resource "aws_security_group" "foo" {
  name = "testing-failure-%d"
}

resource "aws_security_group_rule" "ing" {
  type = "ingress"
  from_port = 0
  to_port = 0
  protocol = "-1"
  cidr_blocks = ["1.2.3.4/33"]
  security_group_id = "${aws_security_group.foo.id}"
}`, rInt)
}

func testAccAWSSecurityGroupRuleInvalidIPv6CIDR(rInt int) string {
	return fmt.Sprintf(`
resource "aws_security_group" "foo" {
  name = "testing-failure-%d"
}

resource "aws_security_group_rule" "ing" {
  type = "egress"
  from_port = 0
  to_port = 0
  protocol = "-1"
  ipv6_cidr_blocks = ["::/244"]
  security_group_id = "${aws_security_group.foo.id}"
}`, rInt)
}
