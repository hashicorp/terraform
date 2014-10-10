package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/autoscaling"
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
				),
			},
		},
	})
}

func testAccCheckAWSLaunchConfigurationDestroy(s *terraform.State) error {
	conn := testAccProvider.autoscalingconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_launch_configuration" {
			continue
		}

		describe, err := conn.DescribeLaunchConfigurations(
			&autoscaling.DescribeLaunchConfigurations{
				Names: []string{rs.Primary.ID},
			})

		if err == nil {
			if len(describe.LaunchConfigurations) != 0 &&
				describe.LaunchConfigurations[0].Name == rs.Primary.ID {
				return fmt.Errorf("Launch Configuration still exists")
			}
		}

		// Verify the error
		providerErr, ok := err.(*autoscaling.Error)
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
		if conf.ImageId != "ami-21f78e11" {
			return fmt.Errorf("Bad image_id: %s", conf.ImageId)
		}

		if conf.Name != "foobar-terraform-test" {
			return fmt.Errorf("Bad name: %s", conf.Name)
		}

		if conf.InstanceType != "t1.micro" {
			return fmt.Errorf("Bad instance_type: %s", conf.InstanceType)
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

		conn := testAccProvider.autoscalingconn

		describeOpts := autoscaling.DescribeLaunchConfigurations{
			Names: []string{rs.Primary.ID},
		}
		describe, err := conn.DescribeLaunchConfigurations(&describeOpts)

		if err != nil {
			return err
		}

		if len(describe.LaunchConfigurations) != 1 ||
			describe.LaunchConfigurations[0].Name != rs.Primary.ID {
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
}
`
