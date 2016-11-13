package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSELBListener_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSELBListenerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSELBListenersConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSListenersExist("aws_elb_listener.bar"),
				),
			},
		},
	})
}

func getAWSElbListenerLBNames(lbnames string) []interface{} {
	names := make([]interface{}, 0)

	lbnamesSpilt := strings.Split(lbnames, ",")
	for i, lbname := range lbnamesSpilt {
		names[i] = lbname
	}

	return names
}

func testAccCheckAWSListenersExist(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ELB Listener ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).elbconn

		lbNames, lbNamesOk := rs.Primary.Attributes["loadbalancer_names"]
		if !lbNamesOk {
			return fmt.Errorf("Cannot find loadbalancer names in state")
		}
		names := getAWSElbListenerLBNames(lbNames)

		describe, err := conn.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
			LoadBalancerNames: expandStringList(names),
		})

		if err == nil {
			if len(lbNames) != len(describe.LoadBalancerDescriptions) {
				return fmt.Errorf("Not all loadbalancers found. Expected %d:, got %d", len(lbNames), len(describe.LoadBalancerDescriptions))
			}

			for _, lb := range describe.LoadBalancerDescriptions {
				found := false
				for _, listener := range lb.ListenerDescriptions {
					if fmt.Sprintf("%s", *listener.Listener.LoadBalancerPort) == rs.Primary.ID {
						found = true
						break
					}

				}
				if !found {
					return fmt.Errorf("Cannot find Listener with port %d in LB %s", rs.Primary.ID)
				}
			}
		}

		return nil
	}
}

func testAccCheckAWSELBListenerDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elbconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elb_listener" {
			continue
		}

		lbNames, lbNamesOk := rs.Primary.Attributes["loadbalancer_names"]
		if !lbNamesOk {
			return fmt.Errorf("Cannot find loadbalancer names in state")
		}
		names := getAWSElbListenerLBNames(lbNames)

		describe, err := conn.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
			LoadBalancerNames: expandStringList(names),
		})

		if err == nil {
			for _, lb := range describe.LoadBalancerDescriptions {
				for _, listener := range lb.ListenerDescriptions {
					if fmt.Sprintf("%s", *listener.Listener.LoadBalancerPort) == rs.Primary.ID {
						return fmt.Errorf("ELB Listener still exists")
					}
				}
			}
		}
	}

	return nil
}

const testAccAWSELBListenersConfig = `
resource "aws_elb" "foo" {
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }
}

resource "aws_elb" "bar" {
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }
}

resource "aws_elb_listener" "another_listener" {
  loadbalancer_names = ["${aws_elb.foo.name}", "${aws_elb.bar.name}"]
  instance_port = 8500
  instance_protocol = "http"
  lb_port = 8500
  lb_protocol = "http"
}
`
