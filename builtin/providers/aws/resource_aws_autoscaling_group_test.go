package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/autoscaling"
)

func TestAccAWSAutoScalingGroup_basic(t *testing.T) {
	var group autoscaling.AutoScalingGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoScalingGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutoScalingGroupExists("aws_autoscaling_group.bar", &group),
					testAccCheckAWSAutoScalingGroupAttributes(&group),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "availability_zones.0", "us-west-2a"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "name", "foobar3-terraform-test"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "max_size", "5"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "min_size", "2"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "health_check_grace_period", "300"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "health_check_type", "ELB"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "desired_capacity", "4"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "force_delete", "true"),
				),
			},

			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutoScalingGroupExists("aws_autoscaling_group.bar", &group),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "desired_capacity", "5"),
				),
			},
		},
	})
}

func TestAccAWSAutoScalingGroupWithLoadBalancer(t *testing.T) {
	var group autoscaling.AutoScalingGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoScalingGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfigWithLoadBalancer,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutoScalingGroupExists("aws_autoscaling_group.bar", &group),
					testAccCheckAWSAutoScalingGroupAttributesLoadBalancer(&group),
				),
			},
		},
	})
}
func testAccCheckAWSAutoScalingGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.autoscalingconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_autoscaling_group" {
			continue
		}

		// Try to find the Group
		describeGroups, err := conn.DescribeAutoScalingGroups(
			&autoscaling.DescribeAutoScalingGroups{
				Names: []string{rs.Primary.ID},
			})

		if err == nil {
			if len(describeGroups.AutoScalingGroups) != 0 &&
				describeGroups.AutoScalingGroups[0].Name == rs.Primary.ID {
				return fmt.Errorf("AutoScaling Group still exists")
			}
		}

		// Verify the error
		ec2err, ok := err.(*autoscaling.Error)
		if !ok {
			return err
		}
		if ec2err.Code != "InvalidGroup.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSAutoScalingGroupAttributes(group *autoscaling.AutoScalingGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if group.AvailabilityZones[0] != "us-west-2a" {
			return fmt.Errorf("Bad availability_zones: %s", group.AvailabilityZones[0])
		}

		if group.Name != "foobar3-terraform-test" {
			return fmt.Errorf("Bad name: %s", group.Name)
		}

		if group.MaxSize != 5 {
			return fmt.Errorf("Bad max_size: %d", group.MaxSize)
		}

		if group.MinSize != 2 {
			return fmt.Errorf("Bad max_size: %d", group.MinSize)
		}

		if group.HealthCheckType != "ELB" {
			return fmt.Errorf("Bad health_check_type: %s", group.HealthCheckType)
		}

		if group.HealthCheckGracePeriod != 300 {
			return fmt.Errorf("Bad health_check_grace_period: %d", group.HealthCheckGracePeriod)
		}

		if group.DesiredCapacity != 4 {
			return fmt.Errorf("Bad desired_capacity: %d", group.DesiredCapacity)
		}

		if group.LaunchConfigurationName == "" {
			return fmt.Errorf("Bad launch configuration name: %s", group.LaunchConfigurationName)
		}

		return nil
	}
}

func testAccCheckAWSAutoScalingGroupAttributesLoadBalancer(group *autoscaling.AutoScalingGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if group.LoadBalancerNames[0] != "foobar-terraform-test" {
			return fmt.Errorf("Bad load_balancers: %s", group.LoadBalancerNames[0])
		}

		return nil
	}
}

func testAccCheckAWSAutoScalingGroupExists(n string, group *autoscaling.AutoScalingGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No AutoScaling Group ID is set")
		}

		conn := testAccProvider.autoscalingconn

		describeOpts := autoscaling.DescribeAutoScalingGroups{
			Names: []string{rs.Primary.ID},
		}
		describeGroups, err := conn.DescribeAutoScalingGroups(&describeOpts)

		if err != nil {
			return err
		}

		if len(describeGroups.AutoScalingGroups) != 1 ||
			describeGroups.AutoScalingGroups[0].Name != rs.Primary.ID {
			return fmt.Errorf("AutoScaling Group not found")
		}

		*group = describeGroups.AutoScalingGroups[0]

		return nil
	}
}

const testAccAWSAutoScalingGroupConfig = `
resource "aws_launch_configuration" "foobar" {
  name = "foobarautoscaling-terraform-test"
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-west-2a"]
  name = "foobar3-terraform-test"
  max_size = 5
  min_size = 2
  health_check_grace_period = 300
  health_check_type = "ELB"
  desired_capacity = 4
  force_delete = true

  launch_configuration = "${aws_launch_configuration.foobar.name}"
}
`

const testAccAWSAutoScalingGroupConfigUpdate = `
resource "aws_launch_configuration" "foobar" {
  name = "foobarautoscaling-terraform-test"
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-west-2a"]
  name = "foobar3-terraform-test"
  max_size = 5
  min_size = 2
  health_check_grace_period = 300
  health_check_type = "ELB"
  desired_capacity = 5
  force_delete = true

  launch_configuration = "${aws_launch_configuration.foobar.name}"
}
`

const testAccAWSAutoScalingGroupConfigWithLoadBalancer = `
resource "aws_elb" "bar" {
  name = "foobar-terraform-test"
  availability_zones = ["us-west-2a"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }
}

resource "aws_launch_configuration" "foobar" {
  name = "foobarautoscaling-terraform-test"
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-west-2a"]
  name = "foobar3-terraform-test"
  max_size = 5
  min_size = 2
  health_check_grace_period = 300
  health_check_type = "ELB"
  desired_capacity = 4
  force_delete = true

  launch_configuration = "${aws_launch_configuration.foobar.name}"
  load_balancers = ["${aws_elb.bar.name}"]
}
`
