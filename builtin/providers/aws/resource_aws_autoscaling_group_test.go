package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAutoScalingGroup_basic(t *testing.T) {
	var group autoscaling.AutoScalingGroup
	var lc autoscaling.LaunchConfiguration

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
						"aws_autoscaling_group.bar", "availability_zones.2487133097", "us-west-2a"),
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
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "termination_policies.912102603", "OldestInstance"),
				),
			},

			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutoScalingGroupExists("aws_autoscaling_group.bar", &group),
					testAccCheckAWSLaunchConfigurationExists("aws_launch_configuration.new", &lc),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_group.bar", "desired_capacity", "5"),
					testLaunchConfigurationName("aws_autoscaling_group.bar", &lc),
					testAccCheckAutoscalingTags(&group.Tags, "Bar", map[string]interface{}{
						"value":               "bar-foo",
						"propagate_at_launch": true,
					}),
				),
			},
		},
	})
}

func TestAccAWSAutoScalingGroup_tags(t *testing.T) {
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
					testAccCheckAutoscalingTags(&group.Tags, "Foo", map[string]interface{}{
						"value":               "foo-bar",
						"propagate_at_launch": true,
					}),
				),
			},

			resource.TestStep{
				Config: testAccAWSAutoScalingGroupConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutoScalingGroupExists("aws_autoscaling_group.bar", &group),
					testAccCheckAutoscalingTagNotExists(&group.Tags, "Foo"),
					testAccCheckAutoscalingTags(&group.Tags, "Bar", map[string]interface{}{
						"value":               "bar-foo",
						"propagate_at_launch": true,
					}),
				),
			},
		},
	})
}

func TestAccAWSAutoScalingGroup_WithLoadBalancer(t *testing.T) {
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
	conn := testAccProvider.Meta().(*AWSClient).autoscalingconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_autoscaling_group" {
			continue
		}

		// Try to find the Group
		describeGroups, err := conn.DescribeAutoScalingGroups(
			&autoscaling.DescribeAutoScalingGroupsInput{
				AutoScalingGroupNames: []*string{aws.String(rs.Primary.ID)},
			})

		if err == nil {
			if len(describeGroups.AutoScalingGroups) != 0 &&
				*describeGroups.AutoScalingGroups[0].AutoScalingGroupName == rs.Primary.ID {
				return fmt.Errorf("AutoScaling Group still exists")
			}
		}

		// Verify the error
		ec2err, ok := err.(aws.APIError)
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
		if *group.AvailabilityZones[0] != "us-west-2a" {
			return fmt.Errorf("Bad availability_zones: %#v", group.AvailabilityZones[0])
		}

		if *group.AutoScalingGroupName != "foobar3-terraform-test" {
			return fmt.Errorf("Bad name: %s", *group.AutoScalingGroupName)
		}

		if *group.MaxSize != 5 {
			return fmt.Errorf("Bad max_size: %d", *group.MaxSize)
		}

		if *group.MinSize != 2 {
			return fmt.Errorf("Bad max_size: %d", *group.MinSize)
		}

		if *group.HealthCheckType != "ELB" {
			return fmt.Errorf("Bad health_check_type,\nexpected: %s\ngot: %s", "ELB", *group.HealthCheckType)
		}

		if *group.HealthCheckGracePeriod != 300 {
			return fmt.Errorf("Bad health_check_grace_period: %d", *group.HealthCheckGracePeriod)
		}

		if *group.DesiredCapacity != 4 {
			return fmt.Errorf("Bad desired_capacity: %d", *group.DesiredCapacity)
		}

		if *group.LaunchConfigurationName == "" {
			return fmt.Errorf("Bad launch configuration name: %s", *group.LaunchConfigurationName)
		}

		t := &autoscaling.TagDescription{
			Key:               aws.String("Foo"),
			Value:             aws.String("foo-bar"),
			PropagateAtLaunch: aws.Boolean(true),
			ResourceType:      aws.String("auto-scaling-group"),
			ResourceID:        group.AutoScalingGroupName,
		}

		if !reflect.DeepEqual(group.Tags[0], t) {
			return fmt.Errorf(
				"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
				group.Tags[0],
				t)
		}

		return nil
	}
}

func testAccCheckAWSAutoScalingGroupAttributesLoadBalancer(group *autoscaling.AutoScalingGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *group.LoadBalancerNames[0] != "foobar-terraform-test" {
			return fmt.Errorf("Bad load_balancers: %#v", group.LoadBalancerNames[0])
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

		conn := testAccProvider.Meta().(*AWSClient).autoscalingconn

		describeGroups, err := conn.DescribeAutoScalingGroups(
			&autoscaling.DescribeAutoScalingGroupsInput{
				AutoScalingGroupNames: []*string{aws.String(rs.Primary.ID)},
			})

		if err != nil {
			return err
		}

		if len(describeGroups.AutoScalingGroups) != 1 ||
			*describeGroups.AutoScalingGroups[0].AutoScalingGroupName != rs.Primary.ID {
			return fmt.Errorf("AutoScaling Group not found")
		}

		*group = *describeGroups.AutoScalingGroups[0]

		return nil
	}
}

func testLaunchConfigurationName(n string, lc *autoscaling.LaunchConfiguration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if *lc.LaunchConfigurationName != rs.Primary.Attributes["launch_configuration"] {
			return fmt.Errorf("Launch configuration names do not match")
		}

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
  termination_policies = ["OldestInstance"]

  launch_configuration = "${aws_launch_configuration.foobar.name}"

  tag {
    key = "Foo"
    value = "foo-bar"
    propagate_at_launch = true
  }
}
`

const testAccAWSAutoScalingGroupConfigUpdate = `
resource "aws_launch_configuration" "foobar" {
  name = "foobarautoscaling-terraform-test"
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_launch_configuration" "new" {
  name = "foobarautoscaling-terraform-test-new"
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

  launch_configuration = "${aws_launch_configuration.new.name}"

  tag {
    key = "Bar"
    value = "bar-foo"
    propagate_at_launch = true
  }
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
