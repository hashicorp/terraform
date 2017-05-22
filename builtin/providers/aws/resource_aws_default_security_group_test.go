package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDefaultSecurityGroup_basic(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_default_security_group.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSDefaultSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDefaultSecurityGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDefaultSecurityGroupExists("aws_default_security_group.web", &group),
					testAccCheckAWSDefaultSecurityGroupAttributes(&group),
					resource.TestCheckResourceAttr(
						"aws_default_security_group.web", "name", "default"),
					resource.TestCheckResourceAttr(
						"aws_default_security_group.web", "ingress.3629188364.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_default_security_group.web", "ingress.3629188364.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_default_security_group.web", "ingress.3629188364.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_default_security_group.web", "ingress.3629188364.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_default_security_group.web", "ingress.3629188364.cidr_blocks.0", "10.0.0.0/8"),
				),
			},
		},
	})
}

func TestAccAWSDefaultSecurityGroup_classic(t *testing.T) {
	var group ec2.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_default_security_group.web",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSDefaultSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDefaultSecurityGroupConfig_classic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDefaultSecurityGroupExists("aws_default_security_group.web", &group),
					testAccCheckAWSDefaultSecurityGroupAttributes(&group),
					resource.TestCheckResourceAttr(
						"aws_default_security_group.web", "name", "default"),
					resource.TestCheckResourceAttr(
						"aws_default_security_group.web", "ingress.3629188364.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_default_security_group.web", "ingress.3629188364.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_default_security_group.web", "ingress.3629188364.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_default_security_group.web", "ingress.3629188364.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_default_security_group.web", "ingress.3629188364.cidr_blocks.0", "10.0.0.0/8"),
				),
			},
		},
	})
}

func testAccCheckAWSDefaultSecurityGroupDestroy(s *terraform.State) error {
	// We expect Security Group to still exist
	return nil
}

func testAccCheckAWSDefaultSecurityGroupExists(n string, group *ec2.SecurityGroup) resource.TestCheckFunc {
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

func testAccCheckAWSDefaultSecurityGroupAttributes(group *ec2.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		p := &ec2.IpPermission{
			FromPort:   aws.Int64(80),
			ToPort:     aws.Int64(8000),
			IpProtocol: aws.String("tcp"),
			IpRanges:   []*ec2.IpRange{&ec2.IpRange{CidrIp: aws.String("10.0.0.0/8")}},
		}

		if *group.GroupName != "default" {
			return fmt.Errorf("Bad name: %s", *group.GroupName)
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

const testAccAWSDefaultSecurityGroupConfig = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccAWSDefaultSecurityGroupConfig"
	}
}

resource "aws_default_security_group" "web" {
  vpc_id = "${aws_vpc.foo.id}"

  ingress {
    protocol    = "6"
    from_port   = 80
    to_port     = 8000
    cidr_blocks = ["10.0.0.0/8"]
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

const testAccAWSDefaultSecurityGroupConfig_classic = `
provider "aws" {
  region = "us-east-1"
}

resource "aws_default_security_group" "web" {
  ingress {
    protocol    = "6"
    from_port   = 80
    to_port     = 8000
    cidr_blocks = ["10.0.0.0/8"]
  }

  tags {
    Name = "tf-acc-test"
  }
}`
