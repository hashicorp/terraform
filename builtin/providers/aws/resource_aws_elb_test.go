package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/elb"
)

func TestAccAWSELB(t *testing.T) {
	var conf elb.LoadBalancer

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSELBConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.bar", &conf),
					testAccCheckAWSELBAttributes(&conf),
				),
			},
		},
	})
}

func testAccCheckAWSELBDestroy(s *terraform.State) error {
	conn := testAccProvider.elbconn

	for _, rs := range s.Resources {
		if rs.Type != "aws_elb" {
			continue
		}

		describe, err := conn.DescribeLoadBalancers(&elb.DescribeLoadBalancer{
			Names: []string{rs.ID},
		})

		if err == nil {
			if len(describe.LoadBalancers) != 0 &&
				describe.LoadBalancers[0].LoadBalancerName == rs.ID {
				return fmt.Errorf("ELB still exists")
			}
		}

		// Verify the error
		providerErr, ok := err.(*elb.Error)
		if !ok {
			return err
		}

		if providerErr.Code != "InvalidLoadBalancerName.NotFound" {
			return fmt.Errorf("Unexpected error: %s", err)
		}
	}

	return nil
}

func testAccCheckAWSELBAttributes(conf *elb.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if conf.AvailabilityZones[0].AvailabilityZone != "us-east-1a" {
			return fmt.Errorf("bad availability_zones")
		}

		if conf.LoadBalancerName != "foobar-terraform-test" {
			return fmt.Errorf("bad name")
		}

		l := elb.Listener{
			InstancePort:     8000,
			InstanceProtocol: "HTTP",
			LoadBalancerPort: 80,
			Protocol:         "HTTP",
		}

		if !reflect.DeepEqual(conf.Listeners[0], l) {
			return fmt.Errorf(
				"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
				conf.Listeners[0],
				l)
		}

		if conf.DNSName == "" {
			return fmt.Errorf("empty dns_name")
		}

		return nil
	}
}

func testAccCheckAWSELBExists(n string, res *elb.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No ELB ID is set")
		}

		conn := testAccProvider.elbconn

		describe, err := conn.DescribeLoadBalancers(&elb.DescribeLoadBalancer{
			Names: []string{rs.ID},
		})

		if err != nil {
			return err
		}

		if len(describe.LoadBalancers) != 1 ||
			describe.LoadBalancers[0].LoadBalancerName != rs.ID {
			return fmt.Errorf("ELB not found")
		}

		*res = describe.LoadBalancers[0]

		return nil
	}
}

const testAccAWSELBConfig = `
resource "aws_elb" "bar" {
  name = "foobar-terraform-test"
  availability_zones = ["us-east-1a"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }

  instances = []
}
`
