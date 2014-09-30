package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func TestAccAWSSecurityGroup_normal(t *testing.T) {
	var group ec2.SecurityGroupInfo

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSecurityGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupAttributes(&group),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "name", "terraform_acceptance_test_example"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "description", "Used in the terraform acceptance tests"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.0.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.0.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.0.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.0.cidr_blocks.0", "10.0.0.0/8"),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_self(t *testing.T) {
	var group ec2.SecurityGroupInfo

	checkSelf := func(s *terraform.State) (err error) {
		defer func() {
			if e := recover(); e != nil {
				err = fmt.Errorf("bad: %#v", group)
			}
		}()

		if group.IPPerms[0].SourceGroups[0].Id != group.Id {
			return fmt.Errorf("bad: %#v", group)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSecurityGroupConfigSelf,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "name", "terraform_acceptance_test_example"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "description", "Used in the terraform acceptance tests"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.0.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.0.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.0.self", "true"),
					checkSelf,
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_vpc(t *testing.T) {
	var group ec2.SecurityGroupInfo

	testCheck := func(*terraform.State) error {
		if group.VpcId == "" {
			return fmt.Errorf("should have vpc ID")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSecurityGroupConfigVpc,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupAttributes(&group),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "name", "terraform_acceptance_test_example"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "description", "Used in the terraform acceptance tests"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.0.from_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.0.to_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.0.cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_security_group.web", "ingress.0.cidr_blocks.0", "10.0.0.0/8"),
					testCheck,
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_MultiIngress(t *testing.T) {
	var group ec2.SecurityGroupInfo

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSecurityGroupConfigMultiIngress,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
				),
			},
		},
	})
}

func TestAccAWSSecurityGroup_Change(t *testing.T) {
	var group ec2.SecurityGroupInfo

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSecurityGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
				),
			},
			resource.TestStep{
				Config: testAccAWSSecurityGroupConfigChange,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSecurityGroupExists("aws_security_group.web", &group),
					testAccCheckAWSSecurityGroupAttributesChanged(&group),
				),
			},
		},
	})
}

func testAccCheckAWSSecurityGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_security_group" {
			continue
		}

		sgs := []ec2.SecurityGroup{
			ec2.SecurityGroup{
				Id: rs.Primary.ID,
			},
		}

		// Retrieve our group
		resp, err := conn.SecurityGroups(sgs, nil)
		if err == nil {
			if len(resp.Groups) > 0 && resp.Groups[0].Id == rs.Primary.ID {
				return fmt.Errorf("Security Group (%s) still exists.", rs.Primary.ID)
			}

			return nil
		}

		ec2err, ok := err.(*ec2.Error)
		if !ok {
			return err
		}
		// Confirm error code is what we want
		if ec2err.Code != "InvalidGroup.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSSecurityGroupExists(n string, group *ec2.SecurityGroupInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Security Group is set")
		}

		conn := testAccProvider.ec2conn
		sgs := []ec2.SecurityGroup{
			ec2.SecurityGroup{
				Id: rs.Primary.ID,
			},
		}
		resp, err := conn.SecurityGroups(sgs, nil)
		if err != nil {
			return err
		}

		if len(resp.Groups) > 0 && resp.Groups[0].Id == rs.Primary.ID {

			*group = resp.Groups[0]

			return nil
		}

		return fmt.Errorf("Security Group not found")
	}
}

func testAccCheckAWSSecurityGroupAttributes(group *ec2.SecurityGroupInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		p := ec2.IPPerm{
			FromPort:  80,
			ToPort:    8000,
			Protocol:  "tcp",
			SourceIPs: []string{"10.0.0.0/8"},
		}

		if group.Name != "terraform_acceptance_test_example" {
			return fmt.Errorf("Bad name: %s", group.Name)
		}

		if group.Description != "Used in the terraform acceptance tests" {
			return fmt.Errorf("Bad description: %s", group.Description)
		}

		if len(group.IPPerms) == 0 {
			return fmt.Errorf("No IPPerms")
		}

		// Compare our ingress
		if !reflect.DeepEqual(group.IPPerms[0], p) {
			return fmt.Errorf(
				"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
				group.IPPerms[0],
				p)
		}

		return nil
	}
}

func testAccCheckAWSSecurityGroupAttributesChanged(group *ec2.SecurityGroupInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		p := []ec2.IPPerm{
			ec2.IPPerm{
				FromPort:  80,
				ToPort:    9000,
				Protocol:  "tcp",
				SourceIPs: []string{"10.0.0.0/8"},
			},
			ec2.IPPerm{
				FromPort:  80,
				ToPort:    8000,
				Protocol:  "tcp",
				SourceIPs: []string{"0.0.0.0/0", "10.0.0.0/8"},
			},
		}

		if group.Name != "terraform_acceptance_test_example" {
			return fmt.Errorf("Bad name: %s", group.Name)
		}

		if group.Description != "Used in the terraform acceptance tests" {
			return fmt.Errorf("Bad description: %s", group.Description)
		}

		// Compare our ingress
		if len(group.IPPerms) != 2 {
			return fmt.Errorf(
				"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
				group.IPPerms,
				p)
		}

		if group.IPPerms[0].ToPort == 8000 {
			group.IPPerms[1], group.IPPerms[0] =
				group.IPPerms[0], group.IPPerms[1]
		}

		if !reflect.DeepEqual(group.IPPerms, p) {
			return fmt.Errorf(
				"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
				group.IPPerms,
				p)
		}

		return nil
	}
}

const testAccAWSSecurityGroupConfig = `
resource "aws_security_group" "web" {
    name = "terraform_acceptance_test_example"
    description = "Used in the terraform acceptance tests"

    ingress {
        protocol = "tcp"
        from_port = 80
        to_port = 8000
        cidr_blocks = ["10.0.0.0/8"]
    }
}
`

const testAccAWSSecurityGroupConfigChange = `
resource "aws_security_group" "web" {
    name = "terraform_acceptance_test_example"
    description = "Used in the terraform acceptance tests"

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
        cidr_blocks = ["10.0.0.0/8", "0.0.0.0/0"]
    }
}
`

const testAccAWSSecurityGroupConfigSelf = `
resource "aws_security_group" "web" {
    name = "terraform_acceptance_test_example"
    description = "Used in the terraform acceptance tests"

    ingress {
        protocol = "tcp"
        from_port = 80
        to_port = 8000
        self = true
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
}
`

const testAccAWSSecurityGroupConfigMultiIngress = `
resource "aws_security_group" "worker" {
    name = "terraform_acceptance_test_example_1"
    description = "Used in the terraform acceptance tests"

    ingress {
        protocol = "tcp"
        from_port = 80
        to_port = 8000
        cidr_blocks = ["10.0.0.0/8"]
    }
}

resource "aws_security_group" "web" {
    name = "terraform_acceptance_test_example_2"
    description = "Used in the terraform acceptance tests"

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
}
`
