package aws

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSecurityGroupRules_basic(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group_rules.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.web", &group),
					testAccCheckAWSSecurityGroupRulesAttributes(&group),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.cidr_blocks.0", "10.0.0.0/8"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRules_ipv6(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group_rules.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConfigIpv6,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.web", &group),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.2293451516.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.2293451516.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.2293451516.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.2293451516.ipv6_cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.2293451516.ipv6_cidr_blocks.0", "::/0"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRules_self(t *testing.T) {
	var group ec2.SecurityGroup

	checkSelf := func(s *terraform.State) (err error) {
		defer func() {
			if e := recover(); e != nil {
				err = fmt.Errorf("bad: %#v", group)
			}
		}()

		if *group.IpPermissions[0].UserIdGroupPairs[0].GroupId != *group.GroupId {
			return fmt.Errorf("bad: %#v", group)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group_rules.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConfigSelf,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.web", &group),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3971148406.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3971148406.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3971148406.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3971148406.self", "true"),
					checkSelf,
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRules_vpc(t *testing.T) {
	var group ec2.SecurityGroup

	testCheck := func(*terraform.State) error {
		if *group.VpcId == "" {
			return fmt.Errorf("should have vpc ID")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group_rules.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConfigVpc,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.web", &group),
					testAccCheckAWSSecurityGroupRulesAttributes(&group),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.cidr_blocks.0", "10.0.0.0/8"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "egress.3629188364.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "egress.3629188364.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "egress.3629188364.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "egress.3629188364.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "egress.3629188364.cidr_blocks.0", "10.0.0.0/8"),
					testCheck,
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRules_vpcNegOneIngress(t *testing.T) {
	var group ec2.SecurityGroup

	testCheck := func(*terraform.State) error {
		if *group.VpcId == "" {
			return fmt.Errorf("should have vpc ID")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group_rules.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConfigVpcNegOneIngress,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.web", &group),
					testAccCheckAWSSecurityGroupRulesAttributesNegOneProtocol(&group),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.956249133.protocol", "-1"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.956249133.from_port", "0"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.956249133.to_port", "0"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.956249133.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.956249133.cidr_blocks.0", "10.0.0.0/8"),
					testCheck,
				),
			},
		},
	})
}
func TestAccAWSSecurityGroupRules_vpcProtoNumIngress(t *testing.T) {
	var group ec2.SecurityGroup

	testCheck := func(*terraform.State) error {
		if *group.VpcId == "" {
			return fmt.Errorf("should have vpc ID")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group_rules.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConfigVpcProtoNumIngress,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.web", &group),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.2449525218.protocol", "50"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.2449525218.from_port", "0"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.2449525218.to_port", "0"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.2449525218.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.2449525218.cidr_blocks.0", "10.0.0.0/8"),
					testCheck,
				),
			},
		},
	})
}
func TestAccAWSSecurityGroupRules_MultiIngress(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group_rules.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConfigMultiIngress,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.web", &group),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRules_Change(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group_rules.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.web", &group),
				),
			},
			{
				Config: testAccAWSSecurityGroupRulesConfigChange,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.web", &group),
					testAccCheckAWSSecurityGroupRulesAttributesChanged(&group),
				),
			},
		},
	})
}

// Testing drift detection with groups containing the same port and types
func TestAccAWSSecurityGroupRules_drift(t *testing.T) {
	var group ec2.SecurityGroup
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConfig_drift(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.web", &group),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.cidr_blocks.0", "10.0.0.0/8"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRules_drift_complex(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConfig_drift_complex(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.web", &group),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.3629188364.cidr_blocks.0", "10.0.0.0/8"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRules_invalidCIDRBlock(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAWSSecurityGroupRulesInvalidIngressCidr,
				ExpectError: regexp.MustCompile("invalid CIDR address: 1.2.3.4/33"),
			},
			{
				Config:      testAccAWSSecurityGroupRulesInvalidEgressCidr,
				ExpectError: regexp.MustCompile("invalid CIDR address: 1.2.3.4/33"),
			},
			{
				Config:      testAccAWSSecurityGroupRulesInvalidIPv6IngressCidr,
				ExpectError: regexp.MustCompile("invalid CIDR address: ::/244"),
			},
			{
				Config:      testAccAWSSecurityGroupRulesInvalidIPv6EgressCidr,
				ExpectError: regexp.MustCompile("invalid CIDR address: ::/244"),
			},
		},
	})
}

func testAccCheckAWSSecurityGroupRulesEmpty(n string, group *ec2.SecurityGroup) resource.TestCheckFunc {
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

			if len(group.IpPermissions) != 0 {
				return fmt.Errorf("Ingress rules are not empty")
			}

			if len(group.IpPermissionsEgress) != 0 {
				return fmt.Errorf("Egress rules are not empty")
			}

			return nil
		}

		return fmt.Errorf("Security Group not found")
	}
}

func testAccCheckAWSSecurityGroupRulesAttributes(group *ec2.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		p := &ec2.IpPermission{
			FromPort:   aws.Int64(80),
			ToPort:     aws.Int64(8000),
			IpProtocol: aws.String("tcp"),
			IpRanges:   []*ec2.IpRange{{CidrIp: aws.String("10.0.0.0/8")}},
		}

		if len(group.IpPermissions) == 0 {
			return fmt.Errorf("No IPPerms")
		}

		// Compare our ingress
		if !reflect.DeepEqual(group.IpPermissions[0], p) {
			return fmt.Errorf(
				"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
				group.IpPermissions[0],
				p)
		}

		return nil
	}
}

func testAccCheckAWSSecurityGroupRulesAttributesNegOneProtocol(group *ec2.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		p := &ec2.IpPermission{
			IpProtocol: aws.String("-1"),
			IpRanges:   []*ec2.IpRange{{CidrIp: aws.String("10.0.0.0/8")}},
		}

		if len(group.IpPermissions) == 0 {
			return fmt.Errorf("No IPPerms")
		}

		// Compare our ingress
		if !reflect.DeepEqual(group.IpPermissions[0], p) {
			return fmt.Errorf(
				"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
				group.IpPermissions[0],
				p)
		}

		return nil
	}
}

func TestAccAWSSecurityGroupRules_CIDRandGroups(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesCombindCIDRandGroups,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.mixed", &group),
					// testAccCheckAWSSecurityGroupRulesAttributes(&group),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRules_ingressWithCidrAndSGs(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConfig_ingressWithCidrAndSGs,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.web", &group),
					testAccCheckAWSSecurityGroupRulesSGandCidrAttributes(&group),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.#", "2"),
				),
			},
		},
	})
}

// This test requires an EC2 Classic region
func TestAccAWSSecurityGroupRules_ingressWithCidrAndSGs_classic(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConfig_ingressWithCidrAndSGs_classic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.web", &group),
					testAccCheckAWSSecurityGroupRulesSGandCidrAttributes(&group),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.web", "ingress.#", "2"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRules_egressWithPrefixList(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConfigPrefixListEgress,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.egress", &group),
					testAccCheckAWSSecurityGroupRulesPrefixListAttributes(&group),
					resource.TestCheckResourceAttr(
						"aws_security_group_rules.egress", "egress.#", "1"),
				),
			},
		},
	})
}

func testAccCheckAWSSecurityGroupRulesSGandCidrAttributes(group *ec2.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(group.IpPermissions) == 0 {
			return fmt.Errorf("No IPPerms")
		}

		if len(group.IpPermissions) != 2 {
			return fmt.Errorf("Expected 2 ingress rules, got %d", len(group.IpPermissions))
		}

		for _, p := range group.IpPermissions {
			if *p.FromPort == int64(22) {
				if len(p.IpRanges) != 1 || p.UserIdGroupPairs != nil {
					return fmt.Errorf("Found ip perm of 22, but not the right ipranges / pairs: %s", p)
				}
				continue
			} else if *p.FromPort == int64(80) {
				if len(p.IpRanges) != 1 || len(p.UserIdGroupPairs) != 1 {
					return fmt.Errorf("Found ip perm of 80, but not the right ipranges / pairs: %s", p)
				}
				continue
			}
			return fmt.Errorf("Found a rouge rule")
		}

		return nil
	}
}

func testAccCheckAWSSecurityGroupRulesPrefixListAttributes(group *ec2.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(group.IpPermissionsEgress) == 0 {
			return fmt.Errorf("No egress IPPerms")
		}
		if len(group.IpPermissionsEgress) != 1 {
			return fmt.Errorf("Expected 1 egress rule, got %d", len(group.IpPermissions))
		}

		p := group.IpPermissionsEgress[0]

		if len(p.PrefixListIds) != 1 {
			return fmt.Errorf("Expected 1 prefix list, got %d", len(p.PrefixListIds))
		}

		return nil
	}
}

func testAccCheckAWSSecurityGroupRulesAttributesChanged(group *ec2.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		p := []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(80),
				ToPort:     aws.Int64(9000),
				IpProtocol: aws.String("tcp"),
				IpRanges:   []*ec2.IpRange{{CidrIp: aws.String("10.0.0.0/8")}},
			},
			{
				FromPort:   aws.Int64(80),
				ToPort:     aws.Int64(8000),
				IpProtocol: aws.String("tcp"),
				IpRanges: []*ec2.IpRange{
					{
						CidrIp: aws.String("0.0.0.0/0"),
					},
					{
						CidrIp: aws.String("10.0.0.0/8"),
					},
				},
			},
		}

		// Compare our ingress
		if len(group.IpPermissions) != 2 {
			return fmt.Errorf(
				"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
				group.IpPermissions,
				p)
		}

		if *group.IpPermissions[0].ToPort == 8000 {
			group.IpPermissions[1], group.IpPermissions[0] =
				group.IpPermissions[0], group.IpPermissions[1]
		}

		if !reflect.DeepEqual(group.IpPermissions, p) {
			return fmt.Errorf(
				"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
				group.IpPermissions,
				p)
		}

		return nil
	}
}

func TestAccAWSSecurityGroupRules_failWithDiffMismatch(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConfig_failWithDiffMismatch,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.nat", &group),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRules_Destroy(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group_rules.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
				),
			},
			{
				Config: testAccAWSSecurityGroupRulesDestroyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRulesEmpty("aws_security_group.web", &group),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRules_Empty(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group_rules.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.web", &group),
				),
			},
			{
				Config: testAccAWSSecurityGroupRulesEmptyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRulesEmpty("aws_security_group_rules.web", &group),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroupRules_ConflictEmpty(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group_rules.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupRulesConflictEmptyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group_rules.web", &group),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccAWSSecurityGroupRulesConflictEmptyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupRulesEmpty("aws_security_group_rules.web", &group),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

const testAccAWSSecurityGroupRulesConfigIpv6 = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"

	tags {
		Name = "tf-acc-test"
	}
}

resource "aws_security_group_rules" "web" {
  security_group_id = "${aws_security_group.web.id}"

  ingress {
    protocol = "6"
    from_port = 80
    to_port = 8000
    ipv6_cidr_blocks = ["::/0"]
  }

  egress {
    protocol = "tcp"
    from_port = 80
    to_port = 8000
    ipv6_cidr_blocks = ["::/0"]
  }
}
`

const testAccAWSSecurityGroupRulesConfig = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"

  tags {
    Name = "tf-acc-test"
  }
}

resource "aws_security_group_rules" "web" {
  security_group_id = "${aws_security_group.web.id}"

  ingress {
    protocol = "6"
    from_port = 80
    to_port = 8000
    cidr_blocks = ["10.0.0.0/8"]
  }

  egress {
    protocol = "tcp"
    from_port = 80
    to_port = 8000
    cidr_blocks = ["10.0.0.0/8"]
  }
}
`

const testAccAWSSecurityGroupRulesDestroyConfig = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"

  tags {
    Name = "tf-acc-test"
  }
}
`

const testAccAWSSecurityGroupRulesEmptyConfig = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"

  tags {
    Name = "tf-acc-test"
  }
}

resource "aws_security_group_rules" "web" {
  security_group_id = "${aws_security_group.web.id}"
}
`

const testAccAWSSecurityGroupRulesConflictEmptyConfig = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"

  ingress {
    protocol = "6"
    from_port = 80
    to_port = 8000
    cidr_blocks = ["10.0.0.0/8"]
  }

  egress {
    protocol = "tcp"
    from_port = 80
    to_port = 8000
    cidr_blocks = ["10.0.0.0/8"]
  }

  tags {
    Name = "tf-acc-test"
  }
}

resource "aws_security_group_rules" "web" {
  security_group_id = "${aws_security_group.web.id}"
}
`

const testAccAWSSecurityGroupRulesConfigChange = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"

  tags {
    Name = "tf-acc-test"
  }
}

resource "aws_security_group_rules" "web" {
  security_group_id = "${aws_security_group.web.id}"

  ingress {
    protocol = "tcp"
    from_port = 80
    to_port = 9000
    cidr_blocks = ["10.0.0.0/8"]
  }

  ingress {
    protocol = "tcp"
    from_port = 80
    to_port = 8000
    cidr_blocks = ["0.0.0.0/0", "10.0.0.0/8"]
  }

  egress {
    protocol = "tcp"
    from_port = 80
    to_port = 8000
    cidr_blocks = ["10.0.0.0/8"]
  }
}
`

const testAccAWSSecurityGroupRulesConfigSelf = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_security_group_rules" "web" {
  security_group_id = "${aws_security_group.web.id}"

  ingress {
    protocol = "tcp"
    from_port = 80
    to_port = 8000
    self = true
  }

  egress {
    protocol = "tcp"
    from_port = 80
    to_port = 8000
    cidr_blocks = ["10.0.0.0/8"]
  }
}
`

const testAccAWSSecurityGroupRulesConfigVpc = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_security_group_rules" "web" {
  security_group_id = "${aws_security_group.web.id}"

  ingress {
    protocol = "tcp"
    from_port = 80
    to_port = 8000
    cidr_blocks = ["10.0.0.0/8"]
  }

	egress {
		protocol = "tcp"
		from_port = 80
		to_port = 8000
		cidr_blocks = ["10.0.0.0/8"]
	}
}
`

const testAccAWSSecurityGroupRulesConfigVpcNegOneIngress = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "web" {
	name = "terraform_acceptance_test_example"
	description = "Used in the terraform acceptance tests"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_security_group_rules" "web" {
  security_group_id = "${aws_security_group.web.id}"

	ingress {
		protocol = "-1"
		from_port = 0
		to_port = 0
		cidr_blocks = ["10.0.0.0/8"]
	}
}
`

const testAccAWSSecurityGroupRulesConfigVpcProtoNumIngress = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "web" {
	name = "terraform_acceptance_test_example"
	description = "Used in the terraform acceptance tests"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_security_group_rules" "web" {
  security_group_id = "${aws_security_group.web.id}"

	ingress {
		protocol = "50"
		from_port = 0
		to_port = 0
		cidr_blocks = ["10.0.0.0/8"]
	}
}
`

const testAccAWSSecurityGroupRulesConfigMultiIngress = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "worker" {
  name = "terraform_acceptance_test_example_1"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_security_group_rules" "worker" {
  security_group_id = "${aws_security_group.worker.id}"

  ingress {
    protocol = "tcp"
    from_port = 80
    to_port = 8000
    cidr_blocks = ["10.0.0.0/8"]
  }

  egress {
    protocol = "tcp"
    from_port = 80
    to_port = 8000
    cidr_blocks = ["10.0.0.0/8"]
  }
}

resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example_2"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_security_group_rules" "web" {
  security_group_id = "${aws_security_group.web.id}"

  ingress {
    protocol = "tcp"
    from_port = 22
    to_port = 22
    cidr_blocks = ["10.0.0.0/8"]
  }

  ingress {
    protocol = "tcp"
    from_port = 800
    to_port = 800
    cidr_blocks = ["10.0.0.0/8"]
  }

  ingress {
    protocol = "tcp"
    from_port = 80
    to_port = 8000
    security_groups = ["${aws_security_group.worker.id}"]
  }

  egress {
    protocol = "tcp"
    from_port = 80
    to_port = 8000
    cidr_blocks = ["10.0.0.0/8"]
  }
}
`

func testAccAWSSecurityGroupRulesConfig_drift() string {
	return fmt.Sprintf(`
resource "aws_security_group" "web" {
  name = "tf_acc_%d"
  description = "Used in the terraform acceptance tests"

        tags {
                Name = "tf-acc-test"
        }
}

resource "aws_security_group_rules" "web" {
  security_group_id = "${aws_security_group.web.id}"

  ingress {
    protocol = "tcp"
    from_port = 80
    to_port = 8000
    cidr_blocks = ["10.0.0.0/8"]
  }

  ingress {
    protocol = "tcp"
    from_port = 80
    to_port = 8000
    cidr_blocks = ["206.0.0.0/8"]
  }
}
`, acctest.RandInt())
}

func testAccAWSSecurityGroupRulesConfig_drift_complex() string {
	return fmt.Sprintf(`
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "otherweb" {
  name        = "tf_acc_%d"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_security_group" "web" {
  name        = "tf_acc_%d"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"

  tags {
    Name = "tf-acc-test"
  }
}

resource "aws_security_group_rules" "web" {
  security_group_id = "${aws_security_group.web.id}"

  ingress {
    protocol    = "tcp"
    from_port   = 80
    to_port     = 8000
    cidr_blocks = ["10.0.0.0/8"]
  }

  ingress {
    protocol    = "tcp"
    from_port   = 80
    to_port     = 8000
    cidr_blocks = ["206.0.0.0/8"]
  }

  ingress {
    protocol        = "tcp"
    from_port       = 22
    to_port         = 22
    security_groups = ["${aws_security_group.otherweb.id}"]
  }

  egress {
    protocol    = "tcp"
    from_port   = 80
    to_port     = 8000
    cidr_blocks = ["206.0.0.0/8"]
  }

  egress {
    protocol    = "tcp"
    from_port   = 80
    to_port     = 8000
    cidr_blocks = ["10.0.0.0/8"]
  }

  egress {
    protocol        = "tcp"
    from_port       = 22
    to_port         = 22
    security_groups = ["${aws_security_group.otherweb.id}"]
  }
}`, acctest.RandInt(), acctest.RandInt())
}

const testAccAWSSecurityGroupRulesInvalidIngressCidr = `
resource "aws_security_group" "foo" {
  name = "testing-foo"
  description = "foo-testing"
}

resource "aws_security_group_rules" "foo" {
  security_group_id = "${aws_security_group.foo.id}"

  ingress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["1.2.3.4/33"]
  }
}`

const testAccAWSSecurityGroupRulesInvalidEgressCidr = `
resource "aws_security_group" "foo" {
  name = "testing-foo"
  description = "foo-testing"
}

resource "aws_security_group_rules" "foo" {
  security_group_id = "${aws_security_group.foo.id}"

  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["1.2.3.4/33"]
  }
}`

const testAccAWSSecurityGroupRulesInvalidIPv6IngressCidr = `
resource "aws_security_group" "foo" {
  name = "testing-foo"
  description = "foo-testing"
}

resource "aws_security_group_rules" "foo" {
  security_group_id = "${aws_security_group.foo.id}"

  ingress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    ipv6_cidr_blocks = ["::/244"]
  }
}`

const testAccAWSSecurityGroupRulesInvalidIPv6EgressCidr = `
resource "aws_security_group" "foo" {
  name = "testing-foo"
  description = "foo-testing"
}

resource "aws_security_group_rules" "foo" {
  security_group_id = "${aws_security_group.foo.id}"

  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    ipv6_cidr_blocks = ["::/244"]
  }
}`

const testAccAWSSecurityGroupRulesCombindCIDRandGroups = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "two" {
	name = "tf-test-1"
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		Name = "tf-test-1"
	}
}

resource "aws_security_group" "one" {
	name = "tf-test-2"
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		Name = "tf-test-w"
	}
}

resource "aws_security_group" "three" {
	name = "tf-test-3"
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		Name = "tf-test-3"
	}
}

resource "aws_security_group" "mixed" {
  name = "tf-mix-test"
  vpc_id = "${aws_vpc.foo.id}"

  tags {
    Name = "tf-mix-test"
  }
}

resource "aws_security_group_rules" "mixed" {
  security_group_id = "${aws_security_group.mixed.id}"

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/16", "10.1.0.0/16", "10.7.0.0/16"]

    security_groups = [
      "${aws_security_group.one.id}",
      "${aws_security_group.two.id}",
      "${aws_security_group.three.id}",
    ]
  }
}
`

const testAccAWSSecurityGroupRulesConfig_ingressWithCidrAndSGs = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "other_web" {
  name        = "tf_other_acc_tests"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"

  tags {
    Name = "tf-acc-test"
  }
}

resource "aws_security_group" "web" {
  name        = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"

  tags {
    Name = "tf-acc-test"
  }
}

resource "aws_security_group_rules" "web" {
  security_group_id = "${aws_security_group.web.id}"

  ingress {
    protocol  = "tcp"
    from_port = "22"
    to_port   = "22"

    cidr_blocks = [
      "192.168.0.1/32",
    ]
  }

  ingress {
    protocol        = "tcp"
    from_port       = 80
    to_port         = 8000
    cidr_blocks     = ["10.0.0.0/8"]
    security_groups = ["${aws_security_group.other_web.id}"]
  }

  egress {
    protocol    = "tcp"
    from_port   = 80
    to_port     = 8000
    cidr_blocks = ["10.0.0.0/8"]
  }
}
`

const testAccAWSSecurityGroupRulesConfig_ingressWithCidrAndSGs_classic = `
provider "aws" {
	region = "us-east-1"
}

resource "aws_security_group" "other_web" {
  name        = "tf_other_acc_tests"
  description = "Used in the terraform acceptance tests"

  tags {
    Name = "tf-acc-test"
  }
}

resource "aws_security_group" "web" {
  name        = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"

  tags {
    Name = "tf-acc-test"
  }
}

resource "aws_security_group_rules" "web" {
  security_group_id = "${aws_security_group.web.id}"

  ingress {
    protocol  = "tcp"
    from_port = "22"
    to_port   = "22"

    cidr_blocks = [
      "192.168.0.1/32",
    ]
  }

  ingress {
    protocol        = "tcp"
    from_port       = 80
    to_port         = 8000
    cidr_blocks     = ["10.0.0.0/8"]
    security_groups = ["${aws_security_group.other_web.name}"]
  }
}
`

// fails to apply in one pass with the error "diffs didn't match during apply"
// GH-2027
const testAccAWSSecurityGroupRulesConfig_failWithDiffMismatch = `
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"

  tags {
    Name = "tf-test"
  }
}

resource "aws_security_group" "ssh_base" {
  name   = "test-ssh-base"
  vpc_id = "${aws_vpc.main.id}"
}

resource "aws_security_group" "jump" {
  name   = "test-jump"
  vpc_id = "${aws_vpc.main.id}"
}

resource "aws_security_group" "provision" {
  name   = "test-provision"
  vpc_id = "${aws_vpc.main.id}"
}

resource "aws_security_group" "nat" {
  vpc_id      = "${aws_vpc.main.id}"
  name        = "nat"
  description = "For nat servers "
}

resource "aws_security_group_rules" "nat" {
  security_group_id = "${aws_security_group.nat.id}"

  ingress {
    from_port       = 22
    to_port         = 22
    protocol        = "tcp"
    security_groups = ["${aws_security_group.jump.id}"]
  }

  ingress {
    from_port       = 22
    to_port         = 22
    protocol        = "tcp"
    security_groups = ["${aws_security_group.provision.id}"]
  }
}
`

const testAccAWSSecurityGroupRulesConfigPrefixListEgress = `
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

resource "aws_security_group_rules" "egress" {
  security_group_id = "${aws_security_group.egress.id}"

    egress {
      protocol = "-1"
      from_port = 0
      to_port = 0
      prefix_list_ids = ["${aws_vpc_endpoint.s3-us-west-2.prefix_list_id}"]
    }
}
`
