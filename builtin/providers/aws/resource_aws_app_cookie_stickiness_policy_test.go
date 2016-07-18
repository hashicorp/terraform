package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elb"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAppCookieStickinessPolicy_basic(t *testing.T) {
	lbName := fmt.Sprintf("tf-test-lb-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAppCookieStickinessPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAppCookieStickinessPolicyConfig(lbName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppCookieStickinessPolicy(
						"aws_elb.lb",
						"aws_app_cookie_stickiness_policy.foo",
					),
				),
			},
			resource.TestStep{
				Config: testAccAppCookieStickinessPolicyConfigUpdate(lbName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppCookieStickinessPolicy(
						"aws_elb.lb",
						"aws_app_cookie_stickiness_policy.foo",
					),
				),
			},
		},
	})
}

func TestAccAWSAppCookieStickinessPolicy_missingLB(t *testing.T) {
	lbName := fmt.Sprintf("tf-test-lb-%s", acctest.RandString(5))

	// check that we can destroy the policy if the LB is missing
	removeLB := func() {
		conn := testAccProvider.Meta().(*AWSClient).elbconn
		deleteElbOpts := elb.DeleteLoadBalancerInput{
			LoadBalancerName: aws.String(lbName),
		}
		if _, err := conn.DeleteLoadBalancer(&deleteElbOpts); err != nil {
			t.Fatalf("Error deleting ELB: %s", err)
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAppCookieStickinessPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAppCookieStickinessPolicyConfig(lbName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppCookieStickinessPolicy(
						"aws_elb.lb",
						"aws_app_cookie_stickiness_policy.foo",
					),
				),
			},
			resource.TestStep{
				PreConfig: removeLB,
				Config:    testAccAppCookieStickinessPolicyConfigDestroy(lbName),
			},
		},
	})
}

func testAccCheckAppCookieStickinessPolicyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elbconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_app_cookie_stickiness_policy" {
			continue
		}

		lbName, _, policyName := resourceAwsAppCookieStickinessPolicyParseId(
			rs.Primary.ID)
		out, err := conn.DescribeLoadBalancerPolicies(
			&elb.DescribeLoadBalancerPoliciesInput{
				LoadBalancerName: aws.String(lbName),
				PolicyNames:      []*string{aws.String(policyName)},
			})
		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && (ec2err.Code() == "PolicyNotFound" || ec2err.Code() == "LoadBalancerNotFound") {
				continue
			}
			return err
		}

		if len(out.PolicyDescriptions) > 0 {
			return fmt.Errorf("Policy still exists")
		}
	}
	return nil
}

func testAccCheckAppCookieStickinessPolicy(elbResource string, policyResource string) resource.TestCheckFunc {
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
		elbName, _, policyName := resourceAwsAppCookieStickinessPolicyParseId(policy.Primary.ID)
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

// ensure the policy is re-added is it goes missing
func TestAccAWSAppCookieStickinessPolicy_drift(t *testing.T) {
	lbName := fmt.Sprintf("tf-test-lb-%s", acctest.RandString(5))

	// We only want to remove the reference to the policy from the listner,
	// beacause that's all that can be done via the console.
	removePolicy := func() {
		conn := testAccProvider.Meta().(*AWSClient).elbconn

		setLoadBalancerOpts := &elb.SetLoadBalancerPoliciesOfListenerInput{
			LoadBalancerName: aws.String(lbName),
			LoadBalancerPort: aws.Int64(80),
			PolicyNames:      []*string{},
		}

		if _, err := conn.SetLoadBalancerPoliciesOfListener(setLoadBalancerOpts); err != nil {
			t.Fatalf("Error removing AppCookieStickinessPolicy: %s", err)
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAppCookieStickinessPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAppCookieStickinessPolicyConfig(lbName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppCookieStickinessPolicy(
						"aws_elb.lb",
						"aws_app_cookie_stickiness_policy.foo",
					),
				),
			},
			resource.TestStep{
				PreConfig: removePolicy,
				Config:    testAccAppCookieStickinessPolicyConfig(lbName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppCookieStickinessPolicy(
						"aws_elb.lb",
						"aws_app_cookie_stickiness_policy.foo",
					),
				),
			},
		},
	})
}

func testAccAppCookieStickinessPolicyConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_elb" "lb" {
	name = "%s"
	availability_zones = ["us-west-2a"]
	listener {
		instance_port = 8000
		instance_protocol = "http"
		lb_port = 80
		lb_protocol = "http"
	}
}

resource "aws_app_cookie_stickiness_policy" "foo" {
	name = "foo-policy"
	load_balancer = "${aws_elb.lb.id}"
	lb_port = 80
	cookie_name = "MyAppCookie"
}`, rName)
}

// Change the cookie_name to "MyOtherAppCookie".
func testAccAppCookieStickinessPolicyConfigUpdate(rName string) string {
	return fmt.Sprintf(`
resource "aws_elb" "lb" {
	name = "%s"
	availability_zones = ["us-west-2a"]
	listener {
		instance_port = 8000
		instance_protocol = "http"
		lb_port = 80
		lb_protocol = "http"
	}
}

resource "aws_app_cookie_stickiness_policy" "foo" {
	name = "foo-policy"
	load_balancer = "${aws_elb.lb.id}"
	lb_port = 80
	cookie_name = "MyOtherAppCookie"
}`, rName)
}

// attempt to destroy the policy, but we'll delete the LB in the PreConfig
func testAccAppCookieStickinessPolicyConfigDestroy(rName string) string {
	return fmt.Sprintf(`
resource "aws_elb" "lb" {
	name = "%s"
	availability_zones = ["us-west-2a"]
	listener {
		instance_port = 8000
		instance_protocol = "http"
		lb_port = 80
		lb_protocol = "http"
	}
}`, rName)
}
