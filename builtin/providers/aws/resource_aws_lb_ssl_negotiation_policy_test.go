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
					fmt.Sprintf("tf-acctest-%s", acctest.RandString(10))),
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
func testAccSslNegotiationPolicyConfig(certName string) string {
	return fmt.Sprintf(`
resource "aws_iam_server_certificate" "test_cert" {
  name = "%s"
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
resource "aws_elb" "lb" {
	name = "test-lb"
    availability_zones = ["us-east-1a"]
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
`, certName)
}
