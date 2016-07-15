package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/applicationautoscaling"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAppautoscalingPolicy_basic(t *testing.T) {
	var policy applicationautoscaling.ScalingPolicy

	name := fmt.Sprintf("terraform-test-foobar-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAppautoscalingPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAppautoscalingPolicyConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAppautoscalingPolicyExists("aws_appautoscaling_policy.foobar_simple", &policy),
					resource.TestCheckResourceAttr("aws_appautoscaling_policy.foobar_simple", "adjustment_type", "ChangeInCapacity"),
					resource.TestCheckResourceAttr("aws_appautoscaling_policy.foobar_simple", "policy_type", "StepScaling"),
					resource.TestCheckResourceAttr("aws_appautoscaling_policy.foobar_simple", "cooldown", "60"),
					resource.TestCheckResourceAttr("aws_appautoscaling_policy.foobar_simple", "name", "foobar_simple"),
					resource.TestCheckResourceAttr("aws_appautoscaling_policy.foobar_simple", "scaling_adjustment", "2"),
					resource.TestCheckResourceAttr("aws_appautoscaling_policy.foobar_simple", "resource_id", "service/default/foobar_simple"),
					resource.TestCheckResourceAttr("aws_appautoscaling_policy.foobar_simple", "service_namespace", "ecs"),
					resource.TestCheckResourceAttr("aws_appautoscaling_policy.foobar_simple", "scalable_dimension", "ecs:service:DesiredCount"),
				),
			},
		},
	})
}

func testAccCheckAWSAppautoscalingPolicyExists(n string, policy *applicationautoscaling.ScalingPolicy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).appautoscalingconn
		params := &applicationautoscaling.DescribeScalingPoliciesInput{
			ServiceNamespace: aws.String(rs.Primary.Attributes["service_namespace"]),
			PolicyNames:      []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeScalingPolicies(params)
		if err != nil {
			return err
		}
		if len(resp.ScalingPolicies) == 0 {
			return fmt.Errorf("ScalingPolicy %s not found", rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckAWSAppautoscalingPolicyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).appautoscalingconn

	for _, rs := range s.RootModule().Resources {
		params := applicationautoscaling.DescribeScalingPoliciesInput{
			ServiceNamespace: aws.String(rs.Primary.Attributes["service_namespace"]),
			PolicyNames:      []*string{aws.String(rs.Primary.ID)},
		}

		resp, err := conn.DescribeScalingPolicies(&params)

		if err == nil {
			if len(resp.ScalingPolicies) != 0 &&
				*resp.ScalingPolicies[0].PolicyName == rs.Primary.ID {
				return fmt.Errorf("Application autoscaling policy still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccAWSAppautoscalingPolicyConfig(name string) string {
	return fmt.Sprintf(`
resource "aws_appautoscaling_policy" "up" {
	name = "%s"
	service_namespace = "ecs"
	resource_id = "service/default/foobar"
	scalable_dimension = "ecs:service:DesiredCount"

	adjustment_type = "ExactCapacity"
	cooldown = 60
	metric_aggregation_type = "Maximum"
	min_adjustment_magnitude = 1

	step_adjustment {
		scaling_adjustment = 1
	}
}
resource "aws_appautoscaling_policy" "down" {
	name = "%s"
	service_namespace = "ecs"
	resource_id = "service/default/foobar"
	scalable_dimension = "ecs:service:DesiredCount"

	adjustment_type = "ExactCapacity"
	cooldown = 60
	metric_aggregation_type = "Maximum"
	min_adjustment_magnitude = 1

	step_adjustment {
		scaling_adjustment = -1
	}
}
`, name, name)
}
