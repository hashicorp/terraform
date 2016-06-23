package aws

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSpotFleetRequest_basic(t *testing.T) {
	var sfr ec2.SpotFleetRequestConfig

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotFleetRequestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSpotFleetRequestConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSpotFleetRequestExists(
						"aws_spot_fleet_request.foo", &sfr),
					testAccCheckAWSSpotFleetRequestAttributes(&sfr),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "spot_request_state", "active"),
				),
			},
		},
	})
}

func TestAccAWSSpotFleetRequest_launchConfiguration(t *testing.T) {
	var sfr ec2.SpotFleetRequestConfig

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotFleetRequestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSpotFleetRequestWithAdvancedLaunchSpecConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSpotFleetRequestExists(
						"aws_spot_fleet_request.foo", &sfr),
					testAccCheckAWSSpotFleetRequest_LaunchSpecAttributes(&sfr),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "spot_request_state", "active"),
				),
			},
		},
	})
}

func TestAccAWSSpotFleetRequest_CannotUseEmptyKeyName(t *testing.T) {
	_, errors := validateSpotFleetRequestKeyName("", "key_name")
	if len(errors) == 0 {
		t.Fatalf("Expected the key name to trigger a validation error")
	}
}

func testAccCheckAWSSpotFleetRequestExists(
	n string, sfr *ec2.SpotFleetRequestConfig) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Spot fleet request with that id exists")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		params := &ec2.DescribeSpotFleetRequestsInput{
			SpotFleetRequestIds: []*string{&rs.Primary.ID},
		}
		resp, err := conn.DescribeSpotFleetRequests(params)

		if err != nil {
			return err
		}

		if v := len(resp.SpotFleetRequestConfigs); v != 1 {
			return fmt.Errorf("Expected 1 request returned, got %d", v)
		}

		*sfr = *resp.SpotFleetRequestConfigs[0]

		return nil
	}
}

func testAccCheckAWSSpotFleetRequestAttributes(
	sfr *ec2.SpotFleetRequestConfig) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *sfr.SpotFleetRequestConfig.SpotPrice != "0.005" {
			return fmt.Errorf("Unexpected spot price: %s", *sfr.SpotFleetRequestConfig.SpotPrice)
		}
		if *sfr.SpotFleetRequestState != "active" {
			return fmt.Errorf("Unexpected request state: %s", *sfr.SpotFleetRequestState)
		}
		return nil
	}
}

func testAccCheckAWSSpotFleetRequest_LaunchSpecAttributes(
	sfr *ec2.SpotFleetRequestConfig) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(sfr.SpotFleetRequestConfig.LaunchSpecifications) == 0 {
			return fmt.Errorf("Missing launch specification")
		}

		spec := *sfr.SpotFleetRequestConfig.LaunchSpecifications[0]

		if *spec.InstanceType != "m1.small" {
			return fmt.Errorf("Unexpected launch specification instance type: %s", *spec.InstanceType)
		}

		if *spec.ImageId != "ami-d06a90b0" {
			return fmt.Errorf("Unexpected launch specification image id: %s", *spec.ImageId)
		}

		if *spec.SpotPrice != "0.01" {
			return fmt.Errorf("Unexpected launch specification spot price: %s", *spec.SpotPrice)
		}

		if *spec.WeightedCapacity != 2 {
			return fmt.Errorf("Unexpected launch specification weighted capacity: %f", *spec.WeightedCapacity)
		}

		if *spec.UserData != base64.StdEncoding.EncodeToString([]byte("hello-world")) {
			return fmt.Errorf("Unexpected launch specification user data: %s", *spec.UserData)
		}

		return nil
	}
}

func testAccCheckAWSSpotFleetRequestDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_spot_fleet_request" {
			continue
		}

		_, err := conn.CancelSpotFleetRequests(&ec2.CancelSpotFleetRequestsInput{
			SpotFleetRequestIds: []*string{aws.String(rs.Primary.ID)},
			TerminateInstances:  aws.Bool(true),
		})

		if err != nil {
			return fmt.Errorf("Error cancelling spot request (%s): %s", rs.Primary.ID, err)
		}
	}

	return nil
}

const testAccAWSSpotFleetRequestConfig = `
resource "aws_key_pair" "debugging" {
	key_name = "tmp-key"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}

resource "aws_iam_policy_attachment" "test-attach" {
    name = "test-attachment"
    roles = ["${aws_iam_role.test-role.name}"]
    policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetRole"
}

resource "aws_iam_role" "test-role" {
    name = "test-role"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "spotfleet.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_spot_fleet_request" "foo" {
    iam_fleet_role = "${aws_iam_role.test-role.arn}"
    spot_price = "0.005"
    target_capacity = 2
    valid_until = "2019-11-04T20:44:20Z"
    launch_specification {
        instance_type = "m1.small"
        ami = "ami-d06a90b0"
        key_name = "${aws_key_pair.debugging.key_name}"
        availability_zone = "us-west-2a"
    }
    depends_on = ["aws_iam_policy_attachment.test-attach"]
}
`

const testAccAWSSpotFleetRequestWithAdvancedLaunchSpecConfig = `
resource "aws_key_pair" "debugging" {
	key_name = "tmp-key"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}

resource "aws_iam_policy_attachment" "test-attach" {
    name = "test-attachment"
    roles = ["${aws_iam_role.test-role.name}"]
    policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetRole"
}

resource "aws_iam_role" "test-role" {
    name = "test-role"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "spotfleet.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_spot_fleet_request" "foo" {
    iam_fleet_role = "${aws_iam_role.test-role.arn}"
    spot_price = "0.005"
    target_capacity = 4
    valid_until = "2019-11-04T20:44:20Z"
    allocation_strategy = "diversified"
    launch_specification {
        instance_type = "m1.small"
        ami = "ami-d06a90b0"
        key_name = "${aws_key_pair.debugging.key_name}"
        availability_zone = "us-west-2a"
        spot_price = "0.01"
        weighted_capacity = 2
        user_data = "hello-world"
        root_block_device {
            volume_size = "300"
            volume_type = "gp2"
        }
    }
    depends_on = ["aws_iam_policy_attachment.test-attach"]
}
`
