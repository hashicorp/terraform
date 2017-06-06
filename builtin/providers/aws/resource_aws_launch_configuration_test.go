package aws

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func init() {
	resource.AddTestSweepers("aws_launch_configuration", &resource.Sweeper{
		Name:         "aws_launch_configuration",
		Dependencies: []string{"aws_autoscaling_group"},
		F:            testSweepLaunchConfigurations,
	})
}

func testSweepLaunchConfigurations(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}
	autoscalingconn := client.(*AWSClient).autoscalingconn

	resp, err := autoscalingconn.DescribeLaunchConfigurations(&autoscaling.DescribeLaunchConfigurationsInput{})
	if err != nil {
		return fmt.Errorf("Error retrieving launch configuration: %s", err)
	}

	if len(resp.LaunchConfigurations) == 0 {
		log.Print("[DEBUG] No aws launch configurations to sweep")
		return nil
	}

	for _, lc := range resp.LaunchConfigurations {
		var testOptGroup bool
		for _, testName := range []string{"terraform-", "foobar"} {
			if strings.HasPrefix(*lc.LaunchConfigurationName, testName) {
				testOptGroup = true
			}
		}

		if !testOptGroup {
			continue
		}

		_, err := autoscalingconn.DeleteLaunchConfiguration(
			&autoscaling.DeleteLaunchConfigurationInput{
				LaunchConfigurationName: lc.LaunchConfigurationName,
			})
		if err != nil {
			autoscalingerr, ok := err.(awserr.Error)
			if ok && (autoscalingerr.Code() == "InvalidConfiguration.NotFound" || autoscalingerr.Code() == "ValidationError") {
				log.Printf("[DEBUG] Launch configuration (%s) not found", *lc.LaunchConfigurationName)
				return nil
			}

			return err
		}
	}

	return nil
}

func TestAccAWSLaunchConfiguration_basic(t *testing.T) {
	var conf autoscaling.LaunchConfiguration

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationNoNameConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.bar", &conf),
					testAccCheckAWSLaunchConfigurationGeneratedNamePrefix(
						"aws_launch_configuration.bar", "terraform-"),
				),
			},
			{
				Config: testAccAWSLaunchConfigurationPrefixNameConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.baz", &conf),
					testAccCheckAWSLaunchConfigurationGeneratedNamePrefix(
						"aws_launch_configuration.baz", "baz-"),
				),
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_withBlockDevices(t *testing.T) {
	var conf autoscaling.LaunchConfiguration

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.bar", &conf),
					testAccCheckAWSLaunchConfigurationAttributes(&conf),
					resource.TestCheckResourceAttr(
						"aws_launch_configuration.bar", "image_id", "ami-21f78e11"),
					resource.TestCheckResourceAttr(
						"aws_launch_configuration.bar", "instance_type", "m1.small"),
					resource.TestCheckResourceAttr(
						"aws_launch_configuration.bar", "associate_public_ip_address", "true"),
					resource.TestCheckResourceAttr(
						"aws_launch_configuration.bar", "spot_price", ""),
				),
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_updateRootBlockDevice(t *testing.T) {
	var conf autoscaling.LaunchConfiguration
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfigWithRootBlockDevice(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.bar", &conf),
					resource.TestCheckResourceAttr(
						"aws_launch_configuration.bar", "root_block_device.0.volume_size", "11"),
				),
			},
			{
				Config: testAccAWSLaunchConfigurationConfigWithRootBlockDeviceUpdated(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.bar", &conf),
					resource.TestCheckResourceAttr(
						"aws_launch_configuration.bar", "root_block_device.0.volume_size", "20"),
				),
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_withSpotPrice(t *testing.T) {
	var conf autoscaling.LaunchConfiguration

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationWithSpotPriceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.bar", &conf),
					resource.TestCheckResourceAttr(
						"aws_launch_configuration.bar", "spot_price", "0.01"),
				),
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_withVpcClassicLink(t *testing.T) {
	var vpc ec2.Vpc
	var group ec2.SecurityGroup
	var conf autoscaling.LaunchConfiguration

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfig_withVpcClassicLink,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.foo", &conf),
					testAccCheckVpcExists("aws_vpc.foo", &vpc),
					testAccCheckAWSSecurityGroupExists("aws_security_group.foo", &group),
				),
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_withIAMProfile(t *testing.T) {
	var conf autoscaling.LaunchConfiguration

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationConfig_withIAMProfile,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.bar", &conf),
				),
			},
		},
	})
}

func testAccCheckAWSLaunchConfigurationWithEncryption(conf *autoscaling.LaunchConfiguration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Map out the block devices by name, which should be unique.
		blockDevices := make(map[string]*autoscaling.BlockDeviceMapping)
		for _, blockDevice := range conf.BlockDeviceMappings {
			blockDevices[*blockDevice.DeviceName] = blockDevice
		}

		// Check if the root block device exists.
		if _, ok := blockDevices["/dev/sda1"]; !ok {
			return fmt.Errorf("block device doesn't exist: /dev/sda1")
		} else if blockDevices["/dev/sda1"].Ebs.Encrypted != nil {
			return fmt.Errorf("root device should not include value for Encrypted")
		}

		// Check if the secondary block device exists.
		if _, ok := blockDevices["/dev/sdb"]; !ok {
			return fmt.Errorf("block device doesn't exist: /dev/sdb")
		} else if !*blockDevices["/dev/sdb"].Ebs.Encrypted {
			return fmt.Errorf("block device isn't encrypted as expected: /dev/sdb")
		}

		return nil
	}
}

func TestAccAWSLaunchConfiguration_withEncryption(t *testing.T) {
	var conf autoscaling.LaunchConfiguration

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationWithEncryption,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.baz", &conf),
					testAccCheckAWSLaunchConfigurationWithEncryption(&conf),
				),
			},
		},
	})
}

func TestAccAWSLaunchConfiguration_updateEbsBlockDevices(t *testing.T) {
	var conf autoscaling.LaunchConfiguration

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLaunchConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLaunchConfigurationWithEncryption,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.baz", &conf),
					resource.TestCheckResourceAttr(
						"aws_launch_configuration.baz", "ebs_block_device.2764618555.volume_size", "9"),
				),
			},
			{
				Config: testAccAWSLaunchConfigurationWithEncryptionUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.baz", &conf),
					resource.TestCheckResourceAttr(
						"aws_launch_configuration.baz", "ebs_block_device.3859927736.volume_size", "10"),
				),
			},
		},
	})
}

func testAccCheckAWSLaunchConfigurationGeneratedNamePrefix(
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

func testAccCheckAWSLaunchConfigurationDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).autoscalingconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_launch_configuration" {
			continue
		}

		describe, err := conn.DescribeLaunchConfigurations(
			&autoscaling.DescribeLaunchConfigurationsInput{
				LaunchConfigurationNames: []*string{aws.String(rs.Primary.ID)},
			})

		if err == nil {
			if len(describe.LaunchConfigurations) != 0 &&
				*describe.LaunchConfigurations[0].LaunchConfigurationName == rs.Primary.ID {
				return fmt.Errorf("Launch Configuration still exists")
			}
		}

		// Verify the error
		providerErr, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if providerErr.Code() != "InvalidLaunchConfiguration.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSLaunchConfigurationAttributes(conf *autoscaling.LaunchConfiguration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *conf.ImageId != "ami-21f78e11" {
			return fmt.Errorf("Bad image_id: %s", *conf.ImageId)
		}

		if !strings.HasPrefix(*conf.LaunchConfigurationName, "terraform-") {
			return fmt.Errorf("Bad name: %s", *conf.LaunchConfigurationName)
		}

		if *conf.InstanceType != "m1.small" {
			return fmt.Errorf("Bad instance_type: %s", *conf.InstanceType)
		}

		// Map out the block devices by name, which should be unique.
		blockDevices := make(map[string]*autoscaling.BlockDeviceMapping)
		for _, blockDevice := range conf.BlockDeviceMappings {
			blockDevices[*blockDevice.DeviceName] = blockDevice
		}

		// Check if the root block device exists.
		if _, ok := blockDevices["/dev/sda1"]; !ok {
			return fmt.Errorf("block device doesn't exist: /dev/sda1")
		}

		// Check if the secondary block device exists.
		if _, ok := blockDevices["/dev/sdb"]; !ok {
			return fmt.Errorf("block device doesn't exist: /dev/sdb")
		}

		// Check if the third block device exists.
		if _, ok := blockDevices["/dev/sdc"]; !ok {
			return fmt.Errorf("block device doesn't exist: /dev/sdc")
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

		describeOpts := autoscaling.DescribeLaunchConfigurationsInput{
			LaunchConfigurationNames: []*string{aws.String(rs.Primary.ID)},
		}
		describe, err := conn.DescribeLaunchConfigurations(&describeOpts)

		if err != nil {
			return err
		}

		if len(describe.LaunchConfigurations) != 1 ||
			*describe.LaunchConfigurations[0].LaunchConfigurationName != rs.Primary.ID {
			return fmt.Errorf("Launch Configuration Group not found")
		}

		*res = *describe.LaunchConfigurations[0]

		return nil
	}
}

func testAccAWSLaunchConfigurationConfigWithRootBlockDevice(rInt int) string {
	return fmt.Sprintf(`
resource "aws_launch_configuration" "bar" {
  name_prefix = "tf-acc-test-%d"
  image_id = "ami-21f78e11"
  instance_type = "m1.small"
  user_data = "foobar-user-data"
  associate_public_ip_address = true

	root_block_device {
		volume_type = "gp2"
		volume_size = 11
	}

}
`, rInt)
}

func testAccAWSLaunchConfigurationConfigWithRootBlockDeviceUpdated(rInt int) string {
	return fmt.Sprintf(`
resource "aws_launch_configuration" "bar" {
  name_prefix = "tf-acc-test-%d"
  image_id = "ami-21f78e11"
  instance_type = "m1.small"
  user_data = "foobar-user-data"
  associate_public_ip_address = true

	root_block_device {
		volume_type = "gp2"
		volume_size = 20
	}

}
`, rInt)
}

var testAccAWSLaunchConfigurationConfig = fmt.Sprintf(`
resource "aws_launch_configuration" "bar" {
  name = "terraform-test-%d"
  image_id = "ami-21f78e11"
  instance_type = "m1.small"
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
`, rand.New(rand.NewSource(time.Now().UnixNano())).Int())

var testAccAWSLaunchConfigurationWithSpotPriceConfig = fmt.Sprintf(`
resource "aws_launch_configuration" "bar" {
  name = "terraform-test-%d"
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
  spot_price = "0.01"
}
`, rand.New(rand.NewSource(time.Now().UnixNano())).Int())

const testAccAWSLaunchConfigurationNoNameConfig = `
resource "aws_launch_configuration" "bar" {
   image_id = "ami-21f78e11"
   instance_type = "t1.micro"
   user_data = "foobar-user-data-change"
   associate_public_ip_address = false
}
`

const testAccAWSLaunchConfigurationPrefixNameConfig = `
resource "aws_launch_configuration" "baz" {
   name_prefix = "baz-"
   image_id = "ami-21f78e11"
   instance_type = "t1.micro"
   user_data = "foobar-user-data-change"
   associate_public_ip_address = false
}
`

const testAccAWSLaunchConfigurationWithEncryption = `
resource "aws_launch_configuration" "baz" {
   image_id = "ami-5189a661"
   instance_type = "t2.micro"
   associate_public_ip_address = false

   	root_block_device {
   		volume_type = "gp2"
		volume_size = 11
	}
	ebs_block_device {
		device_name = "/dev/sdb"
		volume_size = 9
		encrypted = true
	}
}
`

const testAccAWSLaunchConfigurationWithEncryptionUpdated = `
resource "aws_launch_configuration" "baz" {
   image_id = "ami-5189a661"
   instance_type = "t2.micro"
   associate_public_ip_address = false

   	root_block_device {
   		volume_type = "gp2"
		volume_size = 11
	}
	ebs_block_device {
		device_name = "/dev/sdb"
		volume_size = 10
		encrypted = true
	}
}
`

const testAccAWSLaunchConfigurationConfig_withVpcClassicLink = `
resource "aws_vpc" "foo" {
   cidr_block = "10.0.0.0/16"
   enable_classiclink = true
	tags {
		Name = "testAccAWSLaunchConfigurationConfig_withVpcClassicLink"
	}
}

resource "aws_security_group" "foo" {
  name = "foo"
  vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_launch_configuration" "foo" {
   name = "TestAccAWSLaunchConfiguration_withVpcClassicLink"
   image_id = "ami-21f78e11"
   instance_type = "t1.micro"

   vpc_classic_link_id = "${aws_vpc.foo.id}"
   vpc_classic_link_security_groups = ["${aws_security_group.foo.id}"]
}
`

const testAccAWSLaunchConfigurationConfig_withIAMProfile = `
resource "aws_iam_role" "role" {
	name  = "TestAccAWSLaunchConfiguration-withIAMProfile"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_instance_profile" "profile" {
	name  = "TestAccAWSLaunchConfiguration-withIAMProfile"
	roles = ["${aws_iam_role.role.name}"]
}

resource "aws_launch_configuration" "bar" {
	image_id             = "ami-5189a661"
	instance_type        = "t2.nano"
	iam_instance_profile = "${aws_iam_instance_profile.profile.name}"
}
`
