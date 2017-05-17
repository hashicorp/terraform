package aws

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEIP_importEc2Classic(t *testing.T) {
	oldvar := os.Getenv("AWS_DEFAULT_REGION")
	os.Setenv("AWS_DEFAULT_REGION", "us-east-1")
	defer os.Setenv("AWS_DEFAULT_REGION", oldvar)

	resourceName := "aws_eip.bar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEIPDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEIPInstanceEc2Classic,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEIP_importVpc(t *testing.T) {
	resourceName := "aws_eip.bar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEIPDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEIPNetworkInterfaceConfig,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEIP_basic(t *testing.T) {
	var conf ec2.Address

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_eip.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSEIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEIPConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEIPExists("aws_eip.bar", &conf),
					testAccCheckAWSEIPAttributes(&conf),
				),
			},
		},
	})
}

func TestAccAWSEIP_instance(t *testing.T) {
	var conf ec2.Address

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_eip.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSEIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEIPInstanceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEIPExists("aws_eip.bar", &conf),
					testAccCheckAWSEIPAttributes(&conf),
				),
			},

			resource.TestStep{
				Config: testAccAWSEIPInstanceConfig2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEIPExists("aws_eip.bar", &conf),
					testAccCheckAWSEIPAttributes(&conf),
				),
			},
		},
	})
}

func TestAccAWSEIP_network_interface(t *testing.T) {
	var conf ec2.Address

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_eip.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSEIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEIPNetworkInterfaceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEIPExists("aws_eip.bar", &conf),
					testAccCheckAWSEIPAttributes(&conf),
					testAccCheckAWSEIPAssociated(&conf),
				),
			},
		},
	})
}

func TestAccAWSEIP_twoEIPsOneNetworkInterface(t *testing.T) {
	var one, two ec2.Address

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_eip.one",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSEIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEIPMultiNetworkInterfaceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEIPExists("aws_eip.one", &one),
					testAccCheckAWSEIPAttributes(&one),
					testAccCheckAWSEIPAssociated(&one),
					testAccCheckAWSEIPExists("aws_eip.two", &two),
					testAccCheckAWSEIPAttributes(&two),
					testAccCheckAWSEIPAssociated(&two),
				),
			},
		},
	})
}

// This test is an expansion of TestAccAWSEIP_instance, by testing the
// associated Private EIPs of two instances
func TestAccAWSEIP_associated_user_private_ip(t *testing.T) {
	var one ec2.Address

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_eip.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSEIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEIPInstanceConfig_associated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEIPExists("aws_eip.bar", &one),
					testAccCheckAWSEIPAttributes(&one),
					testAccCheckAWSEIPAssociated(&one),
				),
			},

			resource.TestStep{
				Config: testAccAWSEIPInstanceConfig_associated_switch,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEIPExists("aws_eip.bar", &one),
					testAccCheckAWSEIPAttributes(&one),
					testAccCheckAWSEIPAssociated(&one),
				),
			},
		},
	})
}

func testAccCheckAWSEIPDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_eip" {
			continue
		}

		if strings.Contains(rs.Primary.ID, "eipalloc") {
			req := &ec2.DescribeAddressesInput{
				AllocationIds: []*string{aws.String(rs.Primary.ID)},
			}
			describe, err := conn.DescribeAddresses(req)
			if err != nil {
				// Verify the error is what we want
				if ae, ok := err.(awserr.Error); ok && ae.Code() == "InvalidAllocationID.NotFound" || ae.Code() == "InvalidAddress.NotFound" {
					continue
				}
				return err
			}

			if len(describe.Addresses) > 0 {
				return fmt.Errorf("still exists")
			}
		} else {
			req := &ec2.DescribeAddressesInput{
				PublicIps: []*string{aws.String(rs.Primary.ID)},
			}
			describe, err := conn.DescribeAddresses(req)
			if err != nil {
				// Verify the error is what we want
				if ae, ok := err.(awserr.Error); ok && ae.Code() == "InvalidAllocationID.NotFound" || ae.Code() == "InvalidAddress.NotFound" {
					continue
				}
				return err
			}

			if len(describe.Addresses) > 0 {
				return fmt.Errorf("still exists")
			}
		}
	}

	return nil
}

func testAccCheckAWSEIPAttributes(conf *ec2.Address) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *conf.PublicIp == "" {
			return fmt.Errorf("empty public_ip")
		}

		return nil
	}
}

func testAccCheckAWSEIPAssociated(conf *ec2.Address) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if conf.AssociationId == nil || *conf.AssociationId == "" {
			return fmt.Errorf("empty association_id")
		}

		return nil
	}
}

func testAccCheckAWSEIPExists(n string, res *ec2.Address) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No EIP ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		if strings.Contains(rs.Primary.ID, "eipalloc") {
			req := &ec2.DescribeAddressesInput{
				AllocationIds: []*string{aws.String(rs.Primary.ID)},
			}
			describe, err := conn.DescribeAddresses(req)
			if err != nil {
				return err
			}

			if len(describe.Addresses) != 1 ||
				*describe.Addresses[0].AllocationId != rs.Primary.ID {
				return fmt.Errorf("EIP not found")
			}
			*res = *describe.Addresses[0]

		} else {
			req := &ec2.DescribeAddressesInput{
				PublicIps: []*string{aws.String(rs.Primary.ID)},
			}
			describe, err := conn.DescribeAddresses(req)
			if err != nil {
				return err
			}

			if len(describe.Addresses) != 1 ||
				*describe.Addresses[0].PublicIp != rs.Primary.ID {
				return fmt.Errorf("EIP not found")
			}
			*res = *describe.Addresses[0]
		}

		return nil
	}
}

const testAccAWSEIPConfig = `
resource "aws_eip" "bar" {
}
`

const testAccAWSEIPInstanceEc2Classic = `
provider "aws" {
	region = "us-east-1"
}
resource "aws_instance" "foo" {
	ami = "ami-5469ae3c"
	instance_type = "m1.small"
}

resource "aws_eip" "bar" {
	instance = "${aws_instance.foo.id}"
}
`

const testAccAWSEIPInstanceConfig = `
resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
}

resource "aws_eip" "bar" {
	instance = "${aws_instance.foo.id}"
}
`

const testAccAWSEIPInstanceConfig2 = `
resource "aws_instance" "bar" {
	# us-west-2
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
}

resource "aws_eip" "bar" {
	instance = "${aws_instance.bar.id}"
}
`

const testAccAWSEIPInstanceConfig_associated = `
resource "aws_vpc" "default" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true

  tags {
    Name = "default"
  }
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.default.id}"

  tags {
    Name = "main"
  }
}

resource "aws_subnet" "tf_test_subnet" {
  vpc_id                  = "${aws_vpc.default.id}"
  cidr_block              = "10.0.0.0/24"
  map_public_ip_on_launch = true

  depends_on = ["aws_internet_gateway.gw"]

  tags {
    Name = "tf_test_subnet"
  }
}

resource "aws_instance" "foo" {
  # us-west-2
  ami           = "ami-5189a661"
  instance_type = "t2.micro"

  private_ip = "10.0.0.12"
  subnet_id  = "${aws_subnet.tf_test_subnet.id}"

  tags {
    Name = "foo instance"
  }
}

resource "aws_instance" "bar" {
  # us-west-2

  ami = "ami-5189a661"

  instance_type = "t2.micro"

  private_ip = "10.0.0.19"
  subnet_id  = "${aws_subnet.tf_test_subnet.id}"

  tags {
    Name = "bar instance"
  }
}

resource "aws_eip" "bar" {
  vpc = true

  instance                  = "${aws_instance.bar.id}"
  associate_with_private_ip = "10.0.0.19"
}
`
const testAccAWSEIPInstanceConfig_associated_switch = `
resource "aws_vpc" "default" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true

  tags {
    Name = "default"
  }
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.default.id}"

  tags {
    Name = "main"
  }
}

resource "aws_subnet" "tf_test_subnet" {
  vpc_id                  = "${aws_vpc.default.id}"
  cidr_block              = "10.0.0.0/24"
  map_public_ip_on_launch = true

  depends_on = ["aws_internet_gateway.gw"]

  tags {
    Name = "tf_test_subnet"
  }
}

resource "aws_instance" "foo" {
  # us-west-2
  ami           = "ami-5189a661"
  instance_type = "t2.micro"

  private_ip = "10.0.0.12"
  subnet_id  = "${aws_subnet.tf_test_subnet.id}"

  tags {
    Name = "foo instance"
  }
}

resource "aws_instance" "bar" {
  # us-west-2

  ami = "ami-5189a661"

  instance_type = "t2.micro"

  private_ip = "10.0.0.19"
  subnet_id  = "${aws_subnet.tf_test_subnet.id}"

  tags {
    Name = "bar instance"
  }
}

resource "aws_eip" "bar" {
  vpc = true

  instance                  = "${aws_instance.foo.id}"
  associate_with_private_ip = "10.0.0.12"
}
`

const testAccAWSEIPInstanceConfig_associated_update = `
resource "aws_instance" "bar" {
	# us-west-2
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
}

resource "aws_eip" "bar" {
	instance = "${aws_instance.bar.id}"
}
`

const testAccAWSEIPNetworkInterfaceConfig = `
resource "aws_vpc" "bar" {
	cidr_block = "10.0.0.0/24"
	tags {
		Name = "testAccAWSEIPNetworkInterfaceConfig"
	}
}
resource "aws_internet_gateway" "bar" {
	vpc_id = "${aws_vpc.bar.id}"
}
resource "aws_subnet" "bar" {
  vpc_id = "${aws_vpc.bar.id}"
  availability_zone = "us-west-2a"
  cidr_block = "10.0.0.0/24"
}
resource "aws_network_interface" "bar" {
  subnet_id = "${aws_subnet.bar.id}"
	private_ips = ["10.0.0.10"]
  security_groups = [ "${aws_vpc.bar.default_security_group_id}" ]
}
resource "aws_eip" "bar" {
	vpc = "true"
	network_interface = "${aws_network_interface.bar.id}"
}
`

const testAccAWSEIPMultiNetworkInterfaceConfig = `
resource "aws_vpc" "bar" {
  cidr_block = "10.0.0.0/24"
	tags {
		Name = "testAccAWSEIPMultiNetworkInterfaceConfig"
	}
}

resource "aws_internet_gateway" "bar" {
  vpc_id = "${aws_vpc.bar.id}"
}

resource "aws_subnet" "bar" {
  vpc_id            = "${aws_vpc.bar.id}"
  availability_zone = "us-west-2a"
  cidr_block        = "10.0.0.0/24"
}

resource "aws_network_interface" "bar" {
  subnet_id       = "${aws_subnet.bar.id}"
  private_ips     = ["10.0.0.10", "10.0.0.11"]
  security_groups = ["${aws_vpc.bar.default_security_group_id}"]
}

resource "aws_eip" "one" {
  vpc                       = "true"
  network_interface         = "${aws_network_interface.bar.id}"
  associate_with_private_ip = "10.0.0.10"
}

resource "aws_eip" "two" {
  vpc                       = "true"
  network_interface         = "${aws_network_interface.bar.id}"
  associate_with_private_ip = "10.0.0.11"
}
`
