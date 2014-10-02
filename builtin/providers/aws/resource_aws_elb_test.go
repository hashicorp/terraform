package aws

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/elb"
)

func TestAccAWSELB_basic(t *testing.T) {
	var conf elb.LoadBalancer
	ssl_certificate_id := os.Getenv("AWS_SSL_CERTIFICATE_ID")

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
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "name", "foobar-terraform-test"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "availability_zones.0", "us-west-2a"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "availability_zones.1", "us-west-2b"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "availability_zones.2", "us-west-2c"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "listener.0.instance_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "listener.0.instance_protocol", "http"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "listener.0.ssl_certificate_id", ssl_certificate_id),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "listener.0.lb_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "listener.0.lb_protocol", "http"),
				),
			},
		},
	})
}

func TestAccAWSELB_InstanceAttaching(t *testing.T) {
	var conf elb.LoadBalancer

	testCheckInstanceAttached := func(count int) resource.TestCheckFunc {
		return func(*terraform.State) error {
			if len(conf.Instances) != count {
				return fmt.Errorf("instance count does not match")
			}
			return nil
		}
	}

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

			resource.TestStep{
				Config: testAccAWSELBConfigNewInstance,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.bar", &conf),
					testCheckInstanceAttached(1),
				),
			},
		},
	})
}

func TestAccAWSELB_HealthCheck(t *testing.T) {
	var conf elb.LoadBalancer

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSELBConfigHealthCheck,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.bar", &conf),
					testAccCheckAWSELBAttributesHealthCheck(&conf),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "health_check.0.healthy_threshold", "5"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "health_check.0.unhealthy_threshold", "5"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "health_check.0.target", "HTTP:8000/"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "health_check.0.timeout", "30"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "health_check.0.interval", "60"),
				),
			},
		},
	})
}
func testAccCheckAWSELBDestroy(s *terraform.State) error {
	conn := testAccProvider.elbconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elb" {
			continue
		}

		describe, err := conn.DescribeLoadBalancers(&elb.DescribeLoadBalancer{
			Names: []string{rs.Primary.ID},
		})

		if err == nil {
			if len(describe.LoadBalancers) != 0 &&
				describe.LoadBalancers[0].LoadBalancerName == rs.Primary.ID {
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
		if conf.AvailabilityZones[0].AvailabilityZone != "us-west-2a" {
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

func testAccCheckAWSELBAttributesHealthCheck(conf *elb.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if conf.AvailabilityZones[0].AvailabilityZone != "us-west-2a" {
			return fmt.Errorf("bad availability_zones")
		}

		if conf.LoadBalancerName != "foobar-terraform-test" {
			return fmt.Errorf("bad name")
		}

		check := elb.HealthCheck{
			Timeout:            30,
			UnhealthyThreshold: 5,
			HealthyThreshold:   5,
			Interval:           60,
			Target:             "HTTP:8000/",
		}

		if !reflect.DeepEqual(conf.HealthCheck, check) {
			return fmt.Errorf(
				"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
				conf.HealthCheck,
				check)
		}

		if conf.DNSName == "" {
			return fmt.Errorf("empty dns_name")
		}

		return nil
	}
}

func testAccCheckAWSELBExists(n string, res *elb.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ELB ID is set")
		}

		conn := testAccProvider.elbconn

		describe, err := conn.DescribeLoadBalancers(&elb.DescribeLoadBalancer{
			Names: []string{rs.Primary.ID},
		})

		if err != nil {
			return err
		}

		if len(describe.LoadBalancers) != 1 ||
			describe.LoadBalancers[0].LoadBalancerName != rs.Primary.ID {
			return fmt.Errorf("ELB not found")
		}

		*res = describe.LoadBalancers[0]

		return nil
	}
}

const testAccAWSELBConfig = `
resource "aws_elb" "bar" {
  name = "foobar-terraform-test"
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }

  instances = []
}
`

const testAccAWSELBConfigNewInstance = `
resource "aws_elb" "bar" {
  name = "foobar-terraform-test"
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }

  instances = ["${aws_instance.foo.id}"]
}

resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-043a5034"
	instance_type = "t1.micro"
}
`

const testAccAWSELBConfigListenerSSLCertificateId = `
resource "aws_elb" "bar" {
  name = "foobar-terraform-test"
  availability_zones = ["us-west-2a"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    ssl_certificate_id = "%s"
    lb_port = 443
    lb_protocol = "https"
  }
}
`

const testAccAWSELBConfigHealthCheck = `
resource "aws_elb" "bar" {
  name = "foobar-terraform-test"
  availability_zones = ["us-west-2a"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }

  health_check {
    healthy_threshold = 5
    unhealthy_threshold = 5
    target = "HTTP:8000/"
    interval = 60
    timeout = 30
  }
}
`
