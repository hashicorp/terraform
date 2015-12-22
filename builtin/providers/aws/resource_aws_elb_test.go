package aws

import (
	"fmt"
	"math/rand"
	"reflect"
	"regexp"
	"sort"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSELB_basic(t *testing.T) {
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

	lbName := fmt.Sprintf("Tf-%d",
		rand.New(rand.NewSource(time.Now().UnixNano())).Int())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccAWSELBFullRangeOfCharacters, lbName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.foo", &conf),
					resource.TestCheckResourceAttr(
						"aws_elb.foo", "name", lbName),
				),
			},
		},
	})
}

func TestAccAWSELB_AccessLogs(t *testing.T) {
	var conf elb.LoadBalancerDescription

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSELBAccessLogs,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.foo", &conf),
				),
			},

			resource.TestStep{
				Config: testAccAWSELBAccessLogsOn,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.foo", &conf),
					resource.TestCheckResourceAttr(
						"aws_elb.foo", "access_logs.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_elb.foo", "access_logs.1713209538.bucket", "terraform-access-logs-bucket"),
					resource.TestCheckResourceAttr(
						"aws_elb.foo", "access_logs.1713209538.interval", "5"),
				),
			},

			resource.TestStep{
				Config: testAccAWSELBAccessLogs,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.foo", &conf),
					resource.TestCheckResourceAttr(
						"aws_elb.foo", "access_logs.#", "0"),
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
					testAccLoadTags(&conf, &td),
					testAccCheckELBTags(&td.Tags, "bar", "baz"),
				),
			},

			resource.TestStep{
				Config: testAccAWSELBConfig_TagUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.bar", &conf),
					testAccCheckAWSELBAttributes(&conf),
					testAccLoadTags(&conf, &td),
					testAccCheckELBTags(&td.Tags, "foo", "bar"),
					testAccCheckELBTags(&td.Tags, "new", "type"),
				),
			},
		},
	})
}

func TestAccAWSELB_iam_server_cert(t *testing.T) {
	var conf elb.LoadBalancerDescription
	// var td elb.TagDescription
	testCheck := func(*terraform.State) error {
		if len(conf.ListenerDescriptions) != 1 {
			return fmt.Errorf(
				"TestAccAWSELB_iam_server_cert expected 1 listener, got %d",
				len(conf.ListenerDescriptions))
		}
		return nil
	}
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccELBIAMServerCertConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.bar", &conf),
					testCheck,
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
		if leftHash == rightHash != tc.Match {
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

		if providerErr.Code() != "LoadBalancerNotFound" {
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

		// Confirm source_security_group_id for ELBs in a VPC
		// 	See https://github.com/hashicorp/terraform/pull/3780
		if res.VPCId != nil {
			sgid := rs.Primary.Attributes["source_security_group_id"]
			if sgid == "" {
				return fmt.Errorf("Expected to find source_security_group_id for ELB, but was empty")
			}
		}

		return nil
	}
}

const testAccAWSELBConfig = `
resource "aws_elb" "bar" {
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
  name = "%s"
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }
}
`

const testAccAWSELBAccessLogs = `
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
const testAccAWSELBAccessLogsOn = `
# an S3 bucket configured for Access logs
# The 797873946194 is the AWS ID for us-west-2, so this test
# must be ran in us-west-2
resource "aws_s3_bucket" "acceslogs_bucket" {
  bucket = "terraform-access-logs-bucket"
  acl = "private"
  force_destroy = true
  policy = <<EOF
{
  "Id": "Policy1446577137248",
  "Statement": [
    {
      "Action": "s3:PutObject",
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::797873946194:root"
      },
      "Resource": "arn:aws:s3:::terraform-access-logs-bucket/*",
      "Sid": "Stmt1446575236270"
    }
  ],
  "Version": "2012-10-17"
}
EOF
}

resource "aws_elb" "foo" {
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }

	access_logs {
		interval = 5
		bucket = "${aws_s3_bucket.acceslogs_bucket.bucket}"
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
  ingress {
    protocol = "tcp"
    from_port = 80
    to_port = 80
    cidr_blocks = ["0.0.0.0/0"]
  }
}
`

// This IAM Server config is lifted from
// builtin/providers/aws/resource_aws_iam_server_certificate_test.go
var testAccELBIAMServerCertConfig = `
resource "aws_iam_server_certificate" "test_cert" {
  name = "terraform-test-cert-elb"
  certificate_body = <<EOF
-----BEGIN CERTIFICATE-----
MIIDCDCCAfACAQEwDQYJKoZIhvcNAQELBQAwgY4xCzAJBgNVBAYTAlVTMREwDwYD
VQQIDAhOZXcgWW9yazERMA8GA1UEBwwITmV3IFlvcmsxFjAUBgNVBAoMDUJhcmVm
b290IExhYnMxGDAWBgNVBAMMD0phc29uIEJlcmxpbnNreTEnMCUGCSqGSIb3DQEJ
ARYYamFzb25AYmFyZWZvb3Rjb2RlcnMuY29tMB4XDTE1MDYyMTA1MzcwNVoXDTE2
MDYyMDA1MzcwNVowgYgxCzAJBgNVBAYTAlVTMREwDwYDVQQIDAhOZXcgWW9yazEL
MAkGA1UEBwwCTlkxFjAUBgNVBAoMDUJhcmVmb290IExhYnMxGDAWBgNVBAMMD0ph
c29uIEJlcmxpbnNreTEnMCUGCSqGSIb3DQEJARYYamFzb25AYmFyZWZvb3Rjb2Rl
cnMuY29tMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQD2AVGKRIx+EFM0kkg7
6GoJv9uy0biEDHB4phQBqnDIf8J8/gq9eVvQrR5jJC9Uz4zp5wG/oLZlGuF92/jD
bI/yS+DOAjrh30vN79Au74jGN2Cw8fIak40iDUwjZaczK2Gkna54XIO9pqMcbQ6Q
mLUkQXsqlJ7Q4X2kL3b9iMsXcQIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQCDGNvU
eioQMVPNlmmxW3+Rwo0Kl+/HtUOmqUDKUDvJnelxulBr7O8w75N/Z7h7+aBJCUkt
tz+DwATZswXtsal6TuzHHpAhpFql82jQZVE8OYkrX84XKRQpm8ZnbyZObMdXTJWk
ArC/rGVIWsvhlbgGM8zu7a3zbeuAESZ8Bn4ZbJxnoaRK8p36/alvzAwkgzSf3oUX
HtU4LrdunevBs6/CbKCWrxYcvNCy8EcmHitqCfQL5nxCCXpgf/Mw1vmIPTwbPSJq
oUkh5yjGRKzhh7QbG1TlFX6zUp4vb+UJn5+g4edHrqivRSjIqYrC45ygVMOABn21
hpMXOlZL+YXfR4Kp
-----END CERTIFICATE-----
EOF

  certificate_chain = <<EOF
-----BEGIN CERTIFICATE-----
MIID8TCCAtmgAwIBAgIJAKX2xeCkfFcbMA0GCSqGSIb3DQEBCwUAMIGOMQswCQYD
VQQGEwJVUzERMA8GA1UECAwITmV3IFlvcmsxETAPBgNVBAcMCE5ldyBZb3JrMRYw
FAYDVQQKDA1CYXJlZm9vdCBMYWJzMRgwFgYDVQQDDA9KYXNvbiBCZXJsaW5za3kx
JzAlBgkqhkiG9w0BCQEWGGphc29uQGJhcmVmb290Y29kZXJzLmNvbTAeFw0xNTA2
MjEwNTM2MDZaFw0yNTA2MTgwNTM2MDZaMIGOMQswCQYDVQQGEwJVUzERMA8GA1UE
CAwITmV3IFlvcmsxETAPBgNVBAcMCE5ldyBZb3JrMRYwFAYDVQQKDA1CYXJlZm9v
dCBMYWJzMRgwFgYDVQQDDA9KYXNvbiBCZXJsaW5za3kxJzAlBgkqhkiG9w0BCQEW
GGphc29uQGJhcmVmb290Y29kZXJzLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEP
ADCCAQoCggEBAMteFbwfLz7NyQn3eDxxw22l1ZPBrzfPON0HOAq8nHat4kT4A2cI
45kCtxKMzCVoG84tXoX/rbjGkez7lz9lEfvEuSh+I+UqinFA/sefhcE63foVMZu1
2t6O3+utdxBvOYJwAQaiGW44x0h6fTyqDv6Gc5Ml0uoIVeMWPhT1MREoOcPDz1gb
Ep3VT2aqFULLJedP37qbzS4D04rn1tS7pcm3wYivRyjVNEvs91NsWEvvE1WtS2Cl
2RBt+ihXwq4UNB9UPYG75+FuRcQQvfqameyweyKT9qBmJLELMtYa/KTCYvSch4JY
YVPAPOlhFlO4BcTto/gpBes2WEAWZtE/jnECAwEAAaNQME4wHQYDVR0OBBYEFOna
aiYnm5583EY7FT/mXwTBuLZgMB8GA1UdIwQYMBaAFOnaaiYnm5583EY7FT/mXwTB
uLZgMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBABp/dKQ489CCzzB1
IX78p6RFAdda4e3lL6uVjeS3itzFIIiKvdf1/txhmsEeCEYz0El6aMnXLkpk7jAr
kCwlAOOz2R2hlA8k8opKTYX4IQQau8DATslUFAFOvRGOim/TD/Yuch+a/VF2VQKz
L2lUVi5Hjp9KvWe2HQYPjnJaZs/OKAmZQ4uP547dqFrTz6sWfisF1rJ60JH70cyM
qjZQp/xYHTZIB8TCPvLgtVIGFmd/VAHVBFW2p9IBwtSxBIsEPwYQOV3XbwhhmGIv
DWx5TpnEzH7ZM33RNbAKcdwOBxdRY+SI/ua5hYCm4QngAqY69lEuk4zXZpdDLPq1
qxxQx0E=
-----END CERTIFICATE-----
EOF

	private_key =  <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQD2AVGKRIx+EFM0kkg76GoJv9uy0biEDHB4phQBqnDIf8J8/gq9
eVvQrR5jJC9Uz4zp5wG/oLZlGuF92/jDbI/yS+DOAjrh30vN79Au74jGN2Cw8fIa
k40iDUwjZaczK2Gkna54XIO9pqMcbQ6QmLUkQXsqlJ7Q4X2kL3b9iMsXcQIDAQAB
AoGALmVBQ5p6BKx/hMKx7NqAZSZSAP+clQrji12HGGlUq/usanZfAC0LK+f6eygv
5QbfxJ1UrxdYTukq7dm2qOSooOMUuukWInqC6ztjdLwH70CKnl0bkNB3/NkW2VNc
32YiUuZCM9zaeBuEUclKNs+dhD2EeGdJF8KGntWGOTU/M4ECQQD9gdYb38PvaMdu
opM3sKJF5n9pMoLDleBpCGqq3nD3DFn0V6PHQAwn30EhRN+7BbUEpde5PmfoIdAR
uDlj/XPlAkEA+GyY1e4uU9rz+1K4ubxmtXTp9ZIR2LsqFy5L/MS5hqX2zq5GGq8g
jZYDxnxPEUrxaWQH4nh0qdu3skUBi4a0nQJBAKJaqLkpUd7eB/t++zHLWeHSgP7q
bny8XABod4f+9fICYwntpuJQzngqrxeTeIXaXdggLkxg/0LXhN4UUg0LoVECQQDE
Pi1h2dyY+37/CzLH7q+IKopjJneYqQmv9C+sxs70MgjM7liM3ckub9IdqrdfJr+c
DJw56APo5puvZNm6mbf1AkBVMDyfdOOyoHpJjrhmZWo6QqynujfwErrBYQ0sZQ3l
O57Z0RUNQ8DRyymhLd2t5nAHTfpcFA1sBeKE6CziLbZB
-----END RSA PRIVATE KEY-----
EOF
}

resource "aws_elb" "bar" {
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port = 8000
    instance_protocol = "https"
    lb_port = 80
    // Protocol should be case insensitive
    lb_protocol = "HttPs"
    ssl_certificate_id = "${aws_iam_server_certificate.test_cert.arn}"
  }

	tags {
		bar = "baz"
	}

  cross_zone_load_balancing = true
}
`
