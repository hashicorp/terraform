package aws

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSELB_basic(t *testing.T) {
	var conf elb.LoadBalancerDescription
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
						"aws_elb.bar", "availability_zones.2487133097", "us-west-2a"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "availability_zones.221770259", "us-west-2b"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "availability_zones.2050015877", "us-west-2c"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "listener.206423021.instance_port", "8000"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "listener.206423021.instance_protocol", "http"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "listener.206423021.ssl_certificate_id", ssl_certificate_id),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "listener.206423021.lb_port", "80"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "listener.206423021.lb_protocol", "http"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "cross_zone_load_balancing", "true"),
				),
			},
		},
	})
}

func TestAccAWSELB_fullCharacterRange(t *testing.T) {
	var conf elb.LoadBalancerDescription

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSELBFullRangeOfCharacters,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.foo", &conf),
					resource.TestCheckResourceAttr(
						"aws_elb.foo", "name", "FoobarTerraform-test123"),
				),
			},
		},
	})
}

func TestAccAWSELB_generatedName(t *testing.T) {
	var conf elb.LoadBalancerDescription
	generatedNameRegexp := regexp.MustCompile("^tf-lb-")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSELBGeneratedName,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.foo", &conf),
					resource.TestMatchResourceAttr(
						"aws_elb.foo", "name", generatedNameRegexp),
				),
			},
		},
	})
}

func TestAccAWSELB_tags(t *testing.T) {
	var conf elb.LoadBalancerDescription
	var td elb.TagDescription

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
					testAccLoadTags(&conf, &td),
					testAccCheckELBTags(&td.Tags, "bar", "baz"),
				),
			},

			resource.TestStep{
				Config: testAccAWSELBConfig_TagUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.bar", &conf),
					testAccCheckAWSELBAttributes(&conf),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "name", "foobar-terraform-test"),
					testAccLoadTags(&conf, &td),
					testAccCheckELBTags(&td.Tags, "foo", "bar"),
					testAccCheckELBTags(&td.Tags, "new", "type"),
				),
			},
		},
	})
}

func testAccLoadTags(conf *elb.LoadBalancerDescription, td *elb.TagDescription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).elbconn

		describe, err := conn.DescribeTags(&elb.DescribeTagsInput{
			LoadBalancerNames: []*string{conf.LoadBalancerName},
		})

		if err != nil {
			return err
		}
		if len(describe.TagDescriptions) > 0 {
			*td = *describe.TagDescriptions[0]
		}
		return nil
	}
}

func TestAccAWSELB_InstanceAttaching(t *testing.T) {
	var conf elb.LoadBalancerDescription

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

func TestAccAWSELBUpdate_Listener(t *testing.T) {
	var conf elb.LoadBalancerDescription

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
						"aws_elb.bar", "listener.206423021.instance_port", "8000"),
				),
			},

			resource.TestStep{
				Config: testAccAWSELBConfigListener_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.bar", &conf),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "listener.3931999347.instance_port", "8080"),
				),
			},
		},
	})
}

func TestAccAWSELB_HealthCheck(t *testing.T) {
	var conf elb.LoadBalancerDescription

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
						"aws_elb.bar", "health_check.3484319807.healthy_threshold", "5"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "health_check.3484319807.unhealthy_threshold", "5"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "health_check.3484319807.target", "HTTP:8000/"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "health_check.3484319807.timeout", "30"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "health_check.3484319807.interval", "60"),
				),
			},
		},
	})
}

func TestAccAWSELBUpdate_HealthCheck(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSELBConfigHealthCheck,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "health_check.3484319807.healthy_threshold", "5"),
				),
			},
			resource.TestStep{
				Config: testAccAWSELBConfigHealthCheck_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "health_check.2648756019.healthy_threshold", "10"),
				),
			},
		},
	})
}

func TestAccAWSELB_Timeout(t *testing.T) {
	var conf elb.LoadBalancerDescription

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSELBConfigIdleTimeout,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.bar", &conf),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "idle_timeout", "200",
					),
				),
			},
		},
	})
}

func TestAccAWSELBUpdate_Timeout(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSELBConfigIdleTimeout,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "idle_timeout", "200",
					),
				),
			},
			resource.TestStep{
				Config: testAccAWSELBConfigIdleTimeout_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "idle_timeout", "400",
					),
				),
			},
		},
	})
}

func TestAccAWSELB_ConnectionDraining(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSELBConfigConnectionDraining,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "connection_draining", "true",
					),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "connection_draining_timeout", "400",
					),
				),
			},
		},
	})
}

func TestAccAWSELBUpdate_ConnectionDraining(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSELBConfigConnectionDraining,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "connection_draining", "true",
					),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "connection_draining_timeout", "400",
					),
				),
			},
			resource.TestStep{
				Config: testAccAWSELBConfigConnectionDraining_update_timeout,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "connection_draining", "true",
					),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "connection_draining_timeout", "600",
					),
				),
			},
			resource.TestStep{
				Config: testAccAWSELBConfigConnectionDraining_update_disable,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "connection_draining", "false",
					),
				),
			},
		},
	})
}

func TestAccAWSELB_SecurityGroups(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSELBConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "security_groups.#", "0",
					),
				),
			},
			resource.TestStep{
				Config: testAccAWSELBConfigSecurityGroups,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "security_groups.#", "1",
					),
				),
			},
		},
	})
}

// Unit test for listeners hash
func TestResourceAwsElbListenerHash(t *testing.T) {
	cases := map[string]struct {
		Left  map[string]interface{}
		Right map[string]interface{}
		Match bool
	}{
		"protocols are case insensitive": {
			map[string]interface{}{
				"instance_port":     80,
				"instance_protocol": "TCP",
				"lb_port":           80,
				"lb_protocol":       "TCP",
			},
			map[string]interface{}{
				"instance_port":     80,
				"instance_protocol": "Tcp",
				"lb_port":           80,
				"lb_protocol":       "tcP",
			},
			true,
		},
	}

	for tn, tc := range cases {
		leftHash := resourceAwsElbListenerHash(tc.Left)
		rightHash := resourceAwsElbListenerHash(tc.Right)
		if (leftHash == rightHash) != tc.Match {
			t.Fatalf("%s: expected match: %t, but did not get it", tn, tc.Match)
		}
	}
}

func TestResourceAWSELB_validateElbNameCannotBeginWithHyphen(t *testing.T) {
	var elbName = "-Testing123"
	_, errors := validateElbName(elbName, "SampleKey")

	if len(errors) != 1 {
		t.Fatalf("Expected the ELB Name to trigger a validation error")
	}
}

func TestResourceAWSELB_validateElbNameCannotBeLongerThen32Characters(t *testing.T) {
	var elbName = "Testing123dddddddddddddddddddvvvv"
	_, errors := validateElbName(elbName, "SampleKey")

	if len(errors) != 1 {
		t.Fatalf("Expected the ELB Name to trigger a validation error")
	}
}

func TestResourceAWSELB_validateElbNameCannotHaveSpecialCharacters(t *testing.T) {
	var elbName = "Testing123%%"
	_, errors := validateElbName(elbName, "SampleKey")

	if len(errors) != 1 {
		t.Fatalf("Expected the ELB Name to trigger a validation error")
	}
}

func TestResourceAWSELB_validateElbNameCannotEndWithHyphen(t *testing.T) {
	var elbName = "Testing123-"
	_, errors := validateElbName(elbName, "SampleKey")

	if len(errors) != 1 {
		t.Fatalf("Expected the ELB Name to trigger a validation error")
	}
}

func testAccCheckAWSELBDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elbconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elb" {
			continue
		}

		describe, err := conn.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
			LoadBalancerNames: []*string{aws.String(rs.Primary.ID)},
		})

		if err == nil {
			if len(describe.LoadBalancerDescriptions) != 0 &&
				*describe.LoadBalancerDescriptions[0].LoadBalancerName == rs.Primary.ID {
				return fmt.Errorf("ELB still exists")
			}
		}

		// Verify the error
		providerErr, ok := err.(awserr.Error)
		if !ok {
			return err
		}

		if providerErr.Code() != "InvalidLoadBalancerName.NotFound" {
			return fmt.Errorf("Unexpected error: %s", err)
		}
	}

	return nil
}

func testAccCheckAWSELBAttributes(conf *elb.LoadBalancerDescription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		zones := []string{"us-west-2a", "us-west-2b", "us-west-2c"}
		azs := make([]string, 0, len(conf.AvailabilityZones))
		for _, x := range conf.AvailabilityZones {
			azs = append(azs, *x)
		}
		sort.StringSlice(azs).Sort()
		if !reflect.DeepEqual(azs, zones) {
			return fmt.Errorf("bad availability_zones")
		}

		if *conf.LoadBalancerName != "foobar-terraform-test" {
			return fmt.Errorf("bad name")
		}

		l := elb.Listener{
			InstancePort:     aws.Int64(int64(8000)),
			InstanceProtocol: aws.String("HTTP"),
			LoadBalancerPort: aws.Int64(int64(80)),
			Protocol:         aws.String("HTTP"),
		}

		if !reflect.DeepEqual(conf.ListenerDescriptions[0].Listener, &l) {
			return fmt.Errorf(
				"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
				conf.ListenerDescriptions[0].Listener,
				l)
		}

		if *conf.DNSName == "" {
			return fmt.Errorf("empty dns_name")
		}

		return nil
	}
}

func testAccCheckAWSELBAttributesHealthCheck(conf *elb.LoadBalancerDescription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		zones := []string{"us-west-2a", "us-west-2b", "us-west-2c"}
		azs := make([]string, 0, len(conf.AvailabilityZones))
		for _, x := range conf.AvailabilityZones {
			azs = append(azs, *x)
		}
		sort.StringSlice(azs).Sort()
		if !reflect.DeepEqual(azs, zones) {
			return fmt.Errorf("bad availability_zones")
		}

		if *conf.LoadBalancerName != "foobar-terraform-test" {
			return fmt.Errorf("bad name")
		}

		check := &elb.HealthCheck{
			Timeout:            aws.Int64(int64(30)),
			UnhealthyThreshold: aws.Int64(int64(5)),
			HealthyThreshold:   aws.Int64(int64(5)),
			Interval:           aws.Int64(int64(60)),
			Target:             aws.String("HTTP:8000/"),
		}

		if !reflect.DeepEqual(conf.HealthCheck, check) {
			return fmt.Errorf(
				"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
				conf.HealthCheck,
				check)
		}

		if *conf.DNSName == "" {
			return fmt.Errorf("empty dns_name")
		}

		return nil
	}
}

func testAccCheckAWSELBExists(n string, res *elb.LoadBalancerDescription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ELB ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).elbconn

		describe, err := conn.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
			LoadBalancerNames: []*string{aws.String(rs.Primary.ID)},
		})

		if err != nil {
			return err
		}

		if len(describe.LoadBalancerDescriptions) != 1 ||
			*describe.LoadBalancerDescriptions[0].LoadBalancerName != rs.Primary.ID {
			return fmt.Errorf("ELB not found")
		}

		*res = *describe.LoadBalancerDescriptions[0]

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
    // Protocol should be case insensitive
    lb_protocol = "HttP"
  }

	tags {
		bar = "baz"
	}

  cross_zone_load_balancing = true
}
`

const testAccAWSELBFullRangeOfCharacters = `
resource "aws_elb" "foo" {
  name = "FoobarTerraform-test123"
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }
}
`

const testAccAWSELBGeneratedName = `
resource "aws_elb" "foo" {
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }
}
`

const testAccAWSELBConfig_TagUpdate = `
resource "aws_elb" "bar" {
  name = "foobar-terraform-test"
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }

	tags {
		foo = "bar"
		new = "type"
	}

  cross_zone_load_balancing = true
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
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

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

const testAccAWSELBConfigHealthCheck_update = `
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
    healthy_threshold = 10
    unhealthy_threshold = 5
    target = "HTTP:8000/"
    interval = 60
    timeout = 30
  }
}
`

const testAccAWSELBConfigListener_update = `
resource "aws_elb" "bar" {
  name = "foobar-terraform-test"
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port = 8080
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }
}
`

const testAccAWSELBConfigIdleTimeout = `
resource "aws_elb" "bar" {
	name = "foobar-terraform-test"
	availability_zones = ["us-west-2a"]

	listener {
		instance_port = 8000
		instance_protocol = "http"
		lb_port = 80
		lb_protocol = "http"
	}

	idle_timeout = 200
}
`

const testAccAWSELBConfigIdleTimeout_update = `
resource "aws_elb" "bar" {
	name = "foobar-terraform-test"
	availability_zones = ["us-west-2a"]

	listener {
		instance_port = 8000
		instance_protocol = "http"
		lb_port = 80
		lb_protocol = "http"
	}

	idle_timeout = 400
}
`

const testAccAWSELBConfigConnectionDraining = `
resource "aws_elb" "bar" {
	name = "foobar-terraform-test"
	availability_zones = ["us-west-2a"]

	listener {
		instance_port = 8000
		instance_protocol = "http"
		lb_port = 80
		lb_protocol = "http"
	}

	connection_draining = true
	connection_draining_timeout = 400
}
`

const testAccAWSELBConfigConnectionDraining_update_timeout = `
resource "aws_elb" "bar" {
	name = "foobar-terraform-test"
	availability_zones = ["us-west-2a"]

	listener {
		instance_port = 8000
		instance_protocol = "http"
		lb_port = 80
		lb_protocol = "http"
	}

	connection_draining = true
	connection_draining_timeout = 600
}
`

const testAccAWSELBConfigConnectionDraining_update_disable = `
resource "aws_elb" "bar" {
	name = "foobar-terraform-test"
	availability_zones = ["us-west-2a"]

	listener {
		instance_port = 8000
		instance_protocol = "http"
		lb_port = 80
		lb_protocol = "http"
	}

	connection_draining = false
}
`

const testAccAWSELBConfigSecurityGroups = `
resource "aws_elb" "bar" {
  name = "foobar-terraform-test"
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }

  security_groups = ["${aws_security_group.bar.id}"]
}

resource "aws_security_group" "bar" {
  name = "terraform-elb-acceptance-test"
  description = "Used in the terraform acceptance tests for the elb resource"

  ingress {
    protocol = "tcp"
    from_port = 80
    to_port = 80
    cidr_blocks = ["0.0.0.0/0"]
  }
}
`
