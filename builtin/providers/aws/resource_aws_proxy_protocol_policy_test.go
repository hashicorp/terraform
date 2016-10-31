package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSProxyProtocolPolicy_basic(t *testing.T) {
	lbName := fmt.Sprintf("tf-test-lb-%s", acctest.RandString(5))
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckProxyProtocolPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProxyProtocolPolicyConfig(lbName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_proxy_protocol_policy.smtp", "load_balancer", lbName),
					resource.TestCheckResourceAttr(
						"aws_proxy_protocol_policy.smtp", "instance_ports.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_proxy_protocol_policy.smtp", "instance_ports.4196041389", "25"),
				),
			},
			{
				Config: testAccProxyProtocolPolicyConfigUpdate(lbName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_proxy_protocol_policy.smtp", "load_balancer", lbName),
					resource.TestCheckResourceAttr(
						"aws_proxy_protocol_policy.smtp", "instance_ports.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_proxy_protocol_policy.smtp", "instance_ports.4196041389", "25"),
					resource.TestCheckResourceAttr(
						"aws_proxy_protocol_policy.smtp", "instance_ports.1925441437", "587"),
				),
			},
		},
	})
}

func testAccCheckProxyProtocolPolicyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elbconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_placement_group" {
			continue
		}

		req := &elb.DescribeLoadBalancersInput{
			LoadBalancerNames: []*string{
				aws.String(rs.Primary.Attributes["load_balancer"])},
		}
		_, err := conn.DescribeLoadBalancers(req)
		if err != nil {
			// Verify the error is what we want
			if isLoadBalancerNotFound(err) {
				continue
			}
			return err
		}

		return fmt.Errorf("still exists")
	}
	return nil
}

func testAccProxyProtocolPolicyConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_elb" "lb" {
	name = "%s"
	availability_zones = ["us-west-2a"]

	listener {
		instance_port = 25
		instance_protocol = "tcp"
		lb_port = 25
		lb_protocol = "tcp"
	}

	listener {
		instance_port = 587
		instance_protocol = "tcp"
		lb_port = 587
		lb_protocol = "tcp"
	}
}

resource "aws_proxy_protocol_policy" "smtp" {
	load_balancer = "${aws_elb.lb.name}"
	instance_ports = ["25"]
}`, rName)
}

func testAccProxyProtocolPolicyConfigUpdate(rName string) string {
	return fmt.Sprintf(`
resource "aws_elb" "lb" {
	name = "%s"
	availability_zones = ["us-west-2a"]

	listener {
		instance_port = 25
		instance_protocol = "tcp"
		lb_port = 25
		lb_protocol = "tcp"
	}

	listener {
		instance_port = 587
		instance_protocol = "tcp"
		lb_port = 587
		lb_protocol = "tcp"
	}
}

resource "aws_proxy_protocol_policy" "smtp" {
	load_balancer = "${aws_elb.lb.name}"
	instance_ports = ["25", "587"]
}`, rName)
}
