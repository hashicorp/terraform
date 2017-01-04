package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elb"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSLoadBalancerListenerPolicy_basic(t *testing.T) {
	rChar := acctest.RandStringFromCharSet(6, acctest.CharSetAlpha)
	lbName := fmt.Sprintf("%s", rChar)
	mcName := fmt.Sprintf("%s", rChar)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLoadBalancerListenerPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSLoadBalancerListenerPolicyConfig_basic0(lbName, mcName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLoadBalancerPolicyState("aws_elb.test-lb", "aws_load_balancer_policy.magic-cookie-sticky"),
					testAccCheckAWSLoadBalancerListenerPolicyState(lbName, int64(80), mcName, true),
				),
			},
			resource.TestStep{
				Config: testAccAWSLoadBalancerListenerPolicyConfig_basic1(lbName, mcName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLoadBalancerPolicyState("aws_elb.test-lb", "aws_load_balancer_policy.magic-cookie-sticky"),
					testAccCheckAWSLoadBalancerListenerPolicyState(lbName, int64(80), mcName, true),
				),
			},
			resource.TestStep{
				Config: testAccAWSLoadBalancerListenerPolicyConfig_basic2(lbName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLoadBalancerListenerPolicyState(lbName, int64(80), mcName, false),
				),
			},
		},
	})
}

func policyInListenerPolicies(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

func testAccCheckAWSLoadBalancerListenerPolicyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elbconn

	for _, rs := range s.RootModule().Resources {
		switch {
		case rs.Type == "aws_load_balancer_policy":
			loadBalancerName, policyName := resourceAwsLoadBalancerListenerPoliciesParseId(rs.Primary.ID)
			out, err := conn.DescribeLoadBalancerPolicies(
				&elb.DescribeLoadBalancerPoliciesInput{
					LoadBalancerName: aws.String(loadBalancerName),
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
		case rs.Type == "aws_load_listener_policy":
			loadBalancerName, _ := resourceAwsLoadBalancerListenerPoliciesParseId(rs.Primary.ID)
			out, err := conn.DescribeLoadBalancers(
				&elb.DescribeLoadBalancersInput{
					LoadBalancerNames: []*string{aws.String(loadBalancerName)},
				})
			if err != nil {
				if ec2err, ok := err.(awserr.Error); ok && (ec2err.Code() == "LoadBalancerNotFound") {
					continue
				}
				return err
			}
			policyNames := []string{}
			for k, _ := range rs.Primary.Attributes {
				if strings.HasPrefix(k, "policy_names.") && strings.HasSuffix(k, ".name") {
					value_key := fmt.Sprintf("%s.value", strings.TrimSuffix(k, ".name"))
					policyNames = append(policyNames, rs.Primary.Attributes[value_key])
				}
			}
			for _, policyName := range policyNames {
				for _, listener := range out.LoadBalancerDescriptions[0].ListenerDescriptions {
					policyStrings := []string{}
					for _, pol := range listener.PolicyNames {
						policyStrings = append(policyStrings, *pol)
					}
					if policyInListenerPolicies(policyName, policyStrings) {
						return fmt.Errorf("Policy still exists and is assigned")
					}
				}
			}
		default:
			continue
		}
	}
	return nil
}

func testAccCheckAWSLoadBalancerListenerPolicyState(loadBalancerName string, loadBalancerListenerPort int64, loadBalancerListenerPolicyName string, assigned bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		elbconn := testAccProvider.Meta().(*AWSClient).elbconn

		loadBalancerDescription, err := elbconn.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
			LoadBalancerNames: []*string{aws.String(loadBalancerName)},
		})
		if err != nil {
			return err
		}

		for _, listener := range loadBalancerDescription.LoadBalancerDescriptions[0].ListenerDescriptions {
			if *listener.Listener.LoadBalancerPort != loadBalancerListenerPort {
				continue
			}
			policyStrings := []string{}
			for _, pol := range listener.PolicyNames {
				policyStrings = append(policyStrings, *pol)
			}
			if policyInListenerPolicies(loadBalancerListenerPolicyName, policyStrings) != assigned {
				if assigned {
					return fmt.Errorf("Policy no longer assigned %s not in %+v", loadBalancerListenerPolicyName, policyStrings)
				} else {
					return fmt.Errorf("Policy exists and is assigned")
				}
			}
		}

		return nil
	}
}

func testAccAWSLoadBalancerListenerPolicyConfig_basic0(lbName, mcName string) string {
	return fmt.Sprintf(`
resource "aws_elb" "test-lb" {
  name = "%s"
  availability_zones = ["us-west-2a"]

  listener {
    instance_port = 80
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }

  tags {
    Name = "tf-acc-test"
  }
}

resource "aws_load_balancer_policy" "magic-cookie-sticky" {
  load_balancer_name = "${aws_elb.test-lb.name}"
  policy_name = "%s"
  policy_type_name = "AppCookieStickinessPolicyType"
  policy_attribute = {
    name = "CookieName"
    value = "magic_cookie"
  }
}

resource "aws_load_balancer_listener_policy" "test-lb-listener-policies-80" {
  load_balancer_name = "${aws_elb.test-lb.name}"
  load_balancer_port = 80
  policy_names = [
    "${aws_load_balancer_policy.magic-cookie-sticky.policy_name}",
  ]
}`, lbName, mcName)
}

func testAccAWSLoadBalancerListenerPolicyConfig_basic1(lbName, mcName string) string {
	return fmt.Sprintf(`
resource "aws_elb" "test-lb" {
  name = "%s"
  availability_zones = ["us-west-2a"]

  listener {
    instance_port = 80
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }

  tags {
    Name = "tf-acc-test"
  }
}

resource "aws_load_balancer_policy" "magic-cookie-sticky" {
  load_balancer_name = "${aws_elb.test-lb.name}"
  policy_name = "%s"
  policy_type_name = "AppCookieStickinessPolicyType"
  policy_attribute = {
    name = "CookieName"
    value = "unicorn_cookie"
  }
}

resource "aws_load_balancer_listener_policy" "test-lb-listener-policies-80" {
  load_balancer_name = "${aws_elb.test-lb.name}"
  load_balancer_port = 80
  policy_names = [
    "${aws_load_balancer_policy.magic-cookie-sticky.policy_name}"
  ]
}`, lbName, mcName)
}

func testAccAWSLoadBalancerListenerPolicyConfig_basic2(lbName string) string {
	return fmt.Sprintf(`
resource "aws_elb" "test-lb" {
  name = "%s"
  availability_zones = ["us-west-2a"]

  listener {
    instance_port = 80
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }

  tags {
    Name = "tf-acc-test"
  }
}`, lbName)
}
