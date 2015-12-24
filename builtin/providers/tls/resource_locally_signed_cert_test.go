package tls

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestLocallySignedCert(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		Steps: []r.TestStep{
			r.TestStep{
				Config: fmt.Sprintf(`
                    resource "tls_locally_signed_cert" "test" {
                        cert_request_pem = <<EOT
%s
EOT

                        validity_period_hours = 1

                        allowed_uses = [
                            "key_encipherment",
                            "digital_signature",
                            "server_auth",
                            "client_auth",
                        ]

                        ca_key_algorithm = "RSA"
                        ca_cert_pem = <<EOT
%s
EOT
                        ca_private_key_pem = <<EOT
%s
EOT
                    }
                    output "cert_pem" {
                        value = "${tls_locally_signed_cert.test.cert_pem}"
                    }
                `, testCertRequest, testCACert, testCAPrivateKey),
				Check: func(s *terraform.State) error {
					got := s.RootModule().Outputs["cert_pem"]
					if !strings.HasPrefix(got, "-----BEGIN CERTIFICATE----") {
						return fmt.Errorf("key is missing cert PEM preamble")
					}
					block, _ := pem.Decode([]byte(got))
					cert, err := x509.ParseCertificate(block.Bytes)
					if err != nil {
						return fmt.Errorf("error parsing cert: %s", err)
					}
					if expected, got := "2", cert.Subject.SerialNumber; got != expected {
						return fmt.Errorf("incorrect subject serial number: expected %v, got %v", expected, got)
					}
					if expected, got := "example.com", cert.Subject.CommonName; got != expected {
						return fmt.Errorf("incorrect subject common name: expected %v, got %v", expected, got)
					}
					if expected, got := "Example, Inc", cert.Subject.Organization[0]; got != expected {
						return fmt.Errorf("incorrect subject organization: expected %v, got %v", expected, got)
					}
					if expected, got := "Department of Terraform Testing", cert.Subject.OrganizationalUnit[0]; got != expected {
						return fmt.Errorf("incorrect subject organizational unit: expected %v, got %v", expected, got)
					}
					if expected, got := "5879 Cotton Link", cert.Subject.StreetAddress[0]; got != expected {
						return fmt.Errorf("incorrect subject street address: expected %v, got %v", expected, got)
					}
					if expected, got := "Pirate Harbor", cert.Subject.Locality[0]; got != expected {
						return fmt.Errorf("incorrect subject locality: expected %v, got %v", expected, got)
					}
					if expected, got := "CA", cert.Subject.Province[0]; got != expected {
						return fmt.Errorf("incorrect subject province: expected %v, got %v", expected, got)
					}
					if expected, got := "US", cert.Subject.Country[0]; got != expected {
						return fmt.Errorf("incorrect subject country: expected %v, got %v", expected, got)
					}
					if expected, got := "95559-1227", cert.Subject.PostalCode[0]; got != expected {
						return fmt.Errorf("incorrect subject postal code: expected %v, got %v", expected, got)
					}

					if expected, got := 2, len(cert.DNSNames); got != expected {
						return fmt.Errorf("incorrect number of DNS names: expected %v, got %v", expected, got)
					}
					if expected, got := "example.com", cert.DNSNames[0]; got != expected {
						return fmt.Errorf("incorrect DNS name 0: expected %v, got %v", expected, got)
					}
					if expected, got := "example.net", cert.DNSNames[1]; got != expected {
						return fmt.Errorf("incorrect DNS name 0: expected %v, got %v", expected, got)
					}

					if expected, got := 2, len(cert.IPAddresses); got != expected {
						return fmt.Errorf("incorrect number of IP addresses: expected %v, got %v", expected, got)
					}
					if expected, got := "127.0.0.1", cert.IPAddresses[0].String(); got != expected {
						return fmt.Errorf("incorrect IP address 0: expected %v, got %v", expected, got)
					}
					if expected, got := "127.0.0.2", cert.IPAddresses[1].String(); got != expected {
						return fmt.Errorf("incorrect IP address 0: expected %v, got %v", expected, got)
					}

					if expected, got := []byte{50, 174, 195, 33, 77, 223, 57, 1, 58, 166, 246, 243, 114, 109, 59, 64, 111, 9, 198, 144}, cert.AuthorityKeyId; !bytes.Equal(got, expected) {
						return fmt.Errorf("incorrect AuthorityKeyId: expected %v, got %v", expected, got)
					}

					if expected, got := 2, len(cert.ExtKeyUsage); got != expected {
						return fmt.Errorf("incorrect number of ExtKeyUsage: expected %v, got %v", expected, got)
					}
					if expected, got := x509.ExtKeyUsageServerAuth, cert.ExtKeyUsage[0]; got != expected {
						return fmt.Errorf("incorrect ExtKeyUsage[0]: expected %v, got %v", expected, got)
					}
					if expected, got := x509.ExtKeyUsageClientAuth, cert.ExtKeyUsage[1]; got != expected {
						return fmt.Errorf("incorrect ExtKeyUsage[1]: expected %v, got %v", expected, got)
					}

					if expected, got := x509.KeyUsageKeyEncipherment|x509.KeyUsageDigitalSignature, cert.KeyUsage; got != expected {
						return fmt.Errorf("incorrect KeyUsage: expected %v, got %v", expected, got)
					}

					// This time checking is a bit sloppy to avoid inconsistent test results
					// depending on the power of the machine running the tests.
					now := time.Now()
					if cert.NotBefore.After(now) {
						return fmt.Errorf("certificate validity begins in the future")
					}
					if now.Sub(cert.NotBefore) > (2 * time.Minute) {
						return fmt.Errorf("certificate validity begins more than two minutes in the past")
					}
					if cert.NotAfter.Sub(cert.NotBefore) != time.Hour {
						return fmt.Errorf("certificate validity is not one hour")
					}

					caBlock, _ := pem.Decode([]byte(testCACert))
					caCert, err := x509.ParseCertificate(caBlock.Bytes)
					if err != nil {
						return fmt.Errorf("error parsing ca cert: %s", err)
					}
					certPool := x509.NewCertPool()

					// Verify certificate
					_, err = cert.Verify(x509.VerifyOptions{Roots: certPool})
					if err == nil {
						return errors.New("incorrectly verified certificate")
					} else if _, ok := err.(x509.UnknownAuthorityError); !ok {
						return fmt.Errorf("incorrect verify error: expected UnknownAuthorityError, got %v", err)
					}
					certPool.AddCert(caCert)
					if _, err = cert.Verify(x509.VerifyOptions{Roots: certPool}); err != nil {
						return fmt.Errorf("verify failed: %s", err)
					}

					return nil
				},
			},
		},
	})
}
