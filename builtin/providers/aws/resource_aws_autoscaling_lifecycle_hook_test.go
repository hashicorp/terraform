package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAutoscalingLifecycleHook_basic(t *testing.T) {
	var hook autoscaling.LifecycleHook

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoscalingLifecycleHookDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoscalingLifecycleHookConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLifecycleHookExists("aws_autoscaling_lifecycle_hook.foobar", &hook),
					resource.TestCheckResourceAttr("aws_autoscaling_lifecycle_hook.foobar", "autoscaling_group_name", "terraform-test-foobar5"),
					resource.TestCheckResourceAttr("aws_autoscaling_lifecycle_hook.foobar", "default_result", "CONTINUE"),
					resource.TestCheckResourceAttr("aws_autoscaling_lifecycle_hook.foobar", "heartbeat_timeout", "2000"),
					resource.TestCheckResourceAttr("aws_autoscaling_lifecycle_hook.foobar", "lifecycle_transition", "autoscaling:EC2_INSTANCE_LAUNCHING"),
				),
			},
		},
	})
}

func testAccCheckLifecycleHookExists(n string, hook *autoscaling.LifecycleHook) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).autoscalingconn
		params := &autoscaling.DescribeLifecycleHooksInput{
			AutoScalingGroupName: aws.String(rs.Primary.Attributes["autoscaling_group_name"]),
			LifecycleHookNames:   []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeLifecycleHooks(params)
		if err != nil {
			return err
		}
		if len(resp.LifecycleHooks) == 0 {
			return fmt.Errorf("LifecycleHook not found")
		}

		return nil
	}
}

func testAccCheckAWSAutoscalingLifecycleHookDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).autoscalingconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_autoscaling_group" {
			continue
		}

		params := autoscaling.DescribeLifecycleHooksInput{
			AutoScalingGroupName: aws.String(rs.Primary.Attributes["autoscaling_group_name"]),
			LifecycleHookNames:   []*string{aws.String(rs.Primary.ID)},
		}

		resp, err := conn.DescribeLifecycleHooks(&params)

		if err == nil {
			if len(resp.LifecycleHooks) != 0 &&
				*resp.LifecycleHooks[0].LifecycleHookName == rs.Primary.ID {
				return fmt.Errorf("Lifecycle Hook Still Exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

var testAccAWSAutoscalingLifecycleHookConfig = fmt.Sprintf(`
resource "aws_launch_configuration" "foobar" {
    name = "terraform-test-foobar5"
    image_id = "ami-21f78e11"
    instance_type = "t1.micro"
}

resource "aws_sqs_queue" "foobar" {
  name = "foobar"
  delay_seconds = 90
  max_message_size = 2048
  message_retention_seconds = 86400
  receive_wait_time_seconds = 10
}

resource "aws_iam_role" "foobar" {
    name = "foobar"
    assume_role_policy = <<EOF
{
  "Version" : "2012-10-17",
  "Statement": [ {
    "Effect": "Allow",
    "Principal": {"AWS": "*"},
    "Action": [ "sts:AssumeRole" ]
  } ]
}
EOF
}

resource "aws_iam_role_policy" "foobar" {
    name = "foobar"
    role = "${aws_iam_role.foobar.id}"
    policy = <<EOF
{
    "Version" : "2012-10-17",
    "Statement": [ {
      "Effect": "Allow",
      "Action": [
	"sqs:SendMessage",
	"sqs:GetQueueUrl",
	"sns:Publish"
      ],
      "Resource": [
	"${aws_sqs_queue.foobar.arn}"
      ]
    } ]
}
EOF
}


resource "aws_autoscaling_group" "foobar" {
    availability_zones = ["us-west-2a"]
    name = "terraform-test-foobar5"
    max_size = 5
    min_size = 2
    health_check_grace_period = 300
    health_check_type = "ELB"
    force_delete = true
    termination_policies = ["OldestInstance"]
    launch_configuration = "${aws_launch_configuration.foobar.name}"
    tag {
        key = "Foo"
        value = "foo-bar"
        propagate_at_launch = true
    }
}

resource "aws_autoscaling_lifecycle_hook" "foobar" {
    name = "foobar"
    autoscaling_group_name = "${aws_autoscaling_group.foobar.name}"
    default_result = "CONTINUE"
    heartbeat_timeout = 2000
    lifecycle_transition = "autoscaling:EC2_INSTANCE_LAUNCHING"
    notification_metadata = <<EOF
{
  "foo": "bar"
}
EOF
    notification_target_arn = "${aws_sqs_queue.foobar.arn}"
    role_arn = "${aws_iam_role.foobar.arn}"
}
`)
