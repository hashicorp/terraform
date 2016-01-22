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
                    output "private_key_pem" {
                        value = "${tls_private_key.test.private_key_pem}"
                    }
                    output "public_key_pem" {
                        value = "${tls_private_key.test.public_key_pem}"
                    }
                    output "public_key_openssh" {
                        value = "${tls_private_key.test.public_key_openssh}"
                    }
                `,
				Check: func(s *terraform.State) error {
					gotPrivate := s.RootModule().Outputs["private_key_pem"]
					if !strings.HasPrefix(gotPrivate, "-----BEGIN RSA PRIVATE KEY----") {
						return fmt.Errorf("private key is missing RSA key PEM preamble")
					}
					if len(gotPrivate) > 1700 {
						return fmt.Errorf("private key PEM looks too long for a 2048-bit key (got %v characters)", len(gotPrivate))
					}

					gotPublic := s.RootModule().Outputs["public_key_pem"]
					if !strings.HasPrefix(gotPublic, "-----BEGIN PUBLIC KEY----") {
						return fmt.Errorf("public key is missing public key PEM preamble")
					}

					gotPublicSSH := s.RootModule().Outputs["public_key_openssh"]
					if !strings.HasPrefix(gotPublicSSH, "ssh-rsa ") {
						return fmt.Errorf("SSH public key is missing ssh-rsa prefix")
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
                    output "private_key_pem" {
                        value = "${tls_private_key.test.private_key_pem}"
                    }
                    output "public_key_pem" {
                        value = "${tls_private_key.test.public_key_pem}"
                    }
                    output "public_key_openssh" {
                        value = "${tls_private_key.test.public_key_openssh}"
                    }
                `,
				Check: func(s *terraform.State) error {
					gotPrivate := s.RootModule().Outputs["private_key_pem"]
					if !strings.HasPrefix(gotPrivate, "-----BEGIN EC PRIVATE KEY----") {
						return fmt.Errorf("Private key is missing EC key PEM preamble")
					}

					gotPublic := s.RootModule().Outputs["public_key_pem"]
					if !strings.HasPrefix(gotPublic, "-----BEGIN PUBLIC KEY----") {
						return fmt.Errorf("public key is missing public key PEM preamble")
					}

					gotPublicSSH := s.RootModule().Outputs["public_key_openssh"]
					if gotPublicSSH != "" {
						return fmt.Errorf("P224 EC key should not generate OpenSSH public key")
					}

					return nil
				},
			},
			r.TestStep{
				Config: `
                    resource "tls_private_key" "test" {
                        algorithm = "ECDSA"
                        ecdsa_curve = "P256"
                    }
                    output "private_key_pem" {
                        value = "${tls_private_key.test.private_key_pem}"
                    }
                    output "public_key_pem" {
                        value = "${tls_private_key.test.public_key_pem}"
                    }
                    output "public_key_openssh" {
                        value = "${tls_private_key.test.public_key_openssh}"
                    }
                `,
				Check: func(s *terraform.State) error {
					gotPrivate := s.RootModule().Outputs["private_key_pem"]
					if !strings.HasPrefix(gotPrivate, "-----BEGIN EC PRIVATE KEY----") {
						return fmt.Errorf("Private key is missing EC key PEM preamble")
					}

					gotPublic := s.RootModule().Outputs["public_key_pem"]
					if !strings.HasPrefix(gotPublic, "-----BEGIN PUBLIC KEY----") {
						return fmt.Errorf("public key is missing public key PEM preamble")
					}

					gotPublicSSH := s.RootModule().Outputs["public_key_openssh"]
					if !strings.HasPrefix(gotPublicSSH, "ecdsa-sha2-nistp256 ") {
						return fmt.Errorf("P256 SSH public key is missing ecdsa prefix")
					}

					return nil
				},
			},
		},
	})
}
