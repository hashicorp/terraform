package aws

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestProtocolStateFunc(t *testing.T) {
	cases := []struct {
		input    interface{}
		expected string
	}{
		{
			input:    "tcp",
			expected: "tcp",
		},
		{
			input:    6,
			expected: "",
		},
		{
			input:    "17",
			expected: "udp",
		},
		{
			input:    "all",
			expected: "-1",
		},
		{
			input:    "-1",
			expected: "-1",
		},
		{
			input:    -1,
			expected: "",
		},
		{
			input:    "1",
			expected: "icmp",
		},
		{
			input:    "icmp",
			expected: "icmp",
		},
		{
			input:    1,
			expected: "",
		},
	}
	for _, c := range cases {
		result := protocolStateFunc(c.input)
		if result != c.expected {
			t.Errorf("Error matching protocol, expected (%s), got (%s)", c.expected, result)
		}
	}
}

func TestProtocolForValue(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    "tcp",
			expected: "tcp",
		},
		{
			input:    "6",
			expected: "tcp",
		},
		{
			input:    "udp",
			expected: "udp",
		},
		{
			input:    "17",
			expected: "udp",
		},
		{
			input:    "all",
			expected: "-1",
		},
		{
			input:    "-1",
			expected: "-1",
		},
		{
			input:    "tCp",
			expected: "tcp",
		},
		{
			input:    "6",
			expected: "tcp",
		},
		{
			input:    "UDp",
			expected: "udp",
		},
		{
			input:    "17",
			expected: "udp",
		},
		{
			input:    "ALL",
			expected: "-1",
		},
		{
			input:    "icMp",
			expected: "icmp",
		},
		{
			input:    "1",
			expected: "icmp",
		},
	}

	for _, c := range cases {
		result := protocolForValue(c.input)
		if result != c.expected {
			t.Errorf("Error matching protocol, expected (%s), got (%s)", c.expected, result)
		}
	}
}

func TestResourceAwsSecurityGroupIPPermGather(t *testing.T) {
	raw := []*ec2.IpPermission{
		{
			IpProtocol: aws.String("tcp"),
			FromPort:   aws.Int64(int64(1)),
			ToPort:     aws.Int64(int64(-1)),
			IpRanges:   []*ec2.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
			UserIdGroupPairs: []*ec2.UserIdGroupPair{
				{
					GroupId: aws.String("sg-11111"),
				},
			},
		},
		{
			IpProtocol: aws.String("tcp"),
			FromPort:   aws.Int64(int64(80)),
			ToPort:     aws.Int64(int64(80)),
			UserIdGroupPairs: []*ec2.UserIdGroupPair{
				// VPC
				{
					GroupId: aws.String("sg-22222"),
				},
			},
		},
		{
			IpProtocol: aws.String("tcp"),
			FromPort:   aws.Int64(int64(443)),
			ToPort:     aws.Int64(int64(443)),
			UserIdGroupPairs: []*ec2.UserIdGroupPair{
				// Classic
				{
					UserId:    aws.String("12345"),
					GroupId:   aws.String("sg-33333"),
					GroupName: aws.String("ec2_classic"),
				},
				{
					UserId:    aws.String("amazon-elb"),
					GroupId:   aws.String("sg-d2c979d3"),
					GroupName: aws.String("amazon-elb-sg"),
				},
			},
		},
		{
			IpProtocol:    aws.String("-1"),
			FromPort:      aws.Int64(int64(0)),
			ToPort:        aws.Int64(int64(0)),
			PrefixListIds: []*ec2.PrefixListId{{PrefixListId: aws.String("pl-12345678")}},
			UserIdGroupPairs: []*ec2.UserIdGroupPair{
				// VPC
				{
					GroupId: aws.String("sg-22222"),
				},
			},
		},
	}

	local := []map[string]interface{}{
		{
			"protocol":    "tcp",
			"from_port":   int64(1),
			"to_port":     int64(-1),
			"cidr_blocks": []string{"0.0.0.0/0"},
			"self":        true,
		},
		{
			"protocol":  "tcp",
			"from_port": int64(80),
			"to_port":   int64(80),
			"security_groups": schema.NewSet(schema.HashString, []interface{}{
				"sg-22222",
			}),
		},
		{
			"protocol":  "tcp",
			"from_port": int64(443),
			"to_port":   int64(443),
			"security_groups": schema.NewSet(schema.HashString, []interface{}{
				"ec2_classic",
				"amazon-elb/amazon-elb-sg",
			}),
		},
		{
			"protocol":        "-1",
			"from_port":       int64(0),
			"to_port":         int64(0),
			"prefix_list_ids": []string{"pl-12345678"},
			"security_groups": schema.NewSet(schema.HashString, []interface{}{
				"sg-22222",
			}),
		},
	}

	out := resourceAwsSecurityGroupIPPermGather("sg-11111", raw, aws.String("12345"))
	for _, i := range out {
		// loop and match rules, because the ordering is not guarneteed
		for _, l := range local {
			if i["from_port"] == l["from_port"] {

				if i["to_port"] != l["to_port"] {
					t.Fatalf("to_port does not match")
				}

				if _, ok := i["cidr_blocks"]; ok {
					if !reflect.DeepEqual(i["cidr_blocks"], l["cidr_blocks"]) {
						t.Fatalf("error matching cidr_blocks")
					}
				}

				if _, ok := i["security_groups"]; ok {
					outSet := i["security_groups"].(*schema.Set)
					localSet := l["security_groups"].(*schema.Set)

					if !outSet.Equal(localSet) {
						t.Fatalf("Security Group sets are not equal")
					}
				}
			}
		}
	}
}

func TestAccAWSSecurityGroup_basic(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupAttributes(&group),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "name", "terraform_acceptance_test_example"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "description", "Used in the terraform acceptance tests"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.cidr_blocks.0", "10.0.0.0/8"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_ipv6(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfigIpv6,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "name", "terraform_acceptance_test_example"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "description", "Used in the terraform acceptance tests"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.2293451516.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.2293451516.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.2293451516.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.2293451516.ipv6_cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.2293451516.ipv6_cidr_blocks.0", "::/0"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_tagsCreatedFirst(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAWSSecurityGroupConfigForTagsOrdering,
				ExpectError: regexp.MustCompile("InvalidParameterValue"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.foo", &group),
					testAccCheckTags(&group.Tags, "Name", "tf-acc-test"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_namePrefix(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:        func() { testAccPreCheck(t) },
		IDRefreshName:   "aws_security_group.baz",
		IDRefreshIgnore: []string{"name_prefix"},
		Providers:       testAccProviders,
		CheckDestroy:    testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupPrefixNameConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.baz", &group),
					testAccCheckAWSSecurityGroupGeneratedNamePrefix(
						"aws_security_group.baz", "baz-"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_self(t *testing.T) {
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
		IDRefreshName: "aws_security_group.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfigSelf,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "name", "terraform_acceptance_test_example"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "description", "Used in the terraform acceptance tests"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3971148406.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3971148406.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3971148406.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3971148406.self", "true"),
					checkSelf,
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_vpc(t *testing.T) {
	var group ec2.SecurityGroup

	testCheck := func(*terraform.State) error {
		if *group.VpcId == "" {
			return fmt.Errorf("should have vpc ID")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfigVpc,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupAttributes(&group),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "name", "terraform_acceptance_test_example"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "description", "Used in the terraform acceptance tests"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.cidr_blocks.0", "10.0.0.0/8"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "egress.3629188364.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "egress.3629188364.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "egress.3629188364.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "egress.3629188364.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "egress.3629188364.cidr_blocks.0", "10.0.0.0/8"),
					testCheck,
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_vpcNegOneIngress(t *testing.T) {
	var group ec2.SecurityGroup

	testCheck := func(*terraform.State) error {
		if *group.VpcId == "" {
			return fmt.Errorf("should have vpc ID")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfigVpcNegOneIngress,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupAttributesNegOneProtocol(&group),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "name", "terraform_acceptance_test_example"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "description", "Used in the terraform acceptance tests"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.956249133.protocol", "-1"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.956249133.from_port", "0"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.956249133.to_port", "0"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.956249133.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.956249133.cidr_blocks.0", "10.0.0.0/8"),
					testCheck,
				),
			},
		},
	})
}
func TestAccAWSSecurityGroup_vpcProtoNumIngress(t *testing.T) {
	var group ec2.SecurityGroup

	testCheck := func(*terraform.State) error {
		if *group.VpcId == "" {
			return fmt.Errorf("should have vpc ID")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfigVpcProtoNumIngress,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "name", "terraform_acceptance_test_example"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "description", "Used in the terraform acceptance tests"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.2449525218.protocol", "50"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.2449525218.from_port", "0"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.2449525218.to_port", "0"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.2449525218.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.2449525218.cidr_blocks.0", "10.0.0.0/8"),
					testCheck,
				),
			},
		},
	})
}
func TestAccAWSSecurityGroup_MultiIngress(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfigMultiIngress,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_Change(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
				),
			},
			{
				Config: testAccAWSSecurityGroupConfigChange,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupAttributesChanged(&group),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_generatedName(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfig_generatedName,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "description", "Managed by Terraform"),
					func(s *terraform.State) error {
						if group.GroupName == nil {
							return fmt.Errorf("bad: No SG name")
						}
						if !strings.HasPrefix(*group.GroupName, "terraform-") {
							return fmt.Errorf("No terraform- prefix: %s", *group.GroupName)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_DefaultEgress_VPC(t *testing.T) {

	// VPC
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group.worker",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfigDefaultEgress,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExistsWithoutDefault("aws_security_group.worker"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_DefaultEgress_Classic(t *testing.T) {

	// Classic
	var group ec2.SecurityGroup
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_security_group.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfigClassic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
				),
			},
		},
	})
}

// Testing drift detection with groups containing the same port and types
func TestAccAWSSecurityGroup_drift(t *testing.T) {
	var group ec2.SecurityGroup
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfig_drift(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "description", "Used in the terraform acceptance tests"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.cidr_blocks.0", "10.0.0.0/8"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_drift_complex(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfig_drift_complex(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "description", "Used in the terraform acceptance tests"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.3629188364.cidr_blocks.0", "10.0.0.0/8"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_invalidCIDRBlock(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAWSSecurityGroupInvalidIngressCidr,
				ExpectError: regexp.MustCompile("invalid CIDR address: 1.2.3.4/33"),
			},
			{
				Config:      testAccAWSSecurityGroupInvalidEgressCidr,
				ExpectError: regexp.MustCompile("invalid CIDR address: 1.2.3.4/33"),
			},
			{
				Config:      testAccAWSSecurityGroupInvalidIPv6IngressCidr,
				ExpectError: regexp.MustCompile("invalid CIDR address: ::/244"),
			},
			{
				Config:      testAccAWSSecurityGroupInvalidIPv6EgressCidr,
				ExpectError: regexp.MustCompile("invalid CIDR address: ::/244"),
			},
		},
	})
}

func testAccCheckAWSSecurityGroupDestroy(s *terraform.State) error {
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

func testAccCheckAWSSecurityGroupGeneratedNamePrefix(
	resource, prefix string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		r, ok := s.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("Resource not found")
		}
		name, ok := r.Primary.Attributes["name"]
		if !ok {
			return fmt.Errorf("Name attr not found: %#v", r.Primary.Attributes)
		}
		if !strings.HasPrefix(name, prefix) {
			return fmt.Errorf("Name: %q, does not have prefix: %q", name, prefix)
		}
		return nil
	}
}

func testAccCheckAWSSecurityGroupExists(n string, group *ec2.SecurityGroup) resource.TestCheckFunc {
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

func testAccCheckAWSSecurityGroupAttributes(group *ec2.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		p := &ec2.IpPermission{
			FromPort:   aws.Int64(80),
			ToPort:     aws.Int64(8000),
			IpProtocol: aws.String("tcp"),
			IpRanges:   []*ec2.IpRange{{CidrIp: aws.String("10.0.0.0/8")}},
		}

		if *group.GroupName != "terraform_acceptance_test_example" {
			return fmt.Errorf("Bad name: %s", *group.GroupName)
		}

		if *group.Description != "Used in the terraform acceptance tests" {
			return fmt.Errorf("Bad description: %s", *group.Description)
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

func testAccCheckAWSSecurityGroupAttributesNegOneProtocol(group *ec2.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		p := &ec2.IpPermission{
			IpProtocol: aws.String("-1"),
			IpRanges:   []*ec2.IpRange{{CidrIp: aws.String("10.0.0.0/8")}},
		}

		if *group.GroupName != "terraform_acceptance_test_example" {
			return fmt.Errorf("Bad name: %s", *group.GroupName)
		}

		if *group.Description != "Used in the terraform acceptance tests" {
			return fmt.Errorf("Bad description: %s", *group.Description)
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

func TestAccAWSSecurityGroup_tags(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfigTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.foo", &group),
					testAccCheckTags(&group.Tags, "foo", "bar"),
				),
			},

			{
				Config: testAccAWSSecurityGroupConfigTagsUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.foo", &group),
					testAccCheckTags(&group.Tags, "foo", ""),
					testAccCheckTags(&group.Tags, "bar", "baz"),
					testAccCheckTags(&group.Tags, "env", "Production"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_CIDRandGroups(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupCombindCIDRandGroups,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.mixed", &group),
					// testAccCheckAWSSecurityGroupAttributes(&group),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_ingressWithCidrAndSGs(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfig_ingressWithCidrAndSGs,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupSGandCidrAttributes(&group),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "name", "terraform_acceptance_test_example"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "description", "Used in the terraform acceptance tests"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.#", "2"),
				),
			},
		},
	})
}

// This test requires an EC2 Classic region
func TestAccAWSSecurityGroup_ingressWithCidrAndSGs_classic(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfig_ingressWithCidrAndSGs_classic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupSGandCidrAttributes(&group),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "name", "terraform_acceptance_test_example"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "description", "Used in the terraform acceptance tests"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.#", "2"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_egressWithPrefixList(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfigPrefixListEgress,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.egress", &group),
					testAccCheckAWSSecurityGroupPrefixListAttributes(&group),
					resource.TestCheckResourceAttr(
						"aws_security_group.egress", "egress.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_ipv4andipv6Egress(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfigIpv4andIpv6Egress,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.egress", &group),
					resource.TestCheckResourceAttr(
						"aws_security_group.egress", "egress.#", "2"),
				),
			},
		},
	})
}

func testAccCheckAWSSecurityGroupSGandCidrAttributes(group *ec2.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *group.GroupName != "terraform_acceptance_test_example" {
			return fmt.Errorf("Bad name: %s", *group.GroupName)
		}

		if *group.Description != "Used in the terraform acceptance tests" {
			return fmt.Errorf("Bad description: %s", *group.Description)
		}

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

func testAccCheckAWSSecurityGroupPrefixListAttributes(group *ec2.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *group.GroupName != "terraform_acceptance_test_prefix_list_egress" {
			return fmt.Errorf("Bad name: %s", *group.GroupName)
		}
		if *group.Description != "Used in the terraform acceptance tests" {
			return fmt.Errorf("Bad description: %s", *group.Description)
		}
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

func testAccCheckAWSSecurityGroupAttributesChanged(group *ec2.SecurityGroup) resource.TestCheckFunc {
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

		if *group.GroupName != "terraform_acceptance_test_example" {
			return fmt.Errorf("Bad name: %s", *group.GroupName)
		}

		if *group.Description != "Used in the terraform acceptance tests" {
			return fmt.Errorf("Bad description: %s", *group.Description)
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

func testAccCheckAWSSecurityGroupExistsWithoutDefault(n string) resource.TestCheckFunc {
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
			group := *resp.SecurityGroups[0]

			if len(group.IpPermissionsEgress) != 1 {
				return fmt.Errorf("Security Group should have only 1 egress rule, got %d", len(group.IpPermissionsEgress))
			}
		}

		return nil
	}
}

func TestAccAWSSecurityGroup_failWithDiffMismatch(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfig_failWithDiffMismatch,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.nat", &group),
				),
			},
		},
	})
}

const testAccAWSSecurityGroupConfigForTagsOrdering = `
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
    to_port = 80000
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
}`

const testAccAWSSecurityGroupConfigIpv6 = `
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
    ipv6_cidr_blocks = ["::/0"]
  }

  egress {
    protocol = "tcp"
    from_port = 80
    to_port = 8000
    ipv6_cidr_blocks = ["::/0"]
  }

	tags {
		Name = "tf-acc-test"
	}
}
`

const testAccAWSSecurityGroupConfig = `
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
`

const testAccAWSSecurityGroupConfigChange = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"

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

const testAccAWSSecurityGroupConfigSelf = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"

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

const testAccAWSSecurityGroupConfigVpc = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"

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

const testAccAWSSecurityGroupConfigVpcNegOneIngress = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "web" {
	name = "terraform_acceptance_test_example"
	description = "Used in the terraform acceptance tests"
	vpc_id = "${aws_vpc.foo.id}"

	ingress {
		protocol = "-1"
		from_port = 0
		to_port = 0
		cidr_blocks = ["10.0.0.0/8"]
	}
}
`

const testAccAWSSecurityGroupConfigVpcProtoNumIngress = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "web" {
	name = "terraform_acceptance_test_example"
	description = "Used in the terraform acceptance tests"
	vpc_id = "${aws_vpc.foo.id}"

	ingress {
		protocol = "50"
		from_port = 0
		to_port = 0
		cidr_blocks = ["10.0.0.0/8"]
	}
}
`

const testAccAWSSecurityGroupConfigMultiIngress = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "worker" {
  name = "terraform_acceptance_test_example_1"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"

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

const testAccAWSSecurityGroupConfigTags = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "foo" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"

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

  tags {
    foo = "bar"
  }
}
`

const testAccAWSSecurityGroupConfigTagsUpdate = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "foo" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"

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

  tags {
    bar = "baz"
    env = "Production"
  }
}
`

const testAccAWSSecurityGroupConfig_generatedName = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_security_group" "web" {
  vpc_id = "${aws_vpc.foo.id}"

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

	tags {
		Name = "tf-acc-test"
	}
}
`

const testAccAWSSecurityGroupConfigDefaultEgress = `
resource "aws_vpc" "tf_sg_egress_test" {
        cidr_block = "10.0.0.0/16"
        tags {
                Name = "tf_sg_egress_test"
        }
}

resource "aws_security_group" "worker" {
  name = "terraform_acceptance_test_example_1"
  description = "Used in the terraform acceptance tests"
        vpc_id = "${aws_vpc.tf_sg_egress_test.id}"

  egress {
    protocol = "tcp"
    from_port = 80
    to_port = 8000
    cidr_blocks = ["10.0.0.0/8"]
  }
}
`

const testAccAWSSecurityGroupConfigClassic = `
provider "aws" {
  region = "us-east-1"
}

resource "aws_security_group" "web" {
  name = "terraform_acceptance_test_example_1"
  description = "Used in the terraform acceptance tests"
}
`

const testAccAWSSecurityGroupPrefixNameConfig = `
provider "aws" {
  region = "us-east-1"
}

resource "aws_security_group" "baz" {
   name_prefix = "baz-"
   description = "Used in the terraform acceptance tests"
}
`

func testAccAWSSecurityGroupConfig_drift() string {
	return fmt.Sprintf(`
resource "aws_security_group" "web" {
  name = "tf_acc_%d"
  description = "Used in the terraform acceptance tests"

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

        tags {
                Name = "tf-acc-test"
        }
}
`, acctest.RandInt())
}

func testAccAWSSecurityGroupConfig_drift_complex() string {
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

  tags {
    Name = "tf-acc-test"
  }
}`, acctest.RandInt(), acctest.RandInt())
}

const testAccAWSSecurityGroupInvalidIngressCidr = `
resource "aws_security_group" "foo" {
  name = "testing-foo"
  description = "foo-testing"
  ingress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["1.2.3.4/33"]
  }
}`

const testAccAWSSecurityGroupInvalidEgressCidr = `
resource "aws_security_group" "foo" {
  name = "testing-foo"
  description = "foo-testing"
  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["1.2.3.4/33"]
  }
}`

const testAccAWSSecurityGroupInvalidIPv6IngressCidr = `
resource "aws_security_group" "foo" {
  name = "testing-foo"
  description = "foo-testing"
  ingress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    ipv6_cidr_blocks = ["::/244"]
  }
}`

const testAccAWSSecurityGroupInvalidIPv6EgressCidr = `
resource "aws_security_group" "foo" {
  name = "testing-foo"
  description = "foo-testing"
  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    ipv6_cidr_blocks = ["::/244"]
  }
}`

const testAccAWSSecurityGroupCombindCIDRandGroups = `
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

  tags {
    Name = "tf-mix-test"
  }
}
`

const testAccAWSSecurityGroupConfig_ingressWithCidrAndSGs = `
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

  tags {
    Name = "tf-acc-test"
  }
}
`

const testAccAWSSecurityGroupConfig_ingressWithCidrAndSGs_classic = `
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

  tags {
    Name = "tf-acc-test"
  }
}
`

// fails to apply in one pass with the error "diffs didn't match during apply"
// GH-2027
const testAccAWSSecurityGroupConfig_failWithDiffMismatch = `
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
const testAccAWSSecurityGroupConfig_importSelf = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags {
    Name = "tf_sg_import_test"
  }
}

resource "aws_security_group" "allow_all" {
  name        = "allow_all"
  description = "Allow all inbound traffic"
  vpc_id      = "${aws_vpc.foo.id}"
}

resource "aws_security_group_rule" "allow_all" {
  type        = "ingress"
  from_port   = 0
  to_port     = 65535
  protocol    = "tcp"
  cidr_blocks = ["0.0.0.0/0"]

  security_group_id = "${aws_security_group.allow_all.id}"
}

resource "aws_security_group_rule" "allow_all-1" {
  type      = "ingress"
  from_port = 65534
  to_port   = 65535
  protocol  = "tcp"

  self              = true
  security_group_id = "${aws_security_group.allow_all.id}"
}
`

const testAccAWSSecurityGroupConfig_importSourceSecurityGroup = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags {
    Name = "tf_sg_import_test"
  }
}

resource "aws_security_group" "test_group_1" {
  name        = "test group 1"
  vpc_id      = "${aws_vpc.foo.id}"
}

resource "aws_security_group" "test_group_2" {
  name        = "test group 2"
  vpc_id      = "${aws_vpc.foo.id}"
}

resource "aws_security_group" "test_group_3" {
  name        = "test group 3"
  vpc_id      = "${aws_vpc.foo.id}"
}

resource "aws_security_group_rule" "allow_test_group_2" {
  type      = "ingress"
  from_port = 0
  to_port   = 0
  protocol  = "tcp"

  source_security_group_id = "${aws_security_group.test_group_1.id}"
  security_group_id = "${aws_security_group.test_group_2.id}"
}

resource "aws_security_group_rule" "allow_test_group_3" {
  type      = "ingress"
  from_port = 0
  to_port   = 0
  protocol  = "tcp"

  source_security_group_id = "${aws_security_group.test_group_1.id}"
  security_group_id = "${aws_security_group.test_group_3.id}"
}
`

const testAccAWSSecurityGroupConfig_importIPRangeAndSecurityGroupWithSameRules = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags {
    Name = "tf_sg_import_test"
  }
}

resource "aws_security_group" "test_group_1" {
  name        = "test group 1"
  vpc_id      = "${aws_vpc.foo.id}"
}

resource "aws_security_group" "test_group_2" {
  name        = "test group 2"
  vpc_id      = "${aws_vpc.foo.id}"
}

resource "aws_security_group_rule" "allow_security_group" {
  type      = "ingress"
  from_port = 0
  to_port   = 0
  protocol  = "tcp"

  source_security_group_id = "${aws_security_group.test_group_2.id}"
  security_group_id = "${aws_security_group.test_group_1.id}"
}

resource "aws_security_group_rule" "allow_cidr_block" {
  type      = "ingress"
  from_port = 0
  to_port   = 0
  protocol  = "tcp"

  cidr_blocks = ["10.0.0.0/32"]
  security_group_id = "${aws_security_group.test_group_1.id}"
}

resource "aws_security_group_rule" "allow_ipv6_cidr_block" {
  type      = "ingress"
  from_port = 0
  to_port   = 0
  protocol  = "tcp"

  ipv6_cidr_blocks = ["::/0"]
  security_group_id = "${aws_security_group.test_group_1.id}"
}
`

const testAccAWSSecurityGroupConfig_importIPRangesWithSameRules = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags {
    Name = "tf_sg_import_test"
  }
}

resource "aws_security_group" "test_group_1" {
  name        = "test group 1"
  vpc_id      = "${aws_vpc.foo.id}"
}

resource "aws_security_group_rule" "allow_cidr_block" {
  type      = "ingress"
  from_port = 0
  to_port   = 0
  protocol  = "tcp"

  cidr_blocks = ["10.0.0.0/32"]
  security_group_id = "${aws_security_group.test_group_1.id}"
}

resource "aws_security_group_rule" "allow_ipv6_cidr_block" {
  type      = "ingress"
  from_port = 0
  to_port   = 0
  protocol  = "tcp"

  ipv6_cidr_blocks = ["::/0"]
  security_group_id = "${aws_security_group.test_group_1.id}"
}
`

const testAccAWSSecurityGroupConfigIpv4andIpv6Egress = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
  assign_generated_ipv6_cidr_block = true
  tags {
      Name = "tf_sg_ipv4_and_ipv6_acc_test"
  }
}

resource "aws_security_group" "egress" {
  name = "terraform_acceptance_test_example"
  description = "Used in the terraform acceptance tests"
  vpc_id = "${aws_vpc.foo.id}"
  ingress {
      from_port = 22
      to_port = 22
      protocol = "6"
      cidr_blocks = ["0.0.0.0/0"]
  }
  egress {
    from_port       = 0
    to_port         = 0
    protocol        = "-1"
    cidr_blocks  = ["0.0.0.0/0"]
  }
  egress {
    from_port       = 0
    to_port         = 0
    protocol        = "-1"
    ipv6_cidr_blocks  = ["::/0"]
  }
}
`

const testAccAWSSecurityGroupConfigPrefixListEgress = `
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

    egress {
      protocol = "-1"
      from_port = 0
      to_port = 0
      prefix_list_ids = ["${aws_vpc_endpoint.s3-us-west-2.prefix_list_id}"]
    }
}
`
