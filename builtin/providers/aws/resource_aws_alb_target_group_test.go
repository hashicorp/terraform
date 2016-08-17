package aws

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSALBTargetGroup_basic(t *testing.T) {
	var conf elbv2.TargetGroup
	targetGroupName := fmt.Sprintf("test-target-group-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb_target_group.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBTargetGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBTargetGroupConfig_basic(targetGroupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &conf),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "name", targetGroupName),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "port", "443"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "protocol", "HTTPS"),
					resource.TestCheckResourceAttrSet("aws_alb_target_group.test", "vpc_id"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "deregistration_delay", "200"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "stickiness.#", "1"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "stickiness.0.type", "lb_cookie"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "stickiness.0.cookie_duration", "10000"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.#", "1"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.path", "/health"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.interval", "60"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.port", "8081"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.protocol", "HTTP"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.timeout", "3"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.healthy_threshold", "3"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.unhealthy_threshold", "3"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.matcher", "200-299"),
				),
			},
		},
	})
}

func TestAccAWSALBTargetGroup_updateHealthCheck(t *testing.T) {
	var conf elbv2.TargetGroup
	targetGroupName := fmt.Sprintf("test-target-group-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb_target_group.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBTargetGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBTargetGroupConfig_basic(targetGroupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &conf),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "name", targetGroupName),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "port", "443"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "protocol", "HTTPS"),
					resource.TestCheckResourceAttrSet("aws_alb_target_group.test", "vpc_id"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "deregistration_delay", "200"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "stickiness.#", "1"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "stickiness.0.type", "lb_cookie"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "stickiness.0.cookie_duration", "10000"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.#", "1"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.path", "/health"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.interval", "60"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.port", "8081"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.protocol", "HTTP"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.timeout", "3"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.healthy_threshold", "3"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.unhealthy_threshold", "3"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.matcher", "200-299"),
				),
			},
			{
				Config: testAccAWSALBTargetGroupConfig_updateHealthCheck(targetGroupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &conf),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "name", targetGroupName),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "port", "443"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "protocol", "HTTPS"),
					resource.TestCheckResourceAttrSet("aws_alb_target_group.test", "vpc_id"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "deregistration_delay", "200"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "stickiness.#", "1"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "stickiness.0.type", "lb_cookie"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "stickiness.0.cookie_duration", "10000"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.#", "1"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.path", "/health2"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.interval", "30"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.port", "8082"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.protocol", "HTTPS"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.timeout", "4"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.healthy_threshold", "4"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.unhealthy_threshold", "4"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "health_check.0.matcher", "200"),
				),
			},
		},
	})
}

func testAccCheckAWSALBTargetGroupExists(n string, res *elbv2.TargetGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("No Target Group ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).elbv2conn

		describe, err := conn.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{
			TargetGroupArns: []*string{aws.String(rs.Primary.ID)},
		})

		if err != nil {
			return err
		}

		if len(describe.TargetGroups) != 1 ||
			*describe.TargetGroups[0].TargetGroupArn != rs.Primary.ID {
			return errors.New("Target Group not found")
		}

		*res = *describe.TargetGroups[0]
		return nil
	}
}

func testAccCheckAWSALBTargetGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elbv2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_alb_target_group" {
			continue
		}

		describe, err := conn.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{
			TargetGroupArns: []*string{aws.String(rs.Primary.ID)},
		})

		if err == nil {
			if len(describe.TargetGroups) != 0 &&
				*describe.TargetGroups[0].TargetGroupArn == rs.Primary.ID {
				return fmt.Errorf("Target Group %q still exists", rs.Primary.ID)
			}
		}

		// Verify the error
		if isTargetGroupNotFound(err) {
			return nil
		} else {
			return errwrap.Wrapf("Unexpected error checking ALB destroyed: {{err}}", err)
		}
	}

	return nil
}

func testAccAWSALBTargetGroupConfig_basic(targetGroupName string) string {
	return fmt.Sprintf(`resource "aws_alb_target_group" "test" {
  name = "%s"
  port = 443
  protocol = "HTTPS"
  vpc_id = "${aws_vpc.test.id}"

  deregistration_delay = 200

  stickiness {
    type = "lb_cookie"
    cookie_duration = 10000
  }

  health_check {
    path = "/health"
    interval = 60
    port = 8081
    protocol = "HTTP"
    timeout = 3
    healthy_threshold = 3
    unhealthy_threshold = 3
    matcher = "200-299"
  }
}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags {
    TestName = "TestAccAWSALBTargetGroup_basic"
  }
}`, targetGroupName)
}

func testAccAWSALBTargetGroupConfig_updateHealthCheck(targetGroupName string) string {
	return fmt.Sprintf(`resource "aws_alb_target_group" "test" {
  name = "%s"
  port = 443
  protocol = "HTTPS"
  vpc_id = "${aws_vpc.test.id}"

  deregistration_delay = 200

  stickiness {
    type = "lb_cookie"
    cookie_duration = 10000
  }

  health_check {
    path = "/health2"
    interval = 30
    port = 8082
    protocol = "HTTPS"
    timeout = 4
    healthy_threshold = 4
    unhealthy_threshold = 4
    matcher = "200"
  }
}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags {
    TestName = "TestAccAWSALBTargetGroup_basic"
  }
}`, targetGroupName)
}
