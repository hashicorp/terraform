package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/service/route53"
)

func TestAccAWSRoute53HealthCheck_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_route53_health_check.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckRoute53HealthCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53HealthCheckConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53HealthCheckExists("aws_route53_health_check.foo"),
					resource.TestCheckResourceAttr(
						"aws_route53_health_check.foo", "measure_latency", "true"),
					resource.TestCheckResourceAttr(
						"aws_route53_health_check.foo", "invert_healthcheck", "true"),
				),
			},
			resource.TestStep{
				Config: testAccRoute53HealthCheckConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53HealthCheckExists("aws_route53_health_check.foo"),
					resource.TestCheckResourceAttr(
						"aws_route53_health_check.foo", "failure_threshold", "5"),
					resource.TestCheckResourceAttr(
						"aws_route53_health_check.foo", "invert_healthcheck", "false"),
				),
			},
		},
	})
}

func TestAccAWSRoute53HealthCheck_withChildHealthChecks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53HealthCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53HealthCheckConfig_withChildHealthChecks,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53HealthCheckExists("aws_route53_health_check.foo"),
				),
			},
		},
	})
}

func TestAccAWSRoute53HealthCheck_IpConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53HealthCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53HealthCheckIpConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53HealthCheckExists("aws_route53_health_check.bar"),
				),
			},
		},
	})
}

func testAccCheckRoute53HealthCheckDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).r53conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_route53_health_check" {
			continue
		}

		lopts := &route53.ListHealthChecksInput{}
		resp, err := conn.ListHealthChecks(lopts)
		if err != nil {
			return err
		}
		if len(resp.HealthChecks) == 0 {
			return nil
		}

		for _, check := range resp.HealthChecks {
			if *check.Id == rs.Primary.ID {
				return fmt.Errorf("Record still exists: %#v", check)
			}

		}

	}
	return nil
}

func testAccCheckRoute53HealthCheckExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).r53conn

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		fmt.Print(rs.Primary.ID)

		if rs.Primary.ID == "" {
			return fmt.Errorf("No health check ID is set")
		}

		lopts := &route53.ListHealthChecksInput{}
		resp, err := conn.ListHealthChecks(lopts)
		if err != nil {
			return err
		}
		if len(resp.HealthChecks) == 0 {
			return fmt.Errorf("Health Check does not exist")
		}

		for _, check := range resp.HealthChecks {
			if *check.Id == rs.Primary.ID {
				return nil
			}

		}
		return fmt.Errorf("Health Check does not exist")
	}
}

func testUpdateHappened(n string) resource.TestCheckFunc {
	return nil
}

const testAccRoute53HealthCheckConfig = `
resource "aws_route53_health_check" "foo" {
  fqdn = "dev.notexample.com"
  port = 80
  type = "HTTP"
  resource_path = "/"
  failure_threshold = "2"
  request_interval = "30"
  measure_latency = true
  invert_healthcheck = true

  tags = {
    Name = "tf-test-health-check"
   }
}
`

const testAccRoute53HealthCheckConfigUpdate = `
resource "aws_route53_health_check" "foo" {
  fqdn = "dev.notexample.com"
  port = 80
  type = "HTTP"
  resource_path = "/"
  failure_threshold = "5"
  request_interval = "30"
  measure_latency = true
  invert_healthcheck = false

  tags = {
    Name = "tf-test-health-check"
   }
}
`

const testAccRoute53HealthCheckIpConfig = `
resource "aws_route53_health_check" "bar" {
  ip_address = "1.2.3.4"
  port = 80
  type = "HTTP"
  resource_path = "/"
  failure_threshold = "2"
  request_interval = "30"

  tags = {
    Name = "tf-test-health-check"
   }
}
`

const testAccRoute53HealthCheckConfig_withChildHealthChecks = `
resource "aws_route53_health_check" "child1" {
  fqdn = "child1.notexample.com"
  port = 80
  type = "HTTP"
  resource_path = "/"
  failure_threshold = "2"
  request_interval = "30"
}

resource "aws_route53_health_check" "foo" {
  type = "CALCULATED"
  child_health_threshold = 1
  child_healthchecks = ["${aws_route53_health_check.child1.id}"]

  tags = {
    Name = "tf-test-calculated-health-check"
   }
}
`
