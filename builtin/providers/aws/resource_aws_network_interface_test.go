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

func TestAccAWSENI_basic(t *testing.T) {
	var conf ec2.NetworkInterface

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_network_interface.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSENIDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSENIConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSENIExists("aws_network_interface.bar", &conf),
					testAccCheckAWSENIAttributes(&conf),
					resource.TestCheckResourceAttr(
						"aws_network_interface.bar", "private_ips.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_network_interface.bar", "tags.Name", "bar_interface"),
					resource.TestCheckResourceAttr(
						"aws_network_interface.bar", "description", "Managed by Terraform"),
				),
			},
		},
	})
}

func TestAccAWSENI_updatedDescription(t *testing.T) {
	var conf ec2.NetworkInterface

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_network_interface.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSENIDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSENIConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSENIExists("aws_network_interface.bar", &conf),
					resource.TestCheckResourceAttr(
						"aws_network_interface.bar", "description", "Managed by Terraform"),
				),
			},

			resource.TestStep{
				Config: testAccAWSENIConfigUpdatedDescription,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSENIExists("aws_network_interface.bar", &conf),
					resource.TestCheckResourceAttr(
						"aws_network_interface.bar", "description", "Updated ENI Description"),
				),
			},
		},
	})
}

func TestAccAWSENI_attached(t *testing.T) {
	var conf ec2.NetworkInterface

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_network_interface.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSENIDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSENIConfigWithAttachment,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSENIExists("aws_network_interface.bar", &conf),
					testAccCheckAWSENIAttributesWithAttachment(&conf),
					resource.TestCheckResourceAttr(
						"aws_network_interface.bar", "private_ips.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_network_interface.bar", "tags.Name", "bar_interface"),
				),
			},
		},
	})
}

func TestAccAWSENI_ignoreExternalAttachment(t *testing.T) {
	var conf ec2.NetworkInterface

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_network_interface.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSENIDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSENIConfigExternalAttachment,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSENIExists("aws_network_interface.bar", &conf),
					testAccCheckAWSENIAttributes(&conf),
					testAccCheckAWSENIMakeExternalAttachment("aws_instance.foo", &conf),
				),
			},
		},
	})
}

func TestAccAWSENI_sourceDestCheck(t *testing.T) {
	var conf ec2.NetworkInterface

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_network_interface.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSENIDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSENIConfigWithSourceDestCheck,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSENIExists("aws_network_interface.bar", &conf),
					resource.TestCheckResourceAttr(
						"aws_network_interface.bar", "source_dest_check", "false"),
				),
			},
		},
	})
}

func TestAccAWSENI_computedIPs(t *testing.T) {
	var conf ec2.NetworkInterface

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_network_interface.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSENIDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSENIConfigWithNoPrivateIPs,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSENIExists("aws_network_interface.bar", &conf),
					resource.TestCheckResourceAttr(
						"aws_network_interface.bar", "private_ips.#", "1"),
				),
			},
		},
	})
}

func testAccCheckAWSENIExists(n string, res *ec2.NetworkInterface) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ENI ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		describe_network_interfaces_request := &ec2.DescribeNetworkInterfacesInput{
			NetworkInterfaceIds: []*string{aws.String(rs.Primary.ID)},
		}
		describeResp, err := conn.DescribeNetworkInterfaces(describe_network_interfaces_request)

		if err != nil {
			return err
		}

		if len(describeResp.NetworkInterfaces) != 1 ||
			*describeResp.NetworkInterfaces[0].NetworkInterfaceId != rs.Primary.ID {
			return fmt.Errorf("ENI not found")
		}

		*res = *describeResp.NetworkInterfaces[0]

		return nil
	}
}

func testAccCheckAWSENIAttributes(conf *ec2.NetworkInterface) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if conf.Attachment != nil {
			return fmt.Errorf("expected attachment to be nil")
		}

		if *conf.AvailabilityZone != "us-west-2a" {
			return fmt.Errorf("expected availability_zone to be us-west-2a, but was %s", *conf.AvailabilityZone)
		}

		if len(conf.Groups) != 1 && *conf.Groups[0].GroupName != "foo" {
			return fmt.Errorf("expected security group to be foo, but was %#v", conf.Groups)
		}

		if *conf.PrivateIpAddress != "172.16.10.100" {
			return fmt.Errorf("expected private ip to be 172.16.10.100, but was %s", *conf.PrivateIpAddress)
		}

		if *conf.SourceDestCheck != true {
			return fmt.Errorf("expected source_dest_check to be true, but was %t", *conf.SourceDestCheck)
		}

		if len(conf.TagSet) == 0 {
			return fmt.Errorf("expected tags")
		}

		return nil
	}
}

func testAccCheckAWSENIAttributesWithAttachment(conf *ec2.NetworkInterface) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if conf.Attachment == nil {
			return fmt.Errorf("expected attachment to be set, but was nil")
		}

		if *conf.Attachment.DeviceIndex != 1 {
			return fmt.Errorf("expected attachment device index to be 1, but was %d", *conf.Attachment.DeviceIndex)
		}

		if *conf.AvailabilityZone != "us-west-2a" {
			return fmt.Errorf("expected availability_zone to be us-west-2a, but was %s", *conf.AvailabilityZone)
		}

		if len(conf.Groups) != 1 && *conf.Groups[0].GroupName != "foo" {
			return fmt.Errorf("expected security group to be foo, but was %#v", conf.Groups)
		}

		if *conf.PrivateIpAddress != "172.16.10.100" {
			return fmt.Errorf("expected private ip to be 172.16.10.100, but was %s", *conf.PrivateIpAddress)
		}

		return nil
	}
}

func testAccCheckAWSENIDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_network_interface" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		describe_network_interfaces_request := &ec2.DescribeNetworkInterfacesInput{
			NetworkInterfaceIds: []*string{aws.String(rs.Primary.ID)},
		}
		_, err := conn.DescribeNetworkInterfaces(describe_network_interfaces_request)

		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidNetworkInterfaceID.NotFound" {
				return nil
			}

			return err
		}
	}

	return nil
}

func testAccCheckAWSENIMakeExternalAttachment(n string, conf *ec2.NetworkInterface) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok || rs.Primary.ID == "" {
			return fmt.Errorf("Not found: %s", n)
		}
		attach_request := &ec2.AttachNetworkInterfaceInput{
			DeviceIndex:        aws.Int64(2),
			InstanceId:         aws.String(rs.Primary.ID),
			NetworkInterfaceId: conf.NetworkInterfaceId,
		}
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		_, attach_err := conn.AttachNetworkInterface(attach_request)
		if attach_err != nil {
			return fmt.Errorf("Error attaching ENI: %s", attach_err)
		}
		return nil
	}
}

const testAccAWSENIConfig = `
resource "aws_vpc" "foo" {
    cidr_block = "172.16.0.0/16"
		tags {
			Name = "testAccAWSENIConfig"
		}
}

resource "aws_subnet" "foo" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "172.16.10.0/24"
    availability_zone = "us-west-2a"
}

resource "aws_security_group" "foo" {
  vpc_id = "${aws_vpc.foo.id}"
  description = "foo"
  name = "foo"

        egress {
                from_port = 0
                to_port = 0
                protocol = "tcp"
                cidr_blocks = ["10.0.0.0/16"]
        }
}

resource "aws_network_interface" "bar" {
    subnet_id = "${aws_subnet.foo.id}"
    private_ips = ["172.16.10.100"]
    security_groups = ["${aws_security_group.foo.id}"]
    description = "Managed by Terraform"
    tags {
        Name = "bar_interface"
    }
}
`

const testAccAWSENIConfigUpdatedDescription = `
resource "aws_vpc" "foo" {
    cidr_block = "172.16.0.0/16"
		tags {
			Name = "testAccAWSENIConfigUpdatedDescription"
		}
}

resource "aws_subnet" "foo" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "172.16.10.0/24"
    availability_zone = "us-west-2a"
}

resource "aws_security_group" "foo" {
  vpc_id = "${aws_vpc.foo.id}"
  description = "foo"
  name = "foo"

        egress {
                from_port = 0
                to_port = 0
                protocol = "tcp"
                cidr_blocks = ["10.0.0.0/16"]
        }
}

resource "aws_network_interface" "bar" {
    subnet_id = "${aws_subnet.foo.id}"
    private_ips = ["172.16.10.100"]
    security_groups = ["${aws_security_group.foo.id}"]
    description = "Updated ENI Description"
    tags {
        Name = "bar_interface"
    }
}
`

const testAccAWSENIConfigWithSourceDestCheck = `
resource "aws_vpc" "foo" {
    cidr_block = "172.16.0.0/16"
		tags {
			Name = "testAccAWSENIConfigWithSourceDestCheck"
		}
}

resource "aws_subnet" "foo" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "172.16.10.0/24"
    availability_zone = "us-west-2a"
}

resource "aws_network_interface" "bar" {
    subnet_id = "${aws_subnet.foo.id}"
        source_dest_check = false
    private_ips = ["172.16.10.100"]
}
`

const testAccAWSENIConfigWithNoPrivateIPs = `
resource "aws_vpc" "foo" {
    cidr_block = "172.16.0.0/16"
		tags {
			Name = "testAccAWSENIConfigWithNoPrivateIPs"
		}
}

resource "aws_subnet" "foo" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "172.16.10.0/24"
    availability_zone = "us-west-2a"
}

resource "aws_network_interface" "bar" {
    subnet_id = "${aws_subnet.foo.id}"
        source_dest_check = false
}
`

const testAccAWSENIConfigWithAttachment = `
resource "aws_vpc" "foo" {
    cidr_block = "172.16.0.0/16"
        tags {
            Name = "tf-eni-test"
        }
}

resource "aws_subnet" "foo" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "172.16.10.0/24"
    availability_zone = "us-west-2a"
        tags {
            Name = "tf-foo-eni-test"
        }
}

resource "aws_subnet" "bar" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "172.16.11.0/24"
    availability_zone = "us-west-2a"
        tags {
            Name = "tf-bar-eni-test"
        }
}

resource "aws_security_group" "foo" {
  vpc_id = "${aws_vpc.foo.id}"
  description = "foo"
  name = "foo"
}

resource "aws_instance" "foo" {
    ami = "ami-c5eabbf5"
    instance_type = "t2.micro"
    subnet_id = "${aws_subnet.bar.id}"
    associate_public_ip_address = false
    private_ip = "172.16.11.50"
    tags {
        Name = "foo-tf-eni-test"
    }
}

resource "aws_network_interface" "bar" {
    subnet_id = "${aws_subnet.foo.id}"
    private_ips = ["172.16.10.100"]
    security_groups = ["${aws_security_group.foo.id}"]
    attachment {
        instance = "${aws_instance.foo.id}"
        device_index = 1
    }
    tags {
        Name = "bar_interface"
    }
}
`

const testAccAWSENIConfigExternalAttachment = `
resource "aws_vpc" "foo" {
    cidr_block = "172.16.0.0/16"
        tags {
            Name = "tf-eni-test"
        }
}

resource "aws_subnet" "foo" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "172.16.10.0/24"
    availability_zone = "us-west-2a"
        tags {
            Name = "tf-eni-test"
        }
}

resource "aws_subnet" "bar" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "172.16.11.0/24"
    availability_zone = "us-west-2a"
        tags {
            Name = "tf-eni-test"
        }
}

resource "aws_security_group" "foo" {
  vpc_id = "${aws_vpc.foo.id}"
  description = "foo"
  name = "foo"
}

resource "aws_instance" "foo" {
    ami = "ami-c5eabbf5"
    instance_type = "t2.micro"
    subnet_id = "${aws_subnet.bar.id}"
    associate_public_ip_address = false
    private_ip = "172.16.11.50"
    tags {
        Name = "tf-eni-test"
    }
}

resource "aws_network_interface" "bar" {
    subnet_id = "${aws_subnet.foo.id}"
    private_ips = ["172.16.10.100"]
    security_groups = ["${aws_security_group.foo.id}"]
    tags {
        Name = "bar_interface"
    }
}
`
