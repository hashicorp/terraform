package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elb"

	tlsprovider "github.com/hashicorp/terraform/builtin/providers/tls"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSLoadBalancerBackendServerPolicy_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Providers: map[string]terraform.ResourceProvider{
			"aws": testAccProvider,
			"tls": tlsprovider.Provider(),
		},
		CheckDestroy: testAccCheckAWSLoadBalancerBackendServerPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSLoadBalancerBackendServerPolicyConfig_basic0,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLoadBalancerPolicyState("aws_elb.test-lb", "aws_load_balancer_policy.test-pubkey-policy0"),
					testAccCheckAWSLoadBalancerPolicyState("aws_elb.test-lb", "aws_load_balancer_policy.test-backend-auth-policy0"),
					testAccCheckAWSLoadBalancerBackendServerPolicyState("test-aws-policies-lb", "test-backend-auth-policy0", true),
				),
			},
			resource.TestStep{
				Config: testAccAWSLoadBalancerBackendServerPolicyConfig_basic1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLoadBalancerPolicyState("aws_elb.test-lb", "aws_load_balancer_policy.test-pubkey-policy0"),
					testAccCheckAWSLoadBalancerPolicyState("aws_elb.test-lb", "aws_load_balancer_policy.test-pubkey-policy1"),
					testAccCheckAWSLoadBalancerPolicyState("aws_elb.test-lb", "aws_load_balancer_policy.test-backend-auth-policy0"),
					testAccCheckAWSLoadBalancerBackendServerPolicyState("test-aws-policies-lb", "test-backend-auth-policy0", true),
				),
			},
			resource.TestStep{
				Config: testAccAWSLoadBalancerBackendServerPolicyConfig_basic2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLoadBalancerBackendServerPolicyState("test-aws-policies-lb", "test-backend-auth-policy0", false),
				),
			},
		},
	})
}

func policyInBackendServerPolicies(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

func testAccCheckAWSLoadBalancerBackendServerPolicyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elbconn

	for _, rs := range s.RootModule().Resources {
		switch {
		case rs.Type == "aws_load_balancer_policy":
			loadBalancerName, policyName := resourceAwsLoadBalancerBackendServerPoliciesParseId(rs.Primary.ID)
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
		case rs.Type == "aws_load_balancer_backend_policy":
			loadBalancerName, policyName := resourceAwsLoadBalancerBackendServerPoliciesParseId(rs.Primary.ID)
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
			for _, backendServer := range out.LoadBalancerDescriptions[0].BackendServerDescriptions {
				policyStrings := []string{}
				for _, pol := range backendServer.PolicyNames {
					policyStrings = append(policyStrings, *pol)
				}
				if policyInBackendServerPolicies(policyName, policyStrings) {
					return fmt.Errorf("Policy still exists and is assigned")
				}
			}
		default:
			continue
		}
	}
	return nil
}

func testAccCheckAWSLoadBalancerBackendServerPolicyState(loadBalancerName string, loadBalancerBackendAuthPolicyName string, assigned bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		elbconn := testAccProvider.Meta().(*AWSClient).elbconn

		loadBalancerDescription, err := elbconn.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
			LoadBalancerNames: []*string{aws.String(loadBalancerName)},
		})
		if err != nil {
			return err
		}

		for _, backendServer := range loadBalancerDescription.LoadBalancerDescriptions[0].BackendServerDescriptions {
			policyStrings := []string{}
			for _, pol := range backendServer.PolicyNames {
				policyStrings = append(policyStrings, *pol)
			}
			if policyInBackendServerPolicies(loadBalancerBackendAuthPolicyName, policyStrings) != assigned {
				if assigned {
					return fmt.Errorf("Policy no longer assigned %s not in %+v", loadBalancerBackendAuthPolicyName, policyStrings)
				} else {
					return fmt.Errorf("Policy exists and is assigned")
				}
			}
		}

		return nil
	}
}

const testAccAWSLoadBalancerBackendServerPolicyConfig_basic0 = `
resource "tls_private_key" "example0" {
    algorithm = "RSA"
}

resource "tls_self_signed_cert" "test-cert0" {
    key_algorithm = "RSA"
    private_key_pem = "${tls_private_key.example0.private_key_pem}"

    subject {
        common_name = "example.com"
        organization = "ACME Examples, Inc"
    }

    validity_period_hours = 12

    allowed_uses = [
        "key_encipherment",
        "digital_signature",
        "server_auth",
    ]
}

resource "aws_iam_server_certificate" "test-iam-cert0" {
  name_prefix = "test_cert_"
  certificate_body = "${tls_self_signed_cert.test-cert0.cert_pem}"
  private_key = "${tls_private_key.example0.private_key_pem}"
}

resource "aws_elb" "test-lb" {
  name = "test-aws-policies-lb"
  availability_zones = ["us-west-2a"]

  listener {
    instance_port = 443
    instance_protocol = "https"
    lb_port = 443
    lb_protocol = "https"
    ssl_certificate_id = "${aws_iam_server_certificate.test-iam-cert0.arn}"
  }

  tags {
    Name = "tf-acc-test"
  }
}

resource "aws_load_balancer_policy" "test-pubkey-policy0" {
  load_balancer_name = "${aws_elb.test-lb.name}"
  policy_name = "test-pubkey-policy0"
  policy_type_name = "PublicKeyPolicyType"
  policy_attribute = {
    name = "PublicKey"
    value = "${replace(replace(replace(tls_private_key.example0.public_key_pem, "\n", ""), "-----BEGIN PUBLIC KEY-----", ""), "-----END PUBLIC KEY-----", "")}"
  }
}

resource "aws_load_balancer_policy" "test-backend-auth-policy0" {
  load_balancer_name = "${aws_elb.test-lb.name}"
  policy_name = "test-backend-auth-policy0"
  policy_type_name = "BackendServerAuthenticationPolicyType"
  policy_attribute = {
    name = "PublicKeyPolicyName"
    value = "${aws_load_balancer_policy.test-pubkey-policy0.policy_name}"
  }
}

resource "aws_load_balancer_backend_server_policy" "test-backend-auth-policies-443" {
  load_balancer_name = "${aws_elb.test-lb.name}"
  instance_port = 443
  policy_names = [
    "${aws_load_balancer_policy.test-backend-auth-policy0.policy_name}"
  ]
}
`

const testAccAWSLoadBalancerBackendServerPolicyConfig_basic1 = `
resource "tls_private_key" "example0" {
    algorithm = "RSA"
}

resource "tls_self_signed_cert" "test-cert0" {
    key_algorithm = "RSA"
    private_key_pem = "${tls_private_key.example0.private_key_pem}"

    subject {
        common_name = "example.com"
        organization = "ACME Examples, Inc"
    }

    validity_period_hours = 12

    allowed_uses = [
        "key_encipherment",
        "digital_signature",
        "server_auth",
    ]
}

resource "tls_private_key" "example1" {
    algorithm = "RSA"
}

resource "tls_self_signed_cert" "test-cert1" {
    key_algorithm = "RSA"
    private_key_pem = "${tls_private_key.example1.private_key_pem}"

    subject {
        common_name = "example.com"
        organization = "ACME Examples, Inc"
    }

    validity_period_hours = 12

    allowed_uses = [
        "key_encipherment",
        "digital_signature",
        "server_auth",
    ]
}

resource "aws_iam_server_certificate" "test-iam-cert0" {
  name_prefix = "test_cert_"
  certificate_body = "${tls_self_signed_cert.test-cert0.cert_pem}"
  private_key = "${tls_private_key.example0.private_key_pem}"
}

resource "aws_elb" "test-lb" {
  name = "test-aws-policies-lb"
  availability_zones = ["us-west-2a"]

  listener {
    instance_port = 443
    instance_protocol = "https"
    lb_port = 443
    lb_protocol = "https"
    ssl_certificate_id = "${aws_iam_server_certificate.test-iam-cert0.arn}"
  }

  tags {
    Name = "tf-acc-test"
  }
}

resource "aws_load_balancer_policy" "test-pubkey-policy0" {
  load_balancer_name = "${aws_elb.test-lb.name}"
  policy_name = "test-pubkey-policy0"
  policy_type_name = "PublicKeyPolicyType"
  policy_attribute = {
    name = "PublicKey"
    value = "${replace(replace(replace(tls_private_key.example0.public_key_pem, "\n", ""), "-----BEGIN PUBLIC KEY-----", ""), "-----END PUBLIC KEY-----", "")}"
  }
}

resource "aws_load_balancer_policy" "test-pubkey-policy1" {
  load_balancer_name = "${aws_elb.test-lb.name}"
  policy_name = "test-pubkey-policy1"
  policy_type_name = "PublicKeyPolicyType"
  policy_attribute = {
    name = "PublicKey"
    value = "${replace(replace(replace(tls_private_key.example1.public_key_pem, "\n", ""), "-----BEGIN PUBLIC KEY-----", ""), "-----END PUBLIC KEY-----", "")}"
  }
}

resource "aws_load_balancer_policy" "test-backend-auth-policy0" {
  load_balancer_name = "${aws_elb.test-lb.name}"
  policy_name = "test-backend-auth-policy0"
  policy_type_name = "BackendServerAuthenticationPolicyType"
  policy_attribute = {
    name = "PublicKeyPolicyName"
    value = "${aws_load_balancer_policy.test-pubkey-policy1.policy_name}"
  }
}

resource "aws_load_balancer_backend_server_policy" "test-backend-auth-policies-443" {
  load_balancer_name = "${aws_elb.test-lb.name}"
  instance_port = 443
  policy_names = [
    "${aws_load_balancer_policy.test-backend-auth-policy0.policy_name}"
  ]
}
`

const testAccAWSLoadBalancerBackendServerPolicyConfig_basic2 = `
resource "tls_private_key" "example0" {
    algorithm = "RSA"
}

resource "tls_self_signed_cert" "test-cert0" {
    key_algorithm = "RSA"
    private_key_pem = "${tls_private_key.example0.private_key_pem}"

    subject {
        common_name = "example.com"
        organization = "ACME Examples, Inc"
    }

    validity_period_hours = 12

    allowed_uses = [
        "key_encipherment",
        "digital_signature",
        "server_auth",
    ]
}

resource "tls_private_key" "example1" {
    algorithm = "RSA"
}

resource "tls_self_signed_cert" "test-cert1" {
    key_algorithm = "RSA"
    private_key_pem = "${tls_private_key.example1.private_key_pem}"

    subject {
        common_name = "example.com"
        organization = "ACME Examples, Inc"
    }

    validity_period_hours = 12

    allowed_uses = [
        "key_encipherment",
        "digital_signature",
        "server_auth",
    ]
}

resource "aws_iam_server_certificate" "test-iam-cert0" {
  name_prefix = "test_cert_"
  certificate_body = "${tls_self_signed_cert.test-cert0.cert_pem}"
  private_key = "${tls_private_key.example0.private_key_pem}"
}

resource "aws_elb" "test-lb" {
  name = "test-aws-policies-lb"
  availability_zones = ["us-west-2a"]

  listener {
    instance_port = 443
    instance_protocol = "https"
    lb_port = 443
    lb_protocol = "https"
    ssl_certificate_id = "${aws_iam_server_certificate.test-iam-cert0.arn}"
  }

  tags {
    Name = "tf-acc-test"
  }
}
`
