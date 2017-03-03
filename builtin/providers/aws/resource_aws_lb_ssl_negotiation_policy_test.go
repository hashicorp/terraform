package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elb"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSLBSSLNegotiationPolicy_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBSSLNegotiationPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccSslNegotiationPolicyConfig(
					fmt.Sprintf("tf-acctest-%s", acctest.RandString(10)), fmt.Sprintf("tf-test-lb-%s", acctest.RandString(5))),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBSSLNegotiationPolicy(
						"aws_elb.lb",
						"aws_lb_ssl_negotiation_policy.foo",
					),
					resource.TestCheckResourceAttr(
						"aws_lb_ssl_negotiation_policy.foo", "attribute.#", "7"),
				),
			},
		},
	})
}

func TestAccAWSLBSSLNegotiationPolicy_missingLB(t *testing.T) {
	lbName := fmt.Sprintf("tf-test-lb-%s", acctest.RandString(5))

	// check that we can destroy the policy if the LB is missing
	removeLB := func() {
		conn := testAccProvider.Meta().(*AWSClient).elbconn
		deleteElbOpts := elb.DeleteLoadBalancerInput{
			LoadBalancerName: aws.String(lbName),
		}
		if _, err := conn.DeleteLoadBalancer(&deleteElbOpts); err != nil {
			t.Fatalf("Error deleting ELB: %s", err)
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBSSLNegotiationPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccSslNegotiationPolicyConfig(fmt.Sprintf("tf-acctest-%s", acctest.RandString(10)), lbName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBSSLNegotiationPolicy(
						"aws_elb.lb",
						"aws_lb_ssl_negotiation_policy.foo",
					),
					resource.TestCheckResourceAttr(
						"aws_lb_ssl_negotiation_policy.foo", "attribute.#", "7"),
				),
			},
			resource.TestStep{
				PreConfig: removeLB,
				Config:    testAccSslNegotiationPolicyConfig(fmt.Sprintf("tf-acctest-%s", acctest.RandString(10)), lbName),
			},
		},
	})
}

func testAccCheckLBSSLNegotiationPolicyDestroy(s *terraform.State) error {
	elbconn := testAccProvider.Meta().(*AWSClient).elbconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elb" && rs.Type != "aws_lb_ssl_negotiation_policy" {
			continue
		}

		// Check that the ELB is destroyed
		if rs.Type == "aws_elb" {
			describe, err := elbconn.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
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
		} else {
			// Check that the SSL Negotiation Policy is destroyed
			elbName, _, policyName := resourceAwsLBSSLNegotiationPolicyParseId(rs.Primary.ID)
			_, err := elbconn.DescribeLoadBalancerPolicies(&elb.DescribeLoadBalancerPoliciesInput{
				LoadBalancerName: aws.String(elbName),
				PolicyNames:      []*string{aws.String(policyName)},
			})

			if err == nil {
				return fmt.Errorf("ELB SSL Negotiation Policy still exists")
			}
		}
	}

	return nil
}

func testAccCheckLBSSLNegotiationPolicy(elbResource string, policyResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[elbResource]
		if !ok {
			return fmt.Errorf("Not found: %s", elbResource)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		policy, ok := s.RootModule().Resources[policyResource]
		if !ok {
			return fmt.Errorf("Not found: %s", policyResource)
		}

		elbconn := testAccProvider.Meta().(*AWSClient).elbconn

		elbName, _, policyName := resourceAwsLBSSLNegotiationPolicyParseId(policy.Primary.ID)
		resp, err := elbconn.DescribeLoadBalancerPolicies(&elb.DescribeLoadBalancerPoliciesInput{
			LoadBalancerName: aws.String(elbName),
			PolicyNames:      []*string{aws.String(policyName)},
		})

		if err != nil {
			fmt.Printf("[ERROR] Problem describing load balancer policy '%s': %s", policyName, err)
			return err
		}

		if len(resp.PolicyDescriptions) != 1 {
			return fmt.Errorf("Unable to find policy %#v", resp.PolicyDescriptions)
		}

		attrmap := policyAttributesToMap(&resp.PolicyDescriptions[0].PolicyAttributeDescriptions)
		if attrmap["Protocol-TLSv1"] != "false" {
			return fmt.Errorf("Policy attribute 'Protocol-TLSv1' was of value %s instead of false!", attrmap["Protocol-TLSv1"])
		}
		if attrmap["Protocol-TLSv1.1"] != "false" {
			return fmt.Errorf("Policy attribute 'Protocol-TLSv1.1' was of value %s instead of false!", attrmap["Protocol-TLSv1.1"])
		}
		if attrmap["Protocol-TLSv1.2"] != "true" {
			return fmt.Errorf("Policy attribute 'Protocol-TLSv1.2' was of value %s instead of true!", attrmap["Protocol-TLSv1.2"])
		}
		if attrmap["Server-Defined-Cipher-Order"] != "true" {
			return fmt.Errorf("Policy attribute 'Server-Defined-Cipher-Order' was of value %s instead of true!", attrmap["Server-Defined-Cipher-Order"])
		}
		if attrmap["ECDHE-RSA-AES128-GCM-SHA256"] != "true" {
			return fmt.Errorf("Policy attribute 'ECDHE-RSA-AES128-GCM-SHA256' was of value %s instead of true!", attrmap["ECDHE-RSA-AES128-GCM-SHA256"])
		}
		if attrmap["AES128-GCM-SHA256"] != "true" {
			return fmt.Errorf("Policy attribute 'AES128-GCM-SHA256' was of value %s instead of true!", attrmap["AES128-GCM-SHA256"])
		}
		if attrmap["EDH-RSA-DES-CBC3-SHA"] != "false" {
			return fmt.Errorf("Policy attribute 'EDH-RSA-DES-CBC3-SHA' was of value %s instead of false!", attrmap["EDH-RSA-DES-CBC3-SHA"])
		}

		return nil
	}
}

func policyAttributesToMap(attributes *[]*elb.PolicyAttributeDescription) map[string]string {
	attrmap := make(map[string]string)

	for _, attrdef := range *attributes {
		attrmap[*attrdef.AttributeName] = *attrdef.AttributeValue
	}

	return attrmap
}

// Sets the SSL Negotiation policy with attributes.
// The IAM Server Cert config is lifted from
// builtin/providers/aws/resource_aws_iam_server_certificate_test.go
func testAccSslNegotiationPolicyConfig(certName string, lbName string) string {
	return fmt.Sprintf(`
resource "aws_iam_server_certificate" "test_cert" {
  name = "%s"
  certificate_body = <<EOF
-----BEGIN CERTIFICATE-----
MIICqzCCAhSgAwIBAgIJAOH3Ca1oeCfOMA0GCSqGSIb3DQEBBQUAME4xCzAJBgNV
BAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRIwEAYDVQQKEwlIYXNoaWNvcnAx
FjAUBgNVBAMTDWhhc2hpY29ycC5jb20wHhcNMTYwODEwMTcxNDEwWhcNMTcwODEw
MTcxNDEwWjBkMQswCQYDVQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEUMBIG
A1UEBwwLTG9zIEFuZ2VsZXMxEjAQBgNVBAoMCUhhc2hpY29ycDEWMBQGA1UEAwwN
aGFzaGljb3JwLmNvbTCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAlQMKKTiK
bawxxGOwX9iyIm/ITyVwjnSyyZ8kuz7flXUAw4u/ZqGmRck0gdOBlzPcvdu/ngCZ
wMg6x03oe7iouDQHapQ6kCAUwl6zDmSOnjj8b4fKiaxW6Kw/UynrUjbjbdqKKsH3
fBYxa1sIVhnsDBCaOnnznkCXFbeiMeUX6YkCAwEAAaN7MHkwCQYDVR0TBAIwADAs
BglghkgBhvhCAQ0EHxYdT3BlblNTTCBHZW5lcmF0ZWQgQ2VydGlmaWNhdGUwHQYD
VR0OBBYEFB+VNDp3tesqOLJTZEbOXIzINdecMB8GA1UdIwQYMBaAFDnmEwagl6fs
/9oVTSmNdPUkhaRDMA0GCSqGSIb3DQEBBQUAA4GBAHMTokhZfM66L1dI8e21p4yp
F2GMGYNqR2CLy7pCk3z9NovB5F1plk1cDnbpJPS/jXU7N5i3LgfjjbYmlNsezV3u
gzYm7p7D6/AiMheL6VljPor5ZXXcq2yZ3xMJu6/hrSJGj0wtg9xsNPYPDGCyH+iI
zAYQVBuFaLoTi3Fs7g1s
-----END CERTIFICATE-----
EOF
  certificate_chain = <<EOF
-----BEGIN CERTIFICATE-----
MIICyzCCAjSgAwIBAgIJAOH3Ca1oeCfNMA0GCSqGSIb3DQEBBQUAME4xCzAJBgNV
BAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRIwEAYDVQQKEwlIYXNoaWNvcnAx
FjAUBgNVBAMTDWhhc2hpY29ycC5jb20wHhcNMTYwODEwMTcxMTAzWhcNMTkwODEw
MTcxMTAzWjBOMQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTESMBAG
A1UEChMJSGFzaGljb3JwMRYwFAYDVQQDEw1oYXNoaWNvcnAuY29tMIGfMA0GCSqG
SIb3DQEBAQUAA4GNADCBiQKBgQDOOIUDgTP+v6yXq0cI99S99jrczNv274BfmBzS
XhExPnm62s5dnLGtzFokat/DIN0pyOh0C4+QnS4Qk7r31UCh1jLJRVkJJHtet8TM
7PhebIUIAFaQQ5+792L7ZkCXkzl0MxENeE0avGUf5QXMd7/eUt36BOS4KaEfGVUw
2Ldy0wIDAQABo4GwMIGtMB0GA1UdDgQWBBQ55hMGoJen7P/aFU0pjXT1JIWkQzB+
BgNVHSMEdzB1gBQ55hMGoJen7P/aFU0pjXT1JIWkQ6FSpFAwTjELMAkGA1UEBhMC
VVMxEzARBgNVBAgTCkNhbGlmb3JuaWExEjAQBgNVBAoTCUhhc2hpY29ycDEWMBQG
A1UEAxMNaGFzaGljb3JwLmNvbYIJAOH3Ca1oeCfNMAwGA1UdEwQFMAMBAf8wDQYJ
KoZIhvcNAQEFBQADgYEAvKhhRHHWuUl253pjlQJxHqJLv3a9g7pcF0vGkImw30lu
B0LFpM6xZmfoFR3aflTWDGHDbwNbP+VatZNwZt7GpO7qiLOXCV9/UM0utxI1Doyd
6oOaCDXtDDI9NliSFyAvNG5PKafR3ysWHsqEa/7VDWnRGYvCAIsaAEyurl4Gogk=
-----END CERTIFICATE-----
EOF
  private_key =  <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQCVAwopOIptrDHEY7Bf2LIib8hPJXCOdLLJnyS7Pt+VdQDDi79m
oaZFyTSB04GXM9y927+eAJnAyDrHTeh7uKi4NAdqlDqQIBTCXrMOZI6eOPxvh8qJ
rFborD9TKetSNuNt2ooqwfd8FjFrWwhWGewMEJo6efOeQJcVt6Ix5RfpiQIDAQAB
AoGAdx8p9U/84bXhRxVGfyi1JvBjmlncxBUohCPT8lhN1qXlSW2jQgGB8ZHqhsq1
c1GDaseMRFxIjaPD0WZHrvgs73ReoDGTLf9Ne3mkE3g8Rp0Bg8CFG8ZFHvCbzAtQ
F441nXsa/E3fUajfuxOeIEz8sJUG8VpMMtNUGB2cmJxzlYECQQDGosn4g0trBkn+
wwwJ3CEnymTUZxgFQWr4UhGnScRHaHBJmw0sW9KsVOB5D4DEw/O7BDdVvpCoBlG1
GhL/XFcZAkEAwAuINbY5jKTpa2Xve1MUJXpgGpuraYWCXaAn9sdSUhm6wHONhDHr
O0S0a3P0aMA5M4GQ5JHeUq53r8/2oP2j8QJBAIzObu+8WqT2Y1O1/f2rTtF/FnS+
0/c9xU9cFemJUBryfM6gm/j66l+BF1KZ28UfxtGmjnc4zCBfwmHnptngIlkCQFv5
aeuncRptpKjd8frTSBPG7x3vLgHkghIK8Pjcbw2I6wrejIkiSzFgbzQDHavJW9vS
Eq2VOq/IhOO7qrdholECQQDFmlx7LQsVEOQ26xQX/ieZQolfDqZLA6zhJFec3k2l
wbEcTx10meJdinnhawqW7L0bhifeiTaPxbaCBXv/wiiL
-----END RSA PRIVATE KEY-----
EOF
}
resource "aws_elb" "lb" {
	name = "%s"
	availability_zones = ["us-west-2a"]
	listener {
		instance_port = 8000
		instance_protocol = "https"
		lb_port = 443
		lb_protocol = "https"
		ssl_certificate_id = "${aws_iam_server_certificate.test_cert.arn}"
	}
}
resource "aws_lb_ssl_negotiation_policy" "foo" {
	name = "foo-policy"
	load_balancer = "${aws_elb.lb.id}"
	lb_port = 443
	attribute {
    	name = "Protocol-TLSv1"
        value = "false"
    }
    attribute {
        name = "Protocol-TLSv1.1"
        value = "false"
    }
    attribute {
        name = "Protocol-TLSv1.2"
        value = "true"
    }
    attribute {
        name = "Server-Defined-Cipher-Order"
        value = "true"
    }
    attribute {
        name = "ECDHE-RSA-AES128-GCM-SHA256"
        value = "true"
    }
    attribute {
        name = "AES128-GCM-SHA256"
        value = "true"
    }
    attribute {
        name = "EDH-RSA-DES-CBC3-SHA"
        value = "false"
    }
}
`, certName, lbName)
}
