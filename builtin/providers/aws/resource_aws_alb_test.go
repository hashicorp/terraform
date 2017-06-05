package aws

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestALBCloudwatchSuffixFromARN(t *testing.T) {
	cases := []struct {
		name   string
		arn    *string
		suffix string
	}{
		{
			name:   "valid suffix",
			arn:    aws.String(`arn:aws:elasticloadbalancing:us-east-1:123456:loadbalancer/app/my-alb/abc123`),
			suffix: `app/my-alb/abc123`,
		},
		{
			name:   "no suffix",
			arn:    aws.String(`arn:aws:elasticloadbalancing:us-east-1:123456:loadbalancer`),
			suffix: ``,
		},
		{
			name:   "nil ARN",
			arn:    nil,
			suffix: ``,
		},
	}

	for _, tc := range cases {
		actual := albSuffixFromARN(tc.arn)
		if actual != tc.suffix {
			t.Fatalf("bad suffix: %q\nExpected: %s\n     Got: %s", tc.name, tc.suffix, actual)
		}
	}
}

func TestAccAWSALB_basic(t *testing.T) {
	var conf elbv2.LoadBalancer
	albName := fmt.Sprintf("testaccawsalb-basic-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb.alb_test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBConfig_basic(albName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBExists("aws_alb.alb_test", &conf),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "name", albName),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "internal", "true"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "subnets.#", "2"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "security_groups.#", "1"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "tags.%", "1"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "tags.TestName", "TestAccAWSALB_basic"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "enable_deletion_protection", "false"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "idle_timeout", "30"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "ip_address_type", "ipv4"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "vpc_id"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "zone_id"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "dns_name"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "arn"),
				),
			},
		},
	})
}

func TestAccAWSALB_generatedName(t *testing.T) {
	var conf elbv2.LoadBalancer

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb.alb_test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBConfig_generatedName(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBExists("aws_alb.alb_test", &conf),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "name"),
				),
			},
		},
	})
}

func TestAccAWSALB_generatesNameForZeroValue(t *testing.T) {
	var conf elbv2.LoadBalancer

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb.alb_test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBConfig_zeroValueName(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBExists("aws_alb.alb_test", &conf),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "name"),
				),
			},
		},
	})
}

func TestAccAWSALB_namePrefix(t *testing.T) {
	var conf elbv2.LoadBalancer

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb.alb_test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBConfig_namePrefix(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBExists("aws_alb.alb_test", &conf),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "name"),
					resource.TestMatchResourceAttr("aws_alb.alb_test", "name",
						regexp.MustCompile("^tf-lb-")),
				),
			},
		},
	})
}

func TestAccAWSALB_tags(t *testing.T) {
	var conf elbv2.LoadBalancer
	albName := fmt.Sprintf("testaccawsalb-basic-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb.alb_test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBConfig_basic(albName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBExists("aws_alb.alb_test", &conf),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "tags.%", "1"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "tags.TestName", "TestAccAWSALB_basic"),
				),
			},
			{
				Config: testAccAWSALBConfig_updatedTags(albName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBExists("aws_alb.alb_test", &conf),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "tags.%", "2"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "tags.Type", "Sample Type Tag"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "tags.Environment", "Production"),
				),
			},
		},
	})
}

func TestAccAWSALB_updatedSecurityGroups(t *testing.T) {
	var pre, post elbv2.LoadBalancer
	albName := fmt.Sprintf("testaccawsalb-basic-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb.alb_test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBConfig_basic(albName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBExists("aws_alb.alb_test", &pre),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "security_groups.#", "1"),
				),
			},
			{
				Config: testAccAWSALBConfig_updateSecurityGroups(albName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBExists("aws_alb.alb_test", &post),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "security_groups.#", "2"),
					testAccCheckAWSAlbARNs(&pre, &post),
				),
			},
		},
	})
}

func TestAccAWSALB_updatedSubnets(t *testing.T) {
	var pre, post elbv2.LoadBalancer
	albName := fmt.Sprintf("testaccawsalb-basic-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb.alb_test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBConfig_basic(albName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBExists("aws_alb.alb_test", &pre),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "subnets.#", "2"),
				),
			},
			{
				Config: testAccAWSALBConfig_updateSubnets(albName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBExists("aws_alb.alb_test", &post),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "subnets.#", "3"),
					testAccCheckAWSAlbARNs(&pre, &post),
				),
			},
		},
	})
}

func TestAccAWSALB_updatedIpAddressType(t *testing.T) {
	var pre, post elbv2.LoadBalancer
	albName := fmt.Sprintf("testaccawsalb-basic-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb.alb_test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBConfigWithIpAddressType(albName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBExists("aws_alb.alb_test", &pre),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "ip_address_type", "ipv4"),
				),
			},
			{
				Config: testAccAWSALBConfigWithIpAddressTypeUpdated(albName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBExists("aws_alb.alb_test", &post),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "ip_address_type", "dualstack"),
				),
			},
		},
	})
}

// TestAccAWSALB_noSecurityGroup regression tests the issue in #8264,
// where if an ALB is created without a security group, a default one
// is assigned.
func TestAccAWSALB_noSecurityGroup(t *testing.T) {
	var conf elbv2.LoadBalancer
	albName := fmt.Sprintf("testaccawsalb-nosg-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb.alb_test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBConfig_nosg(albName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBExists("aws_alb.alb_test", &conf),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "name", albName),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "internal", "true"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "subnets.#", "2"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "security_groups.#", "1"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "tags.%", "1"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "tags.TestName", "TestAccAWSALB_basic"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "enable_deletion_protection", "false"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "idle_timeout", "30"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "vpc_id"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "zone_id"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "dns_name"),
				),
			},
		},
	})
}

func TestAccAWSALB_accesslogs(t *testing.T) {
	var conf elbv2.LoadBalancer
	bucketName := fmt.Sprintf("testaccawsalbaccesslogs-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))
	albName := fmt.Sprintf("testaccawsalbaccesslog-%s", acctest.RandStringFromCharSet(4, acctest.CharSetAlpha))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb.alb_test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBConfig_basic(albName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBExists("aws_alb.alb_test", &conf),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "name", albName),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "internal", "true"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "subnets.#", "2"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "security_groups.#", "1"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "tags.%", "1"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "tags.TestName", "TestAccAWSALB_basic"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "enable_deletion_protection", "false"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "idle_timeout", "30"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "vpc_id"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "zone_id"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "dns_name"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "arn"),
				),
			},
			{
				Config: testAccAWSALBConfig_accessLogs(true, albName, bucketName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBExists("aws_alb.alb_test", &conf),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "name", albName),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "internal", "true"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "subnets.#", "2"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "security_groups.#", "1"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "tags.%", "1"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "tags.TestName", "TestAccAWSALB_basic1"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "enable_deletion_protection", "false"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "idle_timeout", "50"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "vpc_id"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "zone_id"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "dns_name"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "access_logs.#", "1"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "access_logs.0.bucket", bucketName),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "access_logs.0.prefix", "testAccAWSALBConfig_accessLogs"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "access_logs.0.enabled", "true"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "arn"),
				),
			},
			{
				Config: testAccAWSALBConfig_accessLogs(false, albName, bucketName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBExists("aws_alb.alb_test", &conf),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "name", albName),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "internal", "true"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "subnets.#", "2"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "security_groups.#", "1"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "tags.%", "1"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "tags.TestName", "TestAccAWSALB_basic1"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "enable_deletion_protection", "false"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "idle_timeout", "50"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "vpc_id"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "zone_id"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "dns_name"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "access_logs.#", "1"),
					resource.TestCheckResourceAttr("aws_alb.alb_test", "access_logs.0.enabled", "false"),
					resource.TestCheckResourceAttrSet("aws_alb.alb_test", "arn"),
				),
			},
		},
	})
}

func testAccCheckAWSAlbARNs(pre, post *elbv2.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *pre.LoadBalancerArn != *post.LoadBalancerArn {
			return errors.New("ALB has been recreated. ARNs are different")
		}

		return nil
	}
}

func testAccCheckAWSALBExists(n string, res *elbv2.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("No ALB ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).elbv2conn

		describe, err := conn.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{
			LoadBalancerArns: []*string{aws.String(rs.Primary.ID)},
		})

		if err != nil {
			return err
		}

		if len(describe.LoadBalancers) != 1 ||
			*describe.LoadBalancers[0].LoadBalancerArn != rs.Primary.ID {
			return errors.New("ALB not found")
		}

		*res = *describe.LoadBalancers[0]
		return nil
	}
}

func testAccCheckAWSALBDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elbv2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_alb" {
			continue
		}

		describe, err := conn.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{
			LoadBalancerArns: []*string{aws.String(rs.Primary.ID)},
		})

		if err == nil {
			if len(describe.LoadBalancers) != 0 &&
				*describe.LoadBalancers[0].LoadBalancerArn == rs.Primary.ID {
				return fmt.Errorf("ALB %q still exists", rs.Primary.ID)
			}
		}

		// Verify the error
		if isLoadBalancerNotFound(err) {
			return nil
		} else {
			return errwrap.Wrapf("Unexpected error checking ALB destroyed: {{err}}", err)
		}
	}

	return nil
}

func testAccAWSALBConfigWithIpAddressTypeUpdated(albName string) string {
	return fmt.Sprintf(`resource "aws_alb" "alb_test" {
  name            = "%s"
  security_groups = ["${aws_security_group.alb_test.id}"]
  subnets         = ["${aws_subnet.alb_test_1.id}", "${aws_subnet.alb_test_2.id}"]

  ip_address_type = "dualstack"

  idle_timeout = 30
  enable_deletion_protection = false

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_alb_listener" "test" {
   load_balancer_arn = "${aws_alb.alb_test.id}"
   protocol = "HTTP"
   port = "80"

   default_action {
     target_group_arn = "${aws_alb_target_group.test.id}"
     type = "forward"
   }
}

resource "aws_alb_target_group" "test" {
  name = "%s"
  port = 80
  protocol = "HTTP"
  vpc_id = "${aws_vpc.alb_test.id}"

  deregistration_delay = 200

  stickiness {
    type = "lb_cookie"
    cookie_duration = 10000
  }

  health_check {
    path = "/health2"
    interval = 30
    port = 8082
    protocol = "HTTPS"
    timeout = 4
    healthy_threshold = 4
    unhealthy_threshold = 4
    matcher = "200"
  }
}

resource "aws_egress_only_internet_gateway" "igw" {
  vpc_id = "${aws_vpc.alb_test.id}"
}

resource "aws_vpc" "alb_test" {
  cidr_block = "10.0.0.0/16"
  assign_generated_ipv6_cidr_block = true

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_internet_gateway" "foo" {
  vpc_id = "${aws_vpc.alb_test.id}"
}

resource "aws_subnet" "alb_test_1" {
  vpc_id                  = "${aws_vpc.alb_test.id}"
  cidr_block              = "10.0.1.0/24"
  map_public_ip_on_launch = true
  availability_zone       = "us-west-2a"
  ipv6_cidr_block = "${cidrsubnet(aws_vpc.alb_test.ipv6_cidr_block, 8, 1)}"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_subnet" "alb_test_2" {
  vpc_id                  = "${aws_vpc.alb_test.id}"
  cidr_block              = "10.0.2.0/24"
  map_public_ip_on_launch = true
  availability_zone       = "us-west-2b"
  ipv6_cidr_block = "${cidrsubnet(aws_vpc.alb_test.ipv6_cidr_block, 8, 2)}"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_security_group" "alb_test" {
  name        = "allow_all_alb_test"
  description = "Used for ALB Testing"
  vpc_id      = "${aws_vpc.alb_test.id}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}`, albName, albName)
}

func testAccAWSALBConfigWithIpAddressType(albName string) string {
	return fmt.Sprintf(`resource "aws_alb" "alb_test" {
  name            = "%s"
  security_groups = ["${aws_security_group.alb_test.id}"]
  subnets         = ["${aws_subnet.alb_test_1.id}", "${aws_subnet.alb_test_2.id}"]

  ip_address_type = "ipv4"

  idle_timeout = 30
  enable_deletion_protection = false

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_alb_listener" "test" {
   load_balancer_arn = "${aws_alb.alb_test.id}"
   protocol = "HTTP"
   port = "80"

   default_action {
     target_group_arn = "${aws_alb_target_group.test.id}"
     type = "forward"
   }
}

resource "aws_alb_target_group" "test" {
  name = "%s"
  port = 80
  protocol = "HTTP"
  vpc_id = "${aws_vpc.alb_test.id}"

  deregistration_delay = 200

  stickiness {
    type = "lb_cookie"
    cookie_duration = 10000
  }

  health_check {
    path = "/health2"
    interval = 30
    port = 8082
    protocol = "HTTPS"
    timeout = 4
    healthy_threshold = 4
    unhealthy_threshold = 4
    matcher = "200"
  }
}

resource "aws_egress_only_internet_gateway" "igw" {
  vpc_id = "${aws_vpc.alb_test.id}"
}

resource "aws_vpc" "alb_test" {
  cidr_block = "10.0.0.0/16"
  assign_generated_ipv6_cidr_block = true

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_internet_gateway" "foo" {
  vpc_id = "${aws_vpc.alb_test.id}"
}

resource "aws_subnet" "alb_test_1" {
  vpc_id                  = "${aws_vpc.alb_test.id}"
  cidr_block              = "10.0.1.0/24"
  map_public_ip_on_launch = true
  availability_zone       = "us-west-2a"
  ipv6_cidr_block = "${cidrsubnet(aws_vpc.alb_test.ipv6_cidr_block, 8, 1)}"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_subnet" "alb_test_2" {
  vpc_id                  = "${aws_vpc.alb_test.id}"
  cidr_block              = "10.0.2.0/24"
  map_public_ip_on_launch = true
  availability_zone       = "us-west-2b"
  ipv6_cidr_block = "${cidrsubnet(aws_vpc.alb_test.ipv6_cidr_block, 8, 2)}"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_security_group" "alb_test" {
  name        = "allow_all_alb_test"
  description = "Used for ALB Testing"
  vpc_id      = "${aws_vpc.alb_test.id}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}`, albName, albName)
}

func testAccAWSALBConfig_basic(albName string) string {
	return fmt.Sprintf(`resource "aws_alb" "alb_test" {
  name            = "%s"
  internal        = true
  security_groups = ["${aws_security_group.alb_test.id}"]
  subnets         = ["${aws_subnet.alb_test.*.id}"]

  idle_timeout = 30
  enable_deletion_protection = false

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

variable "subnets" {
  default = ["10.0.1.0/24", "10.0.2.0/24"]
  type    = "list"
}

data "aws_availability_zones" "available" {}

resource "aws_vpc" "alb_test" {
  cidr_block = "10.0.0.0/16"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_subnet" "alb_test" {
  count                   = 2
  vpc_id                  = "${aws_vpc.alb_test.id}"
  cidr_block              = "${element(var.subnets, count.index)}"
  map_public_ip_on_launch = true
  availability_zone       = "${element(data.aws_availability_zones.available.names, count.index)}"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_security_group" "alb_test" {
  name        = "allow_all_alb_test"
  description = "Used for ALB Testing"
  vpc_id      = "${aws_vpc.alb_test.id}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}`, albName)
}

func testAccAWSALBConfig_updateSubnets(albName string) string {
	return fmt.Sprintf(`resource "aws_alb" "alb_test" {
  name            = "%s"
  internal        = true
  security_groups = ["${aws_security_group.alb_test.id}"]
  subnets         = ["${aws_subnet.alb_test.*.id}"]

  idle_timeout = 30
  enable_deletion_protection = false

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

variable "subnets" {
  default = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  type    = "list"
}

data "aws_availability_zones" "available" {}

resource "aws_vpc" "alb_test" {
  cidr_block = "10.0.0.0/16"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_subnet" "alb_test" {
  count                   = 3
  vpc_id                  = "${aws_vpc.alb_test.id}"
  cidr_block              = "${element(var.subnets, count.index)}"
  map_public_ip_on_launch = true
  availability_zone       = "${element(data.aws_availability_zones.available.names, count.index)}"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_security_group" "alb_test" {
  name        = "allow_all_alb_test"
  description = "Used for ALB Testing"
  vpc_id      = "${aws_vpc.alb_test.id}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}`, albName)
}

func testAccAWSALBConfig_generatedName() string {
	return fmt.Sprintf(`
resource "aws_alb" "alb_test" {
  internal        = true
  security_groups = ["${aws_security_group.alb_test.id}"]
  subnets         = ["${aws_subnet.alb_test.*.id}"]

  idle_timeout = 30
  enable_deletion_protection = false

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

variable "subnets" {
  default = ["10.0.1.0/24", "10.0.2.0/24"]
  type    = "list"
}

data "aws_availability_zones" "available" {}

resource "aws_vpc" "alb_test" {
  cidr_block = "10.0.0.0/16"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.alb_test.id}"

  tags {
    Name = "TestAccAWSALB_basic"
  }
}

resource "aws_subnet" "alb_test" {
  count                   = 2
  vpc_id                  = "${aws_vpc.alb_test.id}"
  cidr_block              = "${element(var.subnets, count.index)}"
  map_public_ip_on_launch = true
  availability_zone       = "${element(data.aws_availability_zones.available.names, count.index)}"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_security_group" "alb_test" {
  name        = "allow_all_alb_test"
  description = "Used for ALB Testing"
  vpc_id      = "${aws_vpc.alb_test.id}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}`)
}

func testAccAWSALBConfig_zeroValueName() string {
	return fmt.Sprintf(`
resource "aws_alb" "alb_test" {
  name            = ""
  internal        = true
  security_groups = ["${aws_security_group.alb_test.id}"]
  subnets         = ["${aws_subnet.alb_test.*.id}"]

  idle_timeout = 30
  enable_deletion_protection = false

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

variable "subnets" {
  default = ["10.0.1.0/24", "10.0.2.0/24"]
  type    = "list"
}

data "aws_availability_zones" "available" {}

resource "aws_vpc" "alb_test" {
  cidr_block = "10.0.0.0/16"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.alb_test.id}"

  tags {
    Name = "TestAccAWSALB_basic"
  }
}

resource "aws_subnet" "alb_test" {
  count                   = 2
  vpc_id                  = "${aws_vpc.alb_test.id}"
  cidr_block              = "${element(var.subnets, count.index)}"
  map_public_ip_on_launch = true
  availability_zone       = "${element(data.aws_availability_zones.available.names, count.index)}"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_security_group" "alb_test" {
  name        = "allow_all_alb_test"
  description = "Used for ALB Testing"
  vpc_id      = "${aws_vpc.alb_test.id}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}`)
}

func testAccAWSALBConfig_namePrefix() string {
	return fmt.Sprintf(`
resource "aws_alb" "alb_test" {
  name_prefix     = "tf-lb-"
  internal        = true
  security_groups = ["${aws_security_group.alb_test.id}"]
  subnets         = ["${aws_subnet.alb_test.*.id}"]

  idle_timeout = 30
  enable_deletion_protection = false

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

variable "subnets" {
  default = ["10.0.1.0/24", "10.0.2.0/24"]
  type    = "list"
}

data "aws_availability_zones" "available" {}

resource "aws_vpc" "alb_test" {
  cidr_block = "10.0.0.0/16"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_subnet" "alb_test" {
  count                   = 2
  vpc_id                  = "${aws_vpc.alb_test.id}"
  cidr_block              = "${element(var.subnets, count.index)}"
  map_public_ip_on_launch = true
  availability_zone       = "${element(data.aws_availability_zones.available.names, count.index)}"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_security_group" "alb_test" {
  name        = "allow_all_alb_test"
  description = "Used for ALB Testing"
  vpc_id      = "${aws_vpc.alb_test.id}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}`)
}
func testAccAWSALBConfig_updatedTags(albName string) string {
	return fmt.Sprintf(`resource "aws_alb" "alb_test" {
  name            = "%s"
  internal        = true
  security_groups = ["${aws_security_group.alb_test.id}"]
  subnets         = ["${aws_subnet.alb_test.*.id}"]

  idle_timeout = 30
  enable_deletion_protection = false

  tags {
    Environment = "Production"
    Type = "Sample Type Tag"
  }
}

variable "subnets" {
  default = ["10.0.1.0/24", "10.0.2.0/24"]
  type    = "list"
}

data "aws_availability_zones" "available" {}

resource "aws_vpc" "alb_test" {
  cidr_block = "10.0.0.0/16"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_subnet" "alb_test" {
  count                   = 2
  vpc_id                  = "${aws_vpc.alb_test.id}"
  cidr_block              = "${element(var.subnets, count.index)}"
  map_public_ip_on_launch = true
  availability_zone       = "${element(data.aws_availability_zones.available.names, count.index)}"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_security_group" "alb_test" {
  name        = "allow_all_alb_test"
  description = "Used for ALB Testing"
  vpc_id      = "${aws_vpc.alb_test.id}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}`, albName)
}

func testAccAWSALBConfig_accessLogs(enabled bool, albName, bucketName string) string {
	return fmt.Sprintf(`resource "aws_alb" "alb_test" {
  name            = "%s"
  internal        = true
  security_groups = ["${aws_security_group.alb_test.id}"]
  subnets         = ["${aws_subnet.alb_test.*.id}"]

  idle_timeout = 50
  enable_deletion_protection = false

  access_logs {
  	bucket = "${aws_s3_bucket.logs.bucket}"
  	prefix = "${var.bucket_prefix}"
  	enabled = "%t"
  }

  tags {
    TestName = "TestAccAWSALB_basic1"
  }
}

variable "bucket_name" {
  type    = "string"
  default = "%s"
}

variable "bucket_prefix" {
  type    = "string"
  default = "testAccAWSALBConfig_accessLogs"
}

resource "aws_s3_bucket" "logs" {
  bucket = "${var.bucket_name}"
  policy = "${data.aws_iam_policy_document.logs_bucket.json}"
  # dangerous, only here for the test...
  force_destroy = true

  tags {
    Name = "ALB Logs Bucket Test"
  }
}

data "aws_caller_identity" "current" {}

data "aws_elb_service_account" "current" {}

data "aws_iam_policy_document" "logs_bucket" {
  statement {
    actions   = ["s3:PutObject"]
    effect    = "Allow"
    resources = ["arn:aws:s3:::${var.bucket_name}/${var.bucket_prefix}/AWSLogs/${data.aws_caller_identity.current.account_id}/*"]

    principals = {
      type        = "AWS"
      identifiers = ["arn:aws:iam::${data.aws_elb_service_account.current.id}:root"]
    }
  }
}

variable "subnets" {
  default = ["10.0.1.0/24", "10.0.2.0/24"]
  type    = "list"
}

data "aws_availability_zones" "available" {}

resource "aws_vpc" "alb_test" {
  cidr_block = "10.0.0.0/16"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_subnet" "alb_test" {
  count                   = 2
  vpc_id                  = "${aws_vpc.alb_test.id}"
  cidr_block              = "${element(var.subnets, count.index)}"
  map_public_ip_on_launch = true
  availability_zone       = "${element(data.aws_availability_zones.available.names, count.index)}"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_security_group" "alb_test" {
  name        = "allow_all_alb_test"
  description = "Used for ALB Testing"
  vpc_id      = "${aws_vpc.alb_test.id}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}`, albName, enabled, bucketName)
}

func testAccAWSALBConfig_nosg(albName string) string {
	return fmt.Sprintf(`resource "aws_alb" "alb_test" {
  name            = "%s"
  internal        = true
  subnets         = ["${aws_subnet.alb_test.*.id}"]

  idle_timeout = 30
  enable_deletion_protection = false

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

variable "subnets" {
  default = ["10.0.1.0/24", "10.0.2.0/24"]
  type    = "list"
}

data "aws_availability_zones" "available" {}

resource "aws_vpc" "alb_test" {
  cidr_block = "10.0.0.0/16"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_subnet" "alb_test" {
  count                   = 2
  vpc_id                  = "${aws_vpc.alb_test.id}"
  cidr_block              = "${element(var.subnets, count.index)}"
  map_public_ip_on_launch = true
  availability_zone       = "${element(data.aws_availability_zones.available.names, count.index)}"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}`, albName)
}

func testAccAWSALBConfig_updateSecurityGroups(albName string) string {
	return fmt.Sprintf(`resource "aws_alb" "alb_test" {
  name            = "%s"
  internal        = true
  security_groups = ["${aws_security_group.alb_test.id}", "${aws_security_group.alb_test_2.id}"]
  subnets         = ["${aws_subnet.alb_test.*.id}"]

  idle_timeout = 30
  enable_deletion_protection = false

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

variable "subnets" {
  default = ["10.0.1.0/24", "10.0.2.0/24"]
  type    = "list"
}

data "aws_availability_zones" "available" {}

resource "aws_vpc" "alb_test" {
  cidr_block = "10.0.0.0/16"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_subnet" "alb_test" {
  count                   = 2
  vpc_id                  = "${aws_vpc.alb_test.id}"
  cidr_block              = "${element(var.subnets, count.index)}"
  map_public_ip_on_launch = true
  availability_zone       = "${element(data.aws_availability_zones.available.names, count.index)}"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_security_group" "alb_test_2" {
  name        = "allow_all_alb_test_2"
  description = "Used for ALB Testing"
  vpc_id      = "${aws_vpc.alb_test.id}"

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "TCP"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    TestName = "TestAccAWSALB_basic_2"
  }
}

resource "aws_security_group" "alb_test" {
  name        = "allow_all_alb_test"
  description = "Used for ALB Testing"
  vpc_id      = "${aws_vpc.alb_test.id}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}`, albName)
}
