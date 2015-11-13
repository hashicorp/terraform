package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAwsLBCookieStickinessPolicy_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBCookieStickinessPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLBCookieStickinessPolicyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBCookieStickinessPolicy(
						"aws_elb.lb",
						"aws_lb_cookie_stickiness_policy.foo",
					),
				),
			},
			resource.TestStep{
				Config: testAccLBCookieStickinessPolicyConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBCookieStickinessPolicy(
						"aws_elb.lb",
						"aws_lb_cookie_stickiness_policy.foo",
					),
				),
			},
		},
	})
}

func testAccCheckLBCookieStickinessPolicyDestroy(s *terraform.State) error {
	if len(s.RootModule().Resources) > 0 {
		return fmt.Errorf("Expected all resources to be gone, but found: %#v", s.RootModule().Resources)
	}

	return nil
}

func testAccCheckLBCookieStickinessPolicy(elbResource string, policyResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[elbResource]
		if !ok {
			return fmt.Errorf("Not found: %s", elbResource)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		policy, ok := s.RootModule().Resources[policyResource]
		if !ok {
			return fmt.Errorf("Not found: %s", policyResource)
		}

		elbconn := testAccProvider.Meta().(*AWSClient).elbconn
		elbName, _, policyName := resourceAwsLBCookieStickinessPolicyParseId(policy.Primary.ID)
		_, err := elbconn.DescribeLoadBalancerPolicies(&elb.DescribeLoadBalancerPoliciesInput{
			LoadBalancerName: aws.String(elbName),
			PolicyNames:      []*string{aws.String(policyName)},
		})

		if err != nil {
			return err
		}

		return nil
	}
}

const testAccLBCookieStickinessPolicyConfig = `
resource "aws_elb" "lb" {
	name = "test-lb"
	availability_zones = ["us-west-2a"]
	listener {
		instance_port = 8000
		instance_protocol = "http"
		lb_port = 80
		lb_protocol = "http"
	}
}

resource "aws_lb_cookie_stickiness_policy" "foo" {
	name = "foo-policy"
	load_balancer = "${aws_elb.lb.id}"
	lb_port = 80
}
`

// Sets the cookie_expiration_period to 300s.
const testAccLBCookieStickinessPolicyConfigUpdate = `
resource "aws_elb" "lb" {
	name = "test-lb"
	availability_zones = ["us-west-2a"]
	listener {
		instance_port = 8000
		instance_protocol = "http"
		lb_port = 80
		lb_protocol = "http"
	}
}

resource "aws_lb_cookie_stickiness_policy" "foo" {
	name = "foo-policy"
	load_balancer = "${aws_elb.lb.id}"
	lb_port = 80
	cookie_expiration_period = 300
}
`
