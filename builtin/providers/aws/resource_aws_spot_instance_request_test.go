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

func TestAccAWSSpotInstanceRequest_basic(t *testing.T) {
	var sir ec2.SpotInstanceRequest

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotInstanceRequestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSpotInstanceRequestConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSpotInstanceRequestExists(
						"aws_spot_instance_request.foo", &sir),
					testAccCheckAWSSpotInstanceRequestAttributes(&sir),
					testCheckKeyPair("tmp-key", &sir),
					resource.TestCheckResourceAttr(
						"aws_spot_instance_request.foo", "spot_bid_status", "fulfilled"),
					resource.TestCheckResourceAttr(
						"aws_spot_instance_request.foo", "spot_request_state", "active"),
				),
			},
		},
	})
}

func TestAccAWSSpotInstanceRequest_withBlockDuration(t *testing.T) {
	var sir ec2.SpotInstanceRequest

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotInstanceRequestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSpotInstanceRequestConfig_withBlockDuration,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSpotInstanceRequestExists(
						"aws_spot_instance_request.foo", &sir),
					testAccCheckAWSSpotInstanceRequestAttributes(&sir),
					testCheckKeyPair("tmp-key", &sir),
					resource.TestCheckResourceAttr(
						"aws_spot_instance_request.foo", "spot_bid_status", "fulfilled"),
					resource.TestCheckResourceAttr(
						"aws_spot_instance_request.foo", "spot_request_state", "active"),
					resource.TestCheckResourceAttr(
						"aws_spot_instance_request.foo", "block_duration_minutes", "60"),
				),
			},
		},
	})
}

func TestAccAWSSpotInstanceRequest_vpc(t *testing.T) {
	var sir ec2.SpotInstanceRequest

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotInstanceRequestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSpotInstanceRequestConfigVPC,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSpotInstanceRequestExists(
						"aws_spot_instance_request.foo_VPC", &sir),
					testAccCheckAWSSpotInstanceRequestAttributes(&sir),
					testCheckKeyPair("tmp-key", &sir),
					testAccCheckAWSSpotInstanceRequestAttributesVPC(&sir),
					resource.TestCheckResourceAttr(
						"aws_spot_instance_request.foo_VPC", "spot_bid_status", "fulfilled"),
					resource.TestCheckResourceAttr(
						"aws_spot_instance_request.foo_VPC", "spot_request_state", "active"),
				),
			},
		},
	})
}

func TestAccAWSSpotInstanceRequest_SubnetAndSG(t *testing.T) {
	var sir ec2.SpotInstanceRequest

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotInstanceRequestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSpotInstanceRequestConfig_SubnetAndSG,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSpotInstanceRequestExists(
						"aws_spot_instance_request.foo", &sir),
					testAccCheckAWSSpotInstanceRequest_InstanceAttributes(&sir),
				),
			},
		},
	})
}

func testCheckKeyPair(keyName string, sir *ec2.SpotInstanceRequest) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if sir.LaunchSpecification.KeyName == nil {
			return fmt.Errorf("No Key Pair found, expected(%s)", keyName)
		}
		if sir.LaunchSpecification.KeyName != nil && *sir.LaunchSpecification.KeyName != keyName {
			return fmt.Errorf("Bad key name, expected (%s), got (%s)", keyName, *sir.LaunchSpecification.KeyName)
		}

		return nil
	}
}

func testAccCheckAWSSpotInstanceRequestDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_spot_instance_request" {
			continue
		}

		req := &ec2.DescribeSpotInstanceRequestsInput{
			SpotInstanceRequestIds: []*string{aws.String(rs.Primary.ID)},
		}

		resp, err := conn.DescribeSpotInstanceRequests(req)
		var s *ec2.SpotInstanceRequest
		if err == nil {
			for _, sir := range resp.SpotInstanceRequests {
				if sir.SpotInstanceRequestId != nil && *sir.SpotInstanceRequestId == rs.Primary.ID {
					s = sir
				}
				continue
			}
		}

		if s == nil {
			// not found
			return nil
		}

		if *s.State == "canceled" {
			// Requests stick around for a while, so we make sure it's cancelled
			return nil
		}

		// Verify the error is what we expect
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidSpotInstanceRequestID.NotFound" {
			return err
		}

		// Now check if the associated Spot Instance was also destroyed
		instId := rs.Primary.Attributes["spot_instance_id"]
		instResp, instErr := conn.DescribeInstances(&ec2.DescribeInstancesInput{
			InstanceIds: []*string{aws.String(instId)},
		})
		if instErr == nil {
			if len(instResp.Reservations) > 0 {
				return fmt.Errorf("Instance still exists.")
			}

			return nil
		}

		// Verify the error is what we expect
		ec2err, ok = err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidInstanceID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSSpotInstanceRequestExists(
	n string, sir *ec2.SpotInstanceRequest) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SNS subscription with that ARN exists")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		params := &ec2.DescribeSpotInstanceRequestsInput{
			SpotInstanceRequestIds: []*string{&rs.Primary.ID},
		}
		resp, err := conn.DescribeSpotInstanceRequests(params)

		if err != nil {
			return err
		}

		if v := len(resp.SpotInstanceRequests); v != 1 {
			return fmt.Errorf("Expected 1 request returned, got %d", v)
		}

		*sir = *resp.SpotInstanceRequests[0]

		return nil
	}
}

func testAccCheckAWSSpotInstanceRequestAttributes(
	sir *ec2.SpotInstanceRequest) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *sir.SpotPrice != "0.050000" {
			return fmt.Errorf("Unexpected spot price: %s", *sir.SpotPrice)
		}
		if *sir.State != "active" {
			return fmt.Errorf("Unexpected request state: %s", *sir.State)
		}
		if *sir.Status.Code != "fulfilled" {
			return fmt.Errorf("Unexpected bid status: %s", *sir.State)
		}
		return nil
	}
}

func testAccCheckAWSSpotInstanceRequest_InstanceAttributes(
	sir *ec2.SpotInstanceRequest) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := conn.DescribeInstances(&ec2.DescribeInstancesInput{
			InstanceIds: []*string{sir.InstanceId},
		})
		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidInstanceID.NotFound" {
				return fmt.Errorf("Spot Instance not found")
			}
			return err
		}

		// If nothing was found, then return no state
		if len(resp.Reservations) == 0 {
			return fmt.Errorf("Spot Instance not found")
		}

		instance := resp.Reservations[0].Instances[0]

		var sgMatch bool
		for _, s := range instance.SecurityGroups {
			// Hardcoded name for the security group that should be added inside the
			// VPC
			if *s.GroupName == "tf_test_sg_ssh" {
				sgMatch = true
			}
		}

		if !sgMatch {
			return fmt.Errorf("Error in matching Spot Instance Security Group, expected 'tf_test_sg_ssh', got %s", instance.SecurityGroups)
		}

		return nil
	}
}

func testAccCheckAWSSpotInstanceRequestAttributesVPC(
	sir *ec2.SpotInstanceRequest) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if sir.LaunchSpecification.SubnetId == nil {
			return fmt.Errorf("SubnetID was not passed, but should have been for this instance to belong to a VPC")
		}
		return nil
	}
}

const testAccAWSSpotInstanceRequestConfig = `
resource "aws_key_pair" "debugging" {
	key_name = "tmp-key"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}

resource "aws_spot_instance_request" "foo" {
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
	key_name = "${aws_key_pair.debugging.key_name}"

	// base price is $0.044 hourly, so bidding above that should theoretically
	// always fulfill
	spot_price = "0.05"

	// we wait for fulfillment because we want to inspect the launched instance
	// and verify termination behavior
	wait_for_fulfillment = true

	tags {
		Name = "terraform-test"
	}
}
`

const testAccAWSSpotInstanceRequestConfig_withBlockDuration = `
resource "aws_key_pair" "debugging" {
	key_name = "tmp-key"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}

resource "aws_spot_instance_request" "foo" {
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
	key_name = "${aws_key_pair.debugging.key_name}"

	// base price is $0.044 hourly, so bidding above that should theoretically
	// always fulfill
	spot_price = "0.05"

	// we wait for fulfillment because we want to inspect the launched instance
	// and verify termination behavior
	wait_for_fulfillment = true

	block_duration_minutes = 60

	tags {
		Name = "terraform-test"
	}
}
`

const testAccAWSSpotInstanceRequestConfigVPC = `
resource "aws_vpc" "foo_VPC" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo_VPC" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo_VPC.id}"
}

resource "aws_key_pair" "debugging" {
	key_name = "tmp-key"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}

resource "aws_spot_instance_request" "foo_VPC" {
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
	key_name = "${aws_key_pair.debugging.key_name}"

	// base price is $0.044 hourly, so bidding above that should theoretically
	// always fulfill
	spot_price = "0.05"

	// VPC settings
	subnet_id = "${aws_subnet.foo_VPC.id}"

	// we wait for fulfillment because we want to inspect the launched instance
	// and verify termination behavior
	wait_for_fulfillment = true

	tags {
		Name = "terraform-test-VPC"
	}
}
`

const testAccAWSSpotInstanceRequestConfig_SubnetAndSG = `
resource "aws_spot_instance_request" "foo" {
  ami                         = "ami-6f6d635f"
  spot_price                  = "0.05"
  instance_type               = "t1.micro"
  wait_for_fulfillment        = true
  subnet_id                   = "${aws_subnet.tf_test_subnet.id}"
  vpc_security_group_ids      = ["${aws_security_group.tf_test_sg_ssh.id}"]
  associate_public_ip_address = true
}

resource "aws_vpc" "default" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true

  tags {
    Name = "tf_test_vpc"
  }
}

resource "aws_subnet" "tf_test_subnet" {
  vpc_id                  = "${aws_vpc.default.id}"
  cidr_block              = "10.0.0.0/24"
  map_public_ip_on_launch = true

  tags {
    Name = "tf_test_subnet"
  }
}

resource "aws_security_group" "tf_test_sg_ssh" {
  name        = "tf_test_sg_ssh"
  description = "tf_test_sg_ssh"
  vpc_id      = "${aws_vpc.default.id}"

  tags {
    Name = "tf_test_sg_ssh"
  }
}
`
