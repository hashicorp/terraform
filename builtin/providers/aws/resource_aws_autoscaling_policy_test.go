package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAutoscalingPolicy_basic(t *testing.T) {
	var policy autoscaling.ScalingPolicy

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoscalingPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoscalingPolicyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalingPolicyExists("aws_autoscaling_policy.foobar_simple", &policy),
					resource.TestCheckResourceAttr("aws_autoscaling_policy.foobar_simple", "adjustment_type", "ChangeInCapacity"),
					resource.TestCheckResourceAttr("aws_autoscaling_policy.foobar_simple", "policy_type", "SimpleScaling"),
					resource.TestCheckResourceAttr("aws_autoscaling_policy.foobar_simple", "cooldown", "300"),
					resource.TestCheckResourceAttr("aws_autoscaling_policy.foobar_simple", "name", "foobar_simple"),
					resource.TestCheckResourceAttr("aws_autoscaling_policy.foobar_simple", "scaling_adjustment", "2"),
					resource.TestCheckResourceAttr("aws_autoscaling_policy.foobar_simple", "autoscaling_group_name", "terraform-test-foobar5"),
					testAccCheckScalingPolicyExists("aws_autoscaling_policy.foobar_step", &policy),
					resource.TestCheckResourceAttr("aws_autoscaling_policy.foobar_step", "adjustment_type", "ChangeInCapacity"),
					resource.TestCheckResourceAttr("aws_autoscaling_policy.foobar_step", "policy_type", "StepScaling"),
					resource.TestCheckResourceAttr("aws_autoscaling_policy.foobar_step", "name", "foobar_step"),
					resource.TestCheckResourceAttr("aws_autoscaling_policy.foobar_step", "metric_aggregation_type", "Minimum"),
					resource.TestCheckResourceAttr("aws_autoscaling_policy.foobar_step", "estimated_instance_warmup", "200"),
					resource.TestCheckResourceAttr("aws_autoscaling_policy.foobar_step", "autoscaling_group_name", "terraform-test-foobar5"),
				),
			},
		},
	})
}

func TestAccAWSAutoscalingPolicy_upgrade(t *testing.T) {
	var policy autoscaling.ScalingPolicy

	name := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoscalingPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoscalingPolicyConfig_upgrade_614(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalingPolicyExists("aws_autoscaling_policy.foobar_simple", &policy),
					resource.TestCheckResourceAttr("aws_autoscaling_policy.foobar_simple", "min_adjustment_step", "0"),
					resource.TestCheckResourceAttr("aws_autoscaling_policy.foobar_simple", "min_adjustment_magnitude", "1"),
				),
				ExpectNonEmptyPlan: true,
			},

			resource.TestStep{
				Config: testAccAWSAutoscalingPolicyConfig_upgrade_615(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalingPolicyExists("aws_autoscaling_policy.foobar_simple", &policy),
					resource.TestCheckResourceAttr("aws_autoscaling_policy.foobar_simple", "min_adjustment_step", "0"),
					resource.TestCheckResourceAttr("aws_autoscaling_policy.foobar_simple", "min_adjustment_magnitude", "1"),
				),
			},
		},
	})
}

func testAccCheckScalingPolicyExists(n string, policy *autoscaling.ScalingPolicy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).autoscalingconn
		params := &autoscaling.DescribePoliciesInput{
			AutoScalingGroupName: aws.String(rs.Primary.Attributes["autoscaling_group_name"]),
			PolicyNames:          []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribePolicies(params)
		if err != nil {
			return err
		}
		if len(resp.ScalingPolicies) == 0 {
			return fmt.Errorf("ScalingPolicy not found")
		}

		return nil
	}
}

func testAccCheckAWSAutoscalingPolicyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).autoscalingconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_autoscaling_group" {
			continue
		}

		params := autoscaling.DescribePoliciesInput{
			AutoScalingGroupName: aws.String(rs.Primary.Attributes["autoscaling_group_name"]),
			PolicyNames:          []*string{aws.String(rs.Primary.ID)},
		}

		resp, err := conn.DescribePolicies(&params)

		if err == nil {
			if len(resp.ScalingPolicies) != 0 &&
				*resp.ScalingPolicies[0].PolicyName == rs.Primary.ID {
				return fmt.Errorf("Scaling Policy Still Exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

var testAccAWSAutoscalingPolicyConfig = fmt.Sprintf(`
resource "aws_launch_configuration" "foobar" {
	name = "terraform-test-foobar5"
	image_id = "ami-21f78e11"
	instance_type = "t1.micro"
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

resource "aws_autoscaling_policy" "foobar_simple" {
	name = "foobar_simple"
	adjustment_type = "ChangeInCapacity"
	cooldown = 300
	policy_type = "SimpleScaling"
	scaling_adjustment = 2
	autoscaling_group_name = "${aws_autoscaling_group.foobar.name}"
}

resource "aws_autoscaling_policy" "foobar_step" {
	name = "foobar_step"
	adjustment_type = "ChangeInCapacity"
	policy_type = "StepScaling"
	estimated_instance_warmup = 200
	metric_aggregation_type = "Minimum"
	step_adjustment {
		scaling_adjustment = 1
		metric_interval_lower_bound = 2.0
	}
	autoscaling_group_name = "${aws_autoscaling_group.foobar.name}"
}
`)

func testAccAWSAutoscalingPolicyConfig_upgrade_614(name string) string {
	return fmt.Sprintf(`
resource "aws_launch_configuration" "foobar" {
  name          = "tf-test-%s"
  image_id      = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "foobar" {
  availability_zones        = ["us-west-2a"]
  name                      = "terraform-test-%s"
  max_size                  = 5
  min_size                  = 1
  health_check_grace_period = 300
  health_check_type         = "ELB"
  force_delete              = true
  termination_policies      = ["OldestInstance"]
  launch_configuration      = "${aws_launch_configuration.foobar.name}"

  tag {
    key                 = "Foo"
    value               = "foo-bar"
    propagate_at_launch = true
  }
}

resource "aws_autoscaling_policy" "foobar_simple" {
  name                   = "foobar_simple_%s"
  adjustment_type        = "PercentChangeInCapacity"
  cooldown               = 300
  policy_type            = "SimpleScaling"
  scaling_adjustment     = 2
  min_adjustment_step    = 1
  autoscaling_group_name = "${aws_autoscaling_group.foobar.name}"
}
`, name, name, name)
}

func testAccAWSAutoscalingPolicyConfig_upgrade_615(name string) string {
	return fmt.Sprintf(`
resource "aws_launch_configuration" "foobar" {
  name          = "tf-test-%s"
  image_id      = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "foobar" {
  availability_zones        = ["us-west-2a"]
  name                      = "terraform-test-%s"
  max_size                  = 5
  min_size                  = 1
  health_check_grace_period = 300
  health_check_type         = "ELB"
  force_delete              = true
  termination_policies      = ["OldestInstance"]
  launch_configuration      = "${aws_launch_configuration.foobar.name}"

  tag {
    key                 = "Foo"
    value               = "foo-bar"
    propagate_at_launch = true
  }
}

resource "aws_autoscaling_policy" "foobar_simple" {
  name                     = "foobar_simple_%s"
  adjustment_type          = "PercentChangeInCapacity"
  cooldown                 = 300
  policy_type              = "SimpleScaling"
  scaling_adjustment       = 2
  min_adjustment_magnitude = 1
  autoscaling_group_name   = "${aws_autoscaling_group.foobar.name}"
}
`, name, name, name)
}
