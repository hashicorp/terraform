package aws

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSALBListener_basic(t *testing.T) {
	var conf elbv2.Listener
	albName := fmt.Sprintf("testlistener-basic-%s", acctest.RandStringFromCharSet(13, acctest.CharSetAlphaNum))
	targetGroupName := fmt.Sprintf("testtargetgroup-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb_listener.front_end",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBListenerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBListenerConfig_basic(albName, targetGroupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBListenerExists("aws_alb_listener.front_end", &conf),
					resource.TestCheckResourceAttrSet("aws_alb_listener.front_end", "load_balancer_arn"),
					resource.TestCheckResourceAttrSet("aws_alb_listener.front_end", "arn"),
					resource.TestCheckResourceAttr("aws_alb_listener.front_end", "protocol", "HTTP"),
					resource.TestCheckResourceAttr("aws_alb_listener.front_end", "port", "80"),
					resource.TestCheckResourceAttr("aws_alb_listener.front_end", "default_action.#", "1"),
					resource.TestCheckResourceAttr("aws_alb_listener.front_end", "default_action.0.type", "forward"),
					resource.TestCheckResourceAttrSet("aws_alb_listener.front_end", "default_action.0.target_group_arn"),
				),
			},
		},
	})
}

func TestAccAWSALBListener_https(t *testing.T) {
	var conf elbv2.Listener
	albName := fmt.Sprintf("testlistener-https-%s", acctest.RandStringFromCharSet(13, acctest.CharSetAlphaNum))
	targetGroupName := fmt.Sprintf("testtargetgroup-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb_listener.front_end",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBListenerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBListenerConfig_https(albName, targetGroupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBListenerExists("aws_alb_listener.front_end", &conf),
					resource.TestCheckResourceAttrSet("aws_alb_listener.front_end", "load_balancer_arn"),
					resource.TestCheckResourceAttrSet("aws_alb_listener.front_end", "arn"),
					resource.TestCheckResourceAttr("aws_alb_listener.front_end", "protocol", "HTTPS"),
					resource.TestCheckResourceAttr("aws_alb_listener.front_end", "port", "443"),
					resource.TestCheckResourceAttr("aws_alb_listener.front_end", "default_action.#", "1"),
					resource.TestCheckResourceAttr("aws_alb_listener.front_end", "default_action.0.type", "forward"),
					resource.TestCheckResourceAttrSet("aws_alb_listener.front_end", "default_action.0.target_group_arn"),
					resource.TestCheckResourceAttrSet("aws_alb_listener.front_end", "certificate_arn"),
					resource.TestCheckResourceAttr("aws_alb_listener.front_end", "ssl_policy", "ELBSecurityPolicy-2015-05"),
				),
			},
		},
	})
}

func testAccCheckAWSALBListenerExists(n string, res *elbv2.Listener) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("No Listener ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).elbv2conn

		describe, err := conn.DescribeListeners(&elbv2.DescribeListenersInput{
			ListenerArns: []*string{aws.String(rs.Primary.ID)},
		})

		if err != nil {
			return err
		}

		if len(describe.Listeners) != 1 ||
			*describe.Listeners[0].ListenerArn != rs.Primary.ID {
			return errors.New("Listener not found")
		}

		*res = *describe.Listeners[0]
		return nil
	}
}

func testAccCheckAWSALBListenerDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elbv2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_alb_listener" {
			continue
		}

		describe, err := conn.DescribeListeners(&elbv2.DescribeListenersInput{
			ListenerArns: []*string{aws.String(rs.Primary.ID)},
		})

		if err == nil {
			if len(describe.Listeners) != 0 &&
				*describe.Listeners[0].ListenerArn == rs.Primary.ID {
				return fmt.Errorf("Listener %q still exists", rs.Primary.ID)
			}
		}

		// Verify the error
		if isListenerNotFound(err) {
			return nil
		} else {
			return errwrap.Wrapf("Unexpected error checking ALB Listener destroyed: {{err}}", err)
		}
	}

	return nil
}

func testAccAWSALBListenerConfig_basic(albName, targetGroupName string) string {
	return fmt.Sprintf(`resource "aws_alb_listener" "front_end" {
   load_balancer_arn = "${aws_alb.alb_test.id}"
   protocol = "HTTP"
   port = "80"

   default_action {
     target_group_arn = "${aws_alb_target_group.test.id}"
     type = "forward"
   }
}

resource "aws_alb" "alb_test" {
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

resource "aws_alb_target_group" "test" {
  name = "%s"
  port = 8080
  protocol = "HTTP"
  vpc_id = "${aws_vpc.alb_test.id}"

  health_check {
    path = "/health"
    interval = 60
    port = 8081
    protocol = "HTTP"
    timeout = 3
    healthy_threshold = 3
    unhealthy_threshold = 3
    matcher = "200-299"
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
}`, albName, targetGroupName)
}

func testAccAWSALBListenerConfig_https(albName, targetGroupName string) string {
	return fmt.Sprintf(`resource "aws_alb_listener" "front_end" {
   load_balancer_arn = "${aws_alb.alb_test.id}"
   protocol = "HTTPS"
   port = "443"
   ssl_policy = "ELBSecurityPolicy-2015-05"
   certificate_arn = "${aws_iam_server_certificate.test_cert.arn}"

   default_action {
     target_group_arn = "${aws_alb_target_group.test.id}"
     type = "forward"
   }
}

resource "aws_alb" "alb_test" {
  name            = "%s"
  internal        = false
  security_groups = ["${aws_security_group.alb_test.id}"]
  subnets         = ["${aws_subnet.alb_test.*.id}"]

  idle_timeout = 30
  enable_deletion_protection = false

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_alb_target_group" "test" {
  name = "%s"
  port = 8080
  protocol = "HTTP"
  vpc_id = "${aws_vpc.alb_test.id}"

  health_check {
    path = "/health"
    interval = 60
    port = 8081
    protocol = "HTTP"
    timeout = 3
    healthy_threshold = 3
    unhealthy_threshold = 3
    matcher = "200-299"
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
}

resource "aws_iam_server_certificate" "test_cert" {
  name = "terraform-test-cert-%d"
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
`, albName, targetGroupName, rand.New(rand.NewSource(time.Now().UnixNano())).Int())
}
