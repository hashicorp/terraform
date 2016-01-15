package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAutoscalingMetricsCollection_basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoscalingMetricsCollectionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoscalingMetricsCollectionConfig_allMetricsCollected,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutoscalingMetricsCollectionExists("aws_autoscaling_metrics_collection.test"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_metrics_collection.test", "metrics.#", "7"),
				),
			},
		},
	})
}

func TestAccAWSAutoscalingMetricsCollection_update(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoscalingMetricsCollectionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoscalingMetricsCollectionConfig_allMetricsCollected,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutoscalingMetricsCollectionExists("aws_autoscaling_metrics_collection.test"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_metrics_collection.test", "metrics.#", "7"),
				),
			},

			resource.TestStep{
				Config: testAccAWSAutoscalingMetricsCollectionConfig_updatingMetricsCollected,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutoscalingMetricsCollectionExists("aws_autoscaling_metrics_collection.test"),
					resource.TestCheckResourceAttr(
						"aws_autoscaling_metrics_collection.test", "metrics.#", "5"),
				),
			},
		},
	})
}
func testAccCheckAWSAutoscalingMetricsCollectionExists(n string) resource.TestCheckFunc {
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

		if describeGroups.AutoScalingGroups[0].EnabledMetrics == nil {
			return fmt.Errorf("AutoScaling Groups Metrics Collection not found")
		}

		return nil
	}
}

func testAccCheckAWSAutoscalingMetricsCollectionDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).autoscalingconn

	for _, rs := range s.RootModule().Resources {
		// Try to find the Group
		describeGroups, err := conn.DescribeAutoScalingGroups(
			&autoscaling.DescribeAutoScalingGroupsInput{
				AutoScalingGroupNames: []*string{aws.String(rs.Primary.ID)},
			})

		if err == nil {
			if len(describeGroups.AutoScalingGroups) != 0 &&
				*describeGroups.AutoScalingGroups[0].AutoScalingGroupName == rs.Primary.ID {
				return fmt.Errorf("AutoScalingGroup still exists")
			}
		}

		// Verify the error
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidGroup.NotFound" {
			return err
		}
	}

	return nil
}

const testAccAWSAutoscalingMetricsCollectionConfig_allMetricsCollected = `
resource "aws_launch_configuration" "foobar" {
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-west-2a"]
  name = "foobar3-terraform-test"
  max_size = 1
  min_size = 0
  health_check_grace_period = 300
  health_check_type = "EC2"
  desired_capacity = 0
  force_delete = true
  termination_policies = ["OldestInstance","ClosestToNextInstanceHour"]

  launch_configuration = "${aws_launch_configuration.foobar.name}"
}

resource "aws_autoscaling_metrics_collection" "test" {
  autoscaling_group_name = "${aws_autoscaling_group.bar.name}"
  granularity = "1Minute"
  metrics = ["GroupTotalInstances",
  	     "GroupPendingInstances",
  	     "GroupTerminatingInstances",
  	     "GroupDesiredCapacity",
  	     "GroupInServiceInstances",
  	     "GroupMinSize",
  	     "GroupMaxSize"
  ]
}
`

const testAccAWSAutoscalingMetricsCollectionConfig_updatingMetricsCollected = `
resource "aws_launch_configuration" "foobar" {
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-west-2a"]
  name = "foobar3-terraform-test"
  max_size = 1
  min_size = 0
  health_check_grace_period = 300
  health_check_type = "EC2"
  desired_capacity = 0
  force_delete = true
  termination_policies = ["OldestInstance","ClosestToNextInstanceHour"]

  launch_configuration = "${aws_launch_configuration.foobar.name}"
}

resource "aws_autoscaling_metrics_collection" "test" {
  autoscaling_group_name = "${aws_autoscaling_group.bar.name}"
  granularity = "1Minute"
  metrics = ["GroupTotalInstances",
  	     "GroupPendingInstances",
  	     "GroupTerminatingInstances",
  	     "GroupDesiredCapacity",
  	     "GroupMaxSize"
  ]
}
`
