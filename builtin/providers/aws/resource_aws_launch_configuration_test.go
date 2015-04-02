package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/autoscaling"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSLaunchConfiguration(t *testing.T) {
	var conf autoscaling.LaunchConfiguration

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSLaunchConfigurationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.bar", &conf),
					testAccCheckAWSLaunchConfigurationAttributes(&conf),
					resource.TestCheckResourceAttr(
						"aws_launch_configuration.bar", "image_id", "ami-21f78e11"),
					resource.TestCheckResourceAttr(
						"aws_launch_configuration.bar", "name", "foobar-terraform-test"),
					resource.TestCheckResourceAttr(
						"aws_launch_configuration.bar", "instance_type", "t1.micro"),
					resource.TestCheckResourceAttr(
						"aws_launch_configuration.bar", "associate_public_ip_address", "true"),
					resource.TestCheckResourceAttr(
						"aws_launch_configuration.bar", "spot_price", ""),
				),
			},

			resource.TestStep{
				Config: TestAccAWSLaunchConfigurationWithSpotPriceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.bar", &conf),
					testAccCheckAWSLaunchConfigurationAttributes(&conf),
					resource.TestCheckResourceAttr(
						"aws_launch_configuration.bar", "spot_price", "0.01"),
				),
			},
		},
	})
}

func testAccCheckAWSLaunchConfigurationDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).autoscalingconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_launch_configuration" {
			continue
		}

		describe, err := conn.DescribeLaunchConfigurations(
			&autoscaling.LaunchConfigurationNamesType{
				LaunchConfigurationNames: []string{rs.Primary.ID},
			})

		if err == nil {
			if len(describe.LaunchConfigurations) != 0 &&
				*describe.LaunchConfigurations[0].LaunchConfigurationName == rs.Primary.ID {
				return fmt.Errorf("Launch Configuration still exists")
			}
		}

		// Verify the error
		providerErr, ok := err.(aws.APIError)
		if !ok {
			return err
		}
		if providerErr.Code != "InvalidLaunchConfiguration.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSLaunchConfigurationAttributes(conf *autoscaling.LaunchConfiguration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *conf.ImageID != "ami-21f78e11" {
			return fmt.Errorf("Bad image_id: %s", *conf.ImageID)
		}

		if *conf.LaunchConfigurationName != "foobar-terraform-test" {
			return fmt.Errorf("Bad name: %s", *conf.LaunchConfigurationName)
		}

		if *conf.InstanceType != "t1.micro" {
			return fmt.Errorf("Bad instance_type: %s", *conf.InstanceType)
		}

		// Map out the block devices by name, which should be unique.
		blockDevices := make(map[string]autoscaling.BlockDeviceMapping)
		for _, blockDevice := range conf.BlockDeviceMappings {
			blockDevices[*blockDevice.DeviceName] = blockDevice
		}

		// Check if the root block device exists.
		if _, ok := blockDevices["/dev/sda1"]; !ok {
			fmt.Errorf("block device doesn't exist: /dev/sda1")
		}

		// Check if the secondary block device exists.
		if _, ok := blockDevices["/dev/sdb"]; !ok {
			fmt.Errorf("block device doesn't exist: /dev/sdb")
		}

		// Check if the third block device exists.
		if _, ok := blockDevices["/dev/sdc"]; !ok {
			fmt.Errorf("block device doesn't exist: /dev/sdc")
		}

		// Check if the secondary block device exists.
		if _, ok := blockDevices["/dev/sdb"]; !ok {
			return fmt.Errorf("block device doesn't exist: /dev/sdb")
		}

		return nil
	}
}

func testAccCheckAWSLaunchConfigurationExists(n string, res *autoscaling.LaunchConfiguration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Launch Configuration ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).autoscalingconn

		describeOpts := autoscaling.LaunchConfigurationNamesType{
			LaunchConfigurationNames: []string{rs.Primary.ID},
		}
		describe, err := conn.DescribeLaunchConfigurations(&describeOpts)

		if err != nil {
			return err
		}

		if len(describe.LaunchConfigurations) != 1 ||
			*describe.LaunchConfigurations[0].LaunchConfigurationName != rs.Primary.ID {
			return fmt.Errorf("Launch Configuration Group not found")
		}

		*res = describe.LaunchConfigurations[0]

		return nil
	}
}

const testAccAWSLaunchConfigurationConfig = `
resource "aws_launch_configuration" "bar" {
  name = "foobar-terraform-test"
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
  user_data = "foobar-user-data"
  associate_public_ip_address = true

	root_block_device {
		volume_type = "gp2"
		volume_size = 11
	}
	ebs_block_device {
		device_name = "/dev/sdb"
		volume_size = 9
	}
	ebs_block_device {
		device_name = "/dev/sdc"
		volume_size = 10
		volume_type = "io1"
		iops = 100
	}
	ephemeral_block_device {
		device_name = "/dev/sde"
		virtual_name = "ephemeral0"
	}
}
`

const TestAccAWSLaunchConfigurationWithSpotPriceConfig = `
resource "aws_launch_configuration" "bar" {
  name = "foobar-terraform-test"
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
  user_data = "foobar-user-data"
  associate_public_ip_address = true
  spot_price = "0.01"
}
`
