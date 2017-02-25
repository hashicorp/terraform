package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAwsAutoscalingAttachment_elb(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAutoscalingAttachment_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutocalingElbAttachmentExists("aws_autoscaling_group.asg", 0),
				),
			},
			// Add in one association
			{
				Config: testAccAWSAutoscalingAttachment_associated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutocalingElbAttachmentExists("aws_autoscaling_group.asg", 1),
				),
			},
			// Test adding a 2nd
			{
				Config: testAccAWSAutoscalingAttachment_double_associated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutocalingElbAttachmentExists("aws_autoscaling_group.asg", 2),
				),
			},
			// Now remove that newest one
			{
				Config: testAccAWSAutoscalingAttachment_associated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutocalingElbAttachmentExists("aws_autoscaling_group.asg", 1),
				),
			},
			// Now remove them both
			{
				Config: testAccAWSAutoscalingAttachment_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutocalingElbAttachmentExists("aws_autoscaling_group.asg", 0),
				),
			},
		},
	})
}

func TestAccAwsAutoscalingAttachment_instances(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAutoscalingAttachment_instances,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutocalingInstanceAttachmentExists("aws_autoscaling_group.asg", 0),
				),
			},
			// Add in one association
			{
				Config: testAccAWSAutoscalingInstanceAttachment_associated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutocalingInstanceAttachmentExists("aws_autoscaling_group.asg", 1),
				),
			},
			// Test adding a 2nd
			{
				Config: testAccAWSAutoscalingInstanceAttachment_double_associated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutocalingInstanceAttachmentExists("aws_autoscaling_group.asg", 2),
				),
			},
			// Now remove them both
			{
				Config: testAccAWSAutoscalingAttachment_instances,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutocalingInstanceAttachmentExists("aws_autoscaling_group.asg", 0),
				),
			},
		},
	})
}

func testAccCheckAWSAutocalingElbAttachmentExists(asgname string, loadBalancerCount int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[asgname]
		if !ok {
			return fmt.Errorf("Not found: %s", asgname)
		}

		conn := testAccProvider.Meta().(*AWSClient).autoscalingconn
		asg := rs.Primary.ID

		actual, err := conn.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{aws.String(asg)},
		})

		if err != nil {
			return fmt.Errorf("Recieved an error when attempting to load %s:  %s", asg, err)
		}

		if loadBalancerCount != len(actual.AutoScalingGroups[0].LoadBalancerNames) {
			return fmt.Errorf("Error: ASG has the wrong number of load balancers associated.  Expected [%d] but got [%d]", loadBalancerCount, len(actual.AutoScalingGroups[0].LoadBalancerNames))
		}

		return nil
	}
}

func testAccCheckAWSAutocalingInstanceAttachmentExists(asgname string, expectedInstanceCount int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[asgname]
		if !ok {
			return fmt.Errorf("Not found: %s", asgname)
		}

		conn := testAccProvider.Meta().(*AWSClient).autoscalingconn
		asg := rs.Primary.ID

		actual, err := conn.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{aws.String(asg)},
		})

		if err != nil {
			return fmt.Errorf("Recieved an error when attempting to load %s:  %s", asg, err)
		}

		if expectedInstanceCount != len(actual.AutoScalingGroups[0].Instances) {
			return fmt.Errorf("Error: ASG has the wrong number of instances associated.  Expected [%d] but got [%d]", expectedInstanceCount, len(actual.AutoScalingGroups[0].Instances))
		}

		return nil
	}
}

const testAccAWSAutoscalingAttachment_instances = `
resource "aws_launch_configuration" "as_conf" {
    name_prefix = "test_config_"
    image_id = "ami-f34032c3"
    instance_type = "t1.micro"
}

resource "aws_instance" "foo" {
    ami = "ami-f34032c3"
    availability_zone = "us-west-2a"
    instance_type = "t1.micro"
}

resource "aws_instance" "bar" {
    ami = "ami-f34032c3"
    availability_zone = "us-west-2b"
    instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "asg" {
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]
  max_size = 1
  min_size = 0
  desired_capacity = 0
  health_check_grace_period = 300
  force_delete = true
  launch_configuration = "${aws_launch_configuration.as_conf.name}"

  tag {
    key = "Name"
    value = "terraform-asg-lg-assoc-test"
    propagate_at_launch = true
  }
}`

const testAccAWSAutoscalingAttachment_basic = `
resource "aws_elb" "foo" {
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port     = 8000
    instance_protocol = "http"
    lb_port           = 80
    lb_protocol       = "http"
  }
}

resource "aws_elb" "bar" {
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port     = 8000
    instance_protocol = "http"
    lb_port           = 80
    lb_protocol       = "http"
  }
}

resource "aws_launch_configuration" "as_conf" {
    name_prefix = "test_config_"
    image_id = "ami-f34032c3"
    instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "asg" {
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]
  max_size = 1
  min_size = 0
  desired_capacity = 0
  health_check_grace_period = 300
  force_delete = true
  launch_configuration = "${aws_launch_configuration.as_conf.name}"

  tag {
    key = "Name"
    value = "terraform-asg-lg-assoc-test"
    propagate_at_launch = true
  }
}

`

const testAccAWSAutoscalingAttachment_associated = testAccAWSAutoscalingAttachment_basic + `
resource "aws_autoscaling_attachment" "asg_attachment_foo" {
  autoscaling_group_name = "${aws_autoscaling_group.asg.id}"
  elb                    = "${aws_elb.foo.id}"
}

`

const testAccAWSAutoscalingAttachment_double_associated = testAccAWSAutoscalingAttachment_associated + `
resource "aws_autoscaling_attachment" "asg_attachment_bar" {
  autoscaling_group_name = "${aws_autoscaling_group.asg.id}"
  elb                    = "${aws_elb.bar.id}"
}

`

const testAccAWSAutoscalingInstanceAttachment_associated = testAccAWSAutoscalingAttachment_instances + `
resource "aws_autoscaling_attachment" "asg_attachment_foo" {
  autoscaling_group_name = "${aws_autoscaling_group.asg.id}"
  instance_ids           = ["${aws_instance.foo.id}"]
}

`

const testAccAWSAutoscalingInstanceAttachment_double_associated = testAccAWSAutoscalingInstanceAttachment_associated + `
resource "aws_autoscaling_attachment" "asg_attachment_foo" {
  autoscaling_group_name = "${aws_autoscaling_group.asg.id}"
  instance_ids           = ["${aws_instance.foo.id}", "${aws_instance.bar.id}"]
}

`
