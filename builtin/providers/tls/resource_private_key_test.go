package tls

import (
	"fmt"
	"strings"
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestPrivateKeyRSA(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		Steps: []r.TestStep{
			r.TestStep{
				Config: `
                    resource "tls_private_key" "test" {
                        algorithm = "RSA"
                    }
                    output "key_pem" {
                        value = "${tls_private_key.test.private_key_pem}"
                    }
                `,
				Check: func(s *terraform.State) error {
					got := s.RootModule().Outputs["key_pem"]
					if !strings.HasPrefix(got, "-----BEGIN RSA PRIVATE KEY----") {
						return fmt.Errorf("key is missing RSA key PEM preamble")
					}
					if len(got) > 1700 {
						return fmt.Errorf("key PEM looks too long for a 2048-bit key (got %v characters)", len(got))
					}
					return nil
				},
			},
			r.TestStep{
				Config: `
                    resource "tls_private_key" "test" {
                        algorithm = "RSA"
                        rsa_bits = 4096
                    }
                    output "key_pem" {
                        value = "${tls_private_key.test.private_key_pem}"
                    }
                `,
				Check: func(s *terraform.State) error {
					got := s.RootModule().Outputs["key_pem"]
					if !strings.HasPrefix(got, "-----BEGIN RSA PRIVATE KEY----") {
						return fmt.Errorf("key is missing RSA key PEM preamble")
					}
					if len(got) < 1700 {
						return fmt.Errorf("key PEM looks too short for a 4096-bit key (got %v characters)", len(got))
					}
					return nil
				},
			},
		},
	})
}

func TestPrivateKeyECDSA(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		Steps: []r.TestStep{
			r.TestStep{
				Config: `
                    resource "tls_private_key" "test" {
                        algorithm = "ECDSA"
                    }
                    output "key_pem" {
                        value = "${tls_private_key.test.private_key_pem}"
                    }
                `,
				Check: func(s *terraform.State) error {
					got := s.RootModule().Outputs["key_pem"]
					if !strings.HasPrefix(got, "-----BEGIN EC PRIVATE KEY----") {
						return fmt.Errorf("Key is missing EC key PEM preamble")
					}
					return nil
				},
			},
		},
	})
}
