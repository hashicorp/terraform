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
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSELB_basic(t *testing.T) {
	var conf elb.LoadBalancerDescription

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSELBConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.bar", &conf),
					testAccCheckAWSELBAttributes(&conf),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "availability_zones.#", "3"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "availability_zones.2487133097", "us-west-2a"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "availability_zones.221770259", "us-west-2b"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "availability_zones.2050015877", "us-west-2c"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "subnets.#", "3"),
					// NOTE: Subnet IDs are different across AWS accounts and cannot be checked.
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
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
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

func TestAccAWSELB_AccessLogs_enabled(t *testing.T) {
	var conf elb.LoadBalancerDescription

	rName := fmt.Sprintf("terraform-access-logs-bucket-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSELBAccessLogs,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.foo", &conf),
				),
			},

			{
				Config: testAccAWSELBAccessLogsOn(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.foo", &conf),
					resource.TestCheckResourceAttr(
						"aws_elb.foo", "access_logs.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_elb.foo", "access_logs.0.bucket", rName),
					resource.TestCheckResourceAttr(
						"aws_elb.foo", "access_logs.0.interval", "5"),
					resource.TestCheckResourceAttr(
						"aws_elb.foo", "access_logs.0.enabled", "true"),
				),
			},

			{
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

func TestAccAWSELB_AccessLogs_disabled(t *testing.T) {
	var conf elb.LoadBalancerDescription

	rName := fmt.Sprintf("terraform-access-logs-bucket-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSELBAccessLogs,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.foo", &conf),
				),
			},

			{
				Config: testAccAWSELBAccessLogsDisabled(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.foo", &conf),
					resource.TestCheckResourceAttr(
						"aws_elb.foo", "access_logs.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_elb.foo", "access_logs.0.bucket", rName),
					resource.TestCheckResourceAttr(
						"aws_elb.foo", "access_logs.0.interval", "5"),
					resource.TestCheckResourceAttr(
						"aws_elb.foo", "access_logs.0.enabled", "false"),
				),
			},

			{
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

func TestAccAWSELB_namePrefix(t *testing.T) {
	var conf elb.LoadBalancerDescription
	nameRegex := regexp.MustCompile("^test-")

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSELB_namePrefix,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.test", &conf),
					resource.TestMatchResourceAttr(
						"aws_elb.test", "name", nameRegex),
				),
			},
		},
	})
}

func TestAccAWSELB_generatedName(t *testing.T) {
	var conf elb.LoadBalancerDescription
	generatedNameRegexp := regexp.MustCompile("^tf-lb-")

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
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

func TestAccAWSELB_generatesNameForZeroValue(t *testing.T) {
	var conf elb.LoadBalancerDescription
	generatedNameRegexp := regexp.MustCompile("^tf-lb-")

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSELB_zeroValueName,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.foo", &conf),
					resource.TestMatchResourceAttr(
						"aws_elb.foo", "name", generatedNameRegexp),
				),
			},
		},
	})
}

func TestAccAWSELB_availabilityZones(t *testing.T) {
	var conf elb.LoadBalancerDescription

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSELBConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.bar", &conf),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "availability_zones.#", "3"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "availability_zones.2487133097", "us-west-2a"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "availability_zones.221770259", "us-west-2b"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "availability_zones.2050015877", "us-west-2c"),
				),
			},

			{
				Config: testAccAWSELBConfig_AvailabilityZonesUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.bar", &conf),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "availability_zones.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "availability_zones.2487133097", "us-west-2a"),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "availability_zones.221770259", "us-west-2b"),
				),
			},
		},
	})
}

func TestAccAWSELB_tags(t *testing.T) {
	var conf elb.LoadBalancerDescription
	var td elb.TagDescription

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSELBConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.bar", &conf),
					testAccCheckAWSELBAttributes(&conf),
					testAccLoadTags(&conf, &td),
					testAccCheckELBTags(&td.Tags, "bar", "baz"),
				),
			},

			{
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
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccELBIAMServerCertConfig(
					fmt.Sprintf("tf-acctest-%s", acctest.RandString(10))),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.bar", &conf),
					testCheck,
				),
			},
		},
	})
}

func TestAccAWSELB_swap_subnets(t *testing.T) {
	var conf elb.LoadBalancerDescription

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.ourapp",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSELBConfig_subnets,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.ourapp", &conf),
					resource.TestCheckResourceAttr(
						"aws_elb.ourapp", "subnets.#", "2"),
				),
			},

			{
				Config: testAccAWSELBConfig_subnet_swap,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.ourapp", &conf),
					resource.TestCheckResourceAttr(
						"aws_elb.ourapp", "subnets.#", "2"),
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
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSELBConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.bar", &conf),
					testAccCheckAWSELBAttributes(&conf),
				),
			},

			{
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
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSELBConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSELBExists("aws_elb.bar", &conf),
					testAccCheckAWSELBAttributes(&conf),
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "listener.206423021.instance_port", "8000"),
				),
			},

			{
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
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
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

func TestAccAWSELBUpdate_HealthCheck(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSELBConfigHealthCheck,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "health_check.0.healthy_threshold", "5"),
				),
			},
			{
				Config: testAccAWSELBConfigHealthCheck_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "health_check.0.healthy_threshold", "10"),
				),
			},
		},
	})
}

func TestAccAWSELB_Timeout(t *testing.T) {
	var conf elb.LoadBalancerDescription

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
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
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSELBConfigIdleTimeout,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "idle_timeout", "200",
					),
				),
			},
			{
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
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
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
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
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
			{
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
			{
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
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_elb.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSELBConfig,
				Check: resource.ComposeTestCheckFunc(
					// ELBs get a default security group
					resource.TestCheckResourceAttr(
						"aws_elb.bar", "security_groups.#", "1",
					),
				),
			},
			{
				Config: testAccAWSELBConfigSecurityGroups,
				Check: resource.ComposeTestCheckFunc(
					// Count should still be one as we swap in a custom security group
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

func TestResourceAWSELB_validateElbNameCanBeAnEmptyString(t *testing.T) {
	var elbName = ""
	_, errors := validateElbName(elbName, "SampleKey")

	if len(errors) != 0 {
		t.Fatalf("Expected the ELB Name to pass validation")
	}
}

func TestResourceAWSELB_validateElbNameCannotBeLongerThan32Characters(t *testing.T) {
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

func TestResourceAWSELB_validateAccessLogsInterval(t *testing.T) {
	type testCases struct {
		Value    int
		ErrCount int
	}

	invalidCases := []testCases{
		{
			Value:    0,
			ErrCount: 1,
		},
		{
			Value:    10,
			ErrCount: 1,
		},
		{
			Value:    -1,
			ErrCount: 1,
		},
	}

	for _, tc := range invalidCases {
		_, errors := validateAccessLogsInterval(tc.Value, "interval")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q to trigger a validation error.", tc.Value)
		}
	}

}

func TestResourceAWSELB_validateListenerProtocol(t *testing.T) {
	type testCases struct {
		Value    string
		ErrCount int
	}

	invalidCases := []testCases{
		{
			Value:    "",
			ErrCount: 1,
		},
		{
			Value:    "incorrect",
			ErrCount: 1,
		},
		{
			Value:    "HTTP:",
			ErrCount: 1,
		},
	}

	for _, tc := range invalidCases {
		_, errors := validateListenerProtocol(tc.Value, "protocol")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q to trigger a validation error.", tc.Value)
		}
	}

	validCases := []testCases{
		{
			Value:    "TCP",
			ErrCount: 0,
		},
		{
			Value:    "ssl",
			ErrCount: 0,
		},
		{
			Value:    "HTTP",
			ErrCount: 0,
		},
	}

	for _, tc := range validCases {
		_, errors := validateListenerProtocol(tc.Value, "protocol")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q not to trigger a validation error.", tc.Value)
		}
	}
}

func TestResourceAWSELB_validateHealthCheckTarget(t *testing.T) {
	type testCase struct {
		Value    string
		ErrCount int
	}

	randomRunes := func(n int) string {
		rand.Seed(time.Now().UTC().UnixNano())

		// A complete set of modern Katakana characters.
		runes := []rune("アイウエオ" +
			"カキクケコガギグゲゴサシスセソザジズゼゾ" +
			"タチツテトダヂヅデドナニヌネノハヒフヘホ" +
			"バビブベボパピプペポマミムメモヤユヨラリ" +
			"ルレロワヰヱヲン")

		s := make([]rune, n)
		for i := range s {
			s[i] = runes[rand.Intn(len(runes))]
		}
		return string(s)
	}

	validCases := []testCase{
		{
			Value:    "TCP:1234",
			ErrCount: 0,
		},
		{
			Value:    "http:80/test",
			ErrCount: 0,
		},
		{
			Value:    fmt.Sprintf("HTTP:8080/%s", randomRunes(5)),
			ErrCount: 0,
		},
		{
			Value:    "SSL:8080",
			ErrCount: 0,
		},
	}

	for _, tc := range validCases {
		_, errors := validateHeathCheckTarget(tc.Value, "target")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q not to trigger a validation error.", tc.Value)
		}
	}

	invalidCases := []testCase{
		{
			Value:    "",
			ErrCount: 1,
		},
		{
			Value:    "TCP:",
			ErrCount: 1,
		},
		{
			Value:    "TCP:1234/",
			ErrCount: 1,
		},
		{
			Value:    "SSL:8080/",
			ErrCount: 1,
		},
		{
			Value:    "HTTP:8080",
			ErrCount: 1,
		},
		{
			Value:    "incorrect-value",
			ErrCount: 1,
		},
		{
			Value:    "TCP:123456",
			ErrCount: 1,
		},
		{
			Value:    "incorrect:80/",
			ErrCount: 1,
		},
		{
			Value:    fmt.Sprintf("HTTP:8080/%s%s", randomString(512), randomRunes(512)),
			ErrCount: 1,
		},
	}

	for _, tc := range invalidCases {
		_, errors := validateHeathCheckTarget(tc.Value, "target")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q to trigger a validation error.", tc.Value)
		}
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

func testAccAWSELBAccessLogsOn(r string) string {
	return fmt.Sprintf(`
# an S3 bucket configured for Access logs
# The 797873946194 is the AWS ID for us-west-2, so this test
# must be ran in us-west-2
resource "aws_s3_bucket" "acceslogs_bucket" {
  bucket = "%s"
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
      "Resource": "arn:aws:s3:::%s/*",
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
`, r, r)
}

func testAccAWSELBAccessLogsDisabled(r string) string {
	return fmt.Sprintf(`
# an S3 bucket configured for Access logs
# The 797873946194 is the AWS ID for us-west-2, so this test
# must be ran in us-west-2
resource "aws_s3_bucket" "acceslogs_bucket" {
  bucket = "%s"
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
      "Resource": "arn:aws:s3:::%s/*",
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
		enabled = false
	}
}
`, r, r)
}

const testAccAWSELB_namePrefix = `
resource "aws_elb" "test" {
  name_prefix = "test-"
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

const testAccAWSELB_zeroValueName = `
resource "aws_elb" "foo" {
  name               = ""
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }
}
`

const testAccAWSELBConfig_AvailabilityZonesUpdate = `
resource "aws_elb" "bar" {
  availability_zones = ["us-west-2a", "us-west-2b"]

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

	tags {
		Name = "tf_elb_sg_test"
	}
}
`

// This IAM Server config is lifted from
// builtin/providers/aws/resource_aws_iam_server_certificate_test.go
func testAccELBIAMServerCertConfig(certName string) string {
	return fmt.Sprintf(`
resource "aws_iam_server_certificate" "test_cert" {
  name = "%s"
  certificate_body = <<EOF
-----BEGIN CERTIFICATE-----
MIIDBjCCAe4CCQCGWwBmOiHQdTANBgkqhkiG9w0BAQUFADBFMQswCQYDVQQGEwJB
VTETMBEGA1UECBMKU29tZS1TdGF0ZTEhMB8GA1UEChMYSW50ZXJuZXQgV2lkZ2l0
cyBQdHkgTHRkMB4XDTE2MDYyMTE2MzM0MVoXDTE3MDYyMTE2MzM0MVowRTELMAkG
A1UEBhMCQVUxEzARBgNVBAgTClNvbWUtU3RhdGUxITAfBgNVBAoTGEludGVybmV0
IFdpZGdpdHMgUHR5IEx0ZDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
AL+LFlsCJG5txZp4yuu+lQnuUrgBXRG+irQqcTXlV91Bp5hpmRIyhnGCtWxxDBUL
xrh4WN3VV/0jDzKT976oLgOy3hj56Cdqf+JlZ1qgMN5bHB3mm3aVWnrnsLbBsfwZ
SEbk3Kht/cE1nK2toNVW+rznS3m+eoV3Zn/DUNwGlZr42hGNs6ETn2jURY78ETqR
mW47xvjf86eIo7vULHJaY6xyarPqkL8DZazOmvY06hUGvGwGBny7gugfXqDG+I8n
cPBsGJGSAmHmVV8o0RCB9UjY+TvSMQRpEDoVlvyrGuglsD8to/4+7UcsuDGlRYN6
jmIOC37mOi/jwRfWL1YUa4MCAwEAATANBgkqhkiG9w0BAQUFAAOCAQEAPDxTH0oQ
JjKXoJgkmQxurB81RfnK/NrswJVzWbOv6ejcbhwh+/ZgJTMc15BrYcxU6vUW1V/i
Z7APU0qJ0icECACML+a2fRI7YdLCTiPIOmY66HY8MZHAn3dGjU5TeiUflC0n0zkP
mxKJe43kcYLNDItbfvUDo/GoxTXrC3EFVZyU0RhFzoVJdODlTHXMVFCzcbQEBrBJ
xKdShCEc8nFMneZcGFeEU488ntZoWzzms8/QpYrKa5S0Sd7umEU2Kwu4HTkvUFg/
CqDUFjhydXxYRsxXBBrEiLOE5BdtJR1sH/QHxIJe23C9iHI2nS1NbLziNEApLwC4
GnSud83VUo9G9w==
-----END CERTIFICATE-----
EOF

	private_key =  <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAv4sWWwIkbm3FmnjK676VCe5SuAFdEb6KtCpxNeVX3UGnmGmZ
EjKGcYK1bHEMFQvGuHhY3dVX/SMPMpP3vqguA7LeGPnoJ2p/4mVnWqAw3lscHeab
dpVaeuewtsGx/BlIRuTcqG39wTWcra2g1Vb6vOdLeb56hXdmf8NQ3AaVmvjaEY2z
oROfaNRFjvwROpGZbjvG+N/zp4iju9QsclpjrHJqs+qQvwNlrM6a9jTqFQa8bAYG
fLuC6B9eoMb4jydw8GwYkZICYeZVXyjREIH1SNj5O9IxBGkQOhWW/Ksa6CWwPy2j
/j7tRyy4MaVFg3qOYg4LfuY6L+PBF9YvVhRrgwIDAQABAoIBAFqJ4h1Om+3e0WK8
6h4YzdYN4ue7LUTv7hxPW4gASlH5cMDoWURywX3yLNN/dBiWom4b5NWmvJqY8dwU
eSyTznxNFhJ0PjozaxOWnw4FXlQceOPhV2bsHgKudadNU1Y4lSN9lpe+tg2Xy+GE
ituM66RTKCf502w3DioiJpx6OEkxuhrnsQAWNcGB0MnTukm2f+629V+04R5MT5V1
nY+5Phx2BpHgYzWBKh6Px1puu7xFv5SMQda1ndlPIKb4cNp0yYn+1lHNjbOE7QL/
oEpWgrauS5Zk/APK33v/p3wVYHrKocIFHlPiCW0uIJJLsOZDY8pQXpTlc+/xGLLy
WBu4boECgYEA6xO+1UNh6ndJ3xGuNippH+ucTi/uq1+0tG1bd63v+75tn5l4LyY2
CWHRaWVlVn+WnDslkQTJzFD68X+9M7Cc4oP6WnhTyPamG7HlGv5JxfFHTC9GOKmz
sSc624BDmqYJ7Xzyhe5kc3iHzqG/L72ZF1aijZdrodQMSY1634UX6aECgYEA0Jdr
cBPSN+mgmEY6ogN5h7sO5uNV3TQQtW2IslfWZn6JhSRF4Rf7IReng48CMy9ZhFBy
Q7H2I1pDGjEC9gQHhgVfm+FyMSVqXfCHEW/97pvvu9ougHA0MhPep1twzTGrqg+K
f3PLW8hVkGyCrTfWgbDlPsHgsocA/wTaQOheaqMCgYBat5z+WemQfQZh8kXDm2xE
KD2Cota9BcsLkeQpdFNXWC6f167cqydRSZFx1fJchhJOKjkeFLX3hgzBY6VVLEPu
2jWj8imLNTv3Fhiu6RD5NVppWRkFRuAUbmo1SPNN2+Oa5YwGCXB0a0Alip/oQYex
zPogIB4mLlmrjNCtL4SB4QKBgCEHKMrZSJrz0irqS9RlanPUaZqjenAJE3A2xMNA
Z0FZXdsIEEyA6JGn1i1dkoKaR7lMp5sSbZ/RZfiatBZSMwLEjQv4mYUwoHP5Ztma
+wEyDbaX6G8L1Sfsv3+OWgETkVPfHBXsNtH0mZ/BnrtgsQVeBh52wmZiPAUlNo26
fWCzAoGBAJOjqovLelLWzyQGqPFx/MwuI56UFXd1CmFlCIvF2WxCFmk3tlExoCN1
HqSpt92vsgYgV7+lAb4U7Uy/v012gwiU1LK+vyAE9geo3pTjG73BNzG4H547xtbY
dg+Sd4Wjm89UQoUUoiIcstY7FPbqfBtYKfh4RYHAHV2BwDFqzZCM
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
`, certName)
}

const testAccAWSELBConfig_subnets = `
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "azelb" {
  cidr_block           = "10.1.0.0/16"
  enable_dns_hostnames = true

  tags {
    Name = "subnet-vpc"
  }
}

resource "aws_subnet" "public_a_one" {
  vpc_id = "${aws_vpc.azelb.id}"

  cidr_block        = "10.1.1.0/24"
  availability_zone = "us-west-2a"
}

resource "aws_subnet" "public_b_one" {
  vpc_id = "${aws_vpc.azelb.id}"

  cidr_block        = "10.1.7.0/24"
  availability_zone = "us-west-2b"
}

resource "aws_subnet" "public_a_two" {
  vpc_id = "${aws_vpc.azelb.id}"

  cidr_block        = "10.1.2.0/24"
  availability_zone = "us-west-2a"
}

resource "aws_elb" "ourapp" {
  name = "terraform-asg-deployment-example"

  subnets = [
    "${aws_subnet.public_a_one.id}",
    "${aws_subnet.public_b_one.id}",
  ]

  listener {
    instance_port     = 80
    instance_protocol = "http"
    lb_port           = 80
    lb_protocol       = "http"
  }

  depends_on = ["aws_internet_gateway.gw"]
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.azelb.id}"

  tags {
    Name = "main"
  }
}
`

const testAccAWSELBConfig_subnet_swap = `
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "azelb" {
  cidr_block           = "10.1.0.0/16"
  enable_dns_hostnames = true

  tags {
    Name = "subnet-vpc"
  }
}

resource "aws_subnet" "public_a_one" {
  vpc_id = "${aws_vpc.azelb.id}"

  cidr_block        = "10.1.1.0/24"
  availability_zone = "us-west-2a"
}

resource "aws_subnet" "public_b_one" {
  vpc_id = "${aws_vpc.azelb.id}"

  cidr_block        = "10.1.7.0/24"
  availability_zone = "us-west-2b"
}

resource "aws_subnet" "public_a_two" {
  vpc_id = "${aws_vpc.azelb.id}"

  cidr_block        = "10.1.2.0/24"
  availability_zone = "us-west-2a"
}

resource "aws_elb" "ourapp" {
  name = "terraform-asg-deployment-example"

  subnets = [
    "${aws_subnet.public_a_two.id}",
    "${aws_subnet.public_b_one.id}",
  ]

  listener {
    instance_port     = 80
    instance_protocol = "http"
    lb_port           = 80
    lb_protocol       = "http"
  }

  depends_on = ["aws_internet_gateway.gw"]
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.azelb.id}"

  tags {
    Name = "main"
  }
}
`
