package aws

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestALBTargetGroupCloudwatchSuffixFromARN(t *testing.T) {
	cases := []struct {
		name   string
		arn    *string
		suffix string
	}{
		{
			name:   "valid suffix",
			arn:    aws.String(`arn:aws:elasticloadbalancing:us-east-1:123456:targetgroup/my-targets/73e2d6bc24d8a067`),
			suffix: `targetgroup/my-targets/73e2d6bc24d8a067`,
		},
		{
			name:   "no suffix",
			arn:    aws.String(`arn:aws:elasticloadbalancing:us-east-1:123456:targetgroup`),
			suffix: ``,
		},
		{
			name:   "nil ARN",
			arn:    nil,
			suffix: ``,
		},
	}

	for _, tc := range cases {
		actual := albTargetGroupSuffixFromARN(tc.arn)
		if actual != tc.suffix {
			t.Fatalf("bad suffix: %q\nExpected: %s\n     Got: %s", tc.name, tc.suffix, actual)
		}
	}
}

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
					resource.TestCheckResourceAttrSet("aws_alb_target_group.test", "arn"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "name", targetGroupName),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "port", "443"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "protocol", "HTTPS"),
					resource.TestCheckResourceAttrSet("aws_alb_target_group.test", "vpc_id"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "deregistration_delay", "200"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "stickiness.#", "1"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "stickiness.0.enabled", "true"),
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
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "tags.%", "1"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "tags.TestName", "TestAccAWSALBTargetGroup_basic"),
				),
			},
		},
	})
}

func TestAccAWSALBTargetGroup_namePrefix(t *testing.T) {
	var conf elbv2.TargetGroup

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb_target_group.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBTargetGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBTargetGroupConfig_namePrefix,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &conf),
					resource.TestMatchResourceAttr("aws_alb_target_group.test", "name", regexp.MustCompile("^tf-")),
				),
			},
		},
	})
}

func TestAccAWSALBTargetGroup_generatedName(t *testing.T) {
	var conf elbv2.TargetGroup

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb_target_group.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBTargetGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBTargetGroupConfig_generatedName,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &conf),
				),
			},
		},
	})
}

func TestAccAWSALBTargetGroup_changeNameForceNew(t *testing.T) {
	var before, after elbv2.TargetGroup
	targetGroupNameBefore := fmt.Sprintf("test-target-group-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	targetGroupNameAfter := fmt.Sprintf("test-target-group-%s", acctest.RandStringFromCharSet(4, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb_target_group.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBTargetGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBTargetGroupConfig_basic(targetGroupNameBefore),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &before),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "name", targetGroupNameBefore),
				),
			},
			{
				Config: testAccAWSALBTargetGroupConfig_basic(targetGroupNameAfter),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &after),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "name", targetGroupNameAfter),
				),
			},
		},
	})
}

func TestAccAWSALBTargetGroup_changeProtocolForceNew(t *testing.T) {
	var before, after elbv2.TargetGroup
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
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &before),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "protocol", "HTTPS"),
				),
			},
			{
				Config: testAccAWSALBTargetGroupConfig_updatedProtocol(targetGroupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &after),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "protocol", "HTTP"),
				),
			},
		},
	})
}

func TestAccAWSALBTargetGroup_changePortForceNew(t *testing.T) {
	var before, after elbv2.TargetGroup
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
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &before),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "port", "443"),
				),
			},
			{
				Config: testAccAWSALBTargetGroupConfig_updatedPort(targetGroupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &after),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "port", "442"),
				),
			},
		},
	})
}

func TestAccAWSALBTargetGroup_changeVpcForceNew(t *testing.T) {
	var before, after elbv2.TargetGroup
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
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &before),
				),
			},
			{
				Config: testAccAWSALBTargetGroupConfig_updatedVpc(targetGroupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &after),
				),
			},
		},
	})
}

func TestAccAWSALBTargetGroup_tags(t *testing.T) {
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
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "tags.%", "1"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "tags.TestName", "TestAccAWSALBTargetGroup_basic"),
				),
			},
			{
				Config: testAccAWSALBTargetGroupConfig_updateTags(targetGroupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &conf),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "tags.%", "2"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "tags.Environment", "Production"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "tags.Type", "ALB Target Group"),
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
					resource.TestCheckResourceAttrSet("aws_alb_target_group.test", "arn"),
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
					resource.TestCheckResourceAttrSet("aws_alb_target_group.test", "arn"),
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

func TestAccAWSALBTargetGroup_updateSticknessEnabled(t *testing.T) {
	var conf elbv2.TargetGroup
	targetGroupName := fmt.Sprintf("test-target-group-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb_target_group.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBTargetGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBTargetGroupConfig_stickiness(targetGroupName, false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &conf),
					resource.TestCheckResourceAttrSet("aws_alb_target_group.test", "arn"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "name", targetGroupName),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "port", "443"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "protocol", "HTTPS"),
					resource.TestCheckResourceAttrSet("aws_alb_target_group.test", "vpc_id"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "deregistration_delay", "200"),
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
			{
				Config: testAccAWSALBTargetGroupConfig_stickiness(targetGroupName, true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &conf),
					resource.TestCheckResourceAttrSet("aws_alb_target_group.test", "arn"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "name", targetGroupName),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "port", "443"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "protocol", "HTTPS"),
					resource.TestCheckResourceAttrSet("aws_alb_target_group.test", "vpc_id"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "deregistration_delay", "200"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "stickiness.#", "1"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "stickiness.0.enabled", "true"),
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
			{
				Config: testAccAWSALBTargetGroupConfig_stickiness(targetGroupName, true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBTargetGroupExists("aws_alb_target_group.test", &conf),
					resource.TestCheckResourceAttrSet("aws_alb_target_group.test", "arn"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "name", targetGroupName),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "port", "443"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "protocol", "HTTPS"),
					resource.TestCheckResourceAttrSet("aws_alb_target_group.test", "vpc_id"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "deregistration_delay", "200"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "stickiness.#", "1"),
					resource.TestCheckResourceAttr("aws_alb_target_group.test", "stickiness.0.enabled", "false"),
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

  tags {
    TestName = "TestAccAWSALBTargetGroup_basic"
  }
}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags {
    TestName = "TestAccAWSALBTargetGroup_basic"
  }
}`, targetGroupName)
}

func testAccAWSALBTargetGroupConfig_updatedPort(targetGroupName string) string {
	return fmt.Sprintf(`resource "aws_alb_target_group" "test" {
  name = "%s"
  port = 442
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

  tags {
    TestName = "TestAccAWSALBTargetGroup_basic"
  }
}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags {
    TestName = "TestAccAWSALBTargetGroup_basic"
  }
}`, targetGroupName)
}

func testAccAWSALBTargetGroupConfig_updatedProtocol(targetGroupName string) string {
	return fmt.Sprintf(`resource "aws_alb_target_group" "test" {
  name = "%s"
  port = 443
  protocol = "HTTP"
  vpc_id = "${aws_vpc.test2.id}"

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

  tags {
    TestName = "TestAccAWSALBTargetGroup_basic"
  }
}

resource "aws_vpc" "test2" {
  cidr_block = "10.10.0.0/16"

  tags {
    TestName = "TestAccAWSALBTargetGroup_basic"
  }
}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags {
    TestName = "TestAccAWSALBTargetGroup_basic"
  }
}`, targetGroupName)
}

func testAccAWSALBTargetGroupConfig_updatedVpc(targetGroupName string) string {
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

  tags {
    TestName = "TestAccAWSALBTargetGroup_basic"
  }
}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags {
    TestName = "TestAccAWSALBTargetGroup_basic"
  }
}`, targetGroupName)
}

func testAccAWSALBTargetGroupConfig_updateTags(targetGroupName string) string {
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

  tags {
    Environment = "Production"
    Type = "ALB Target Group"
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

func testAccAWSALBTargetGroupConfig_stickiness(targetGroupName string, addStickinessBlock bool, enabled bool) string {
	var stickinessBlock string

	if addStickinessBlock {
		stickinessBlock = fmt.Sprintf(`stickiness {
	    enabled = "%t"
	    type = "lb_cookie"
	    cookie_duration = 10000
	  }`, enabled)
	}

	return fmt.Sprintf(`resource "aws_alb_target_group" "test" {
  name = "%s"
  port = 443
  protocol = "HTTPS"
  vpc_id = "${aws_vpc.test.id}"

  deregistration_delay = 200

  %s

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
    TestName = "TestAccAWSALBTargetGroup_stickiness"
  }
}`, targetGroupName, stickinessBlock)
}

const testAccAWSALBTargetGroupConfig_namePrefix = `
resource "aws_alb_target_group" "test" {
  name_prefix = "tf-"
  port = 80
  protocol = "HTTP"
  vpc_id = "${aws_vpc.test.id}"
}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"
	tags {
		Name = "testAccAWSALBTargetGroupConfig_namePrefix"
	}
}
`

const testAccAWSALBTargetGroupConfig_generatedName = `
resource "aws_alb_target_group" "test" {
  port = 80
  protocol = "HTTP"
  vpc_id = "${aws_vpc.test.id}"
}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"
	tags {
		Name = "testAccAWSALBTargetGroupConfig_generatedName"
	}
}
`
