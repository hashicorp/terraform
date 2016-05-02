package aws

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIAMServerCertificate_basic(t *testing.T) {
	var cert iam.ServerCertificate

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMServerCertificateDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccIAMServerCertConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCertExists("aws_iam_server_certificate.test_cert", &cert),
					testAccCheckAWSServerCertAttributes(&cert),
				),
			},
		},
	})
}

func TestAccAWSIAMServerCertificate_name_prefix(t *testing.T) {
	var cert iam.ServerCertificate

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMServerCertificateDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccIAMServerCertConfig_random,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCertExists("aws_iam_server_certificate.test_cert", &cert),
					testAccCheckAWSServerCertAttributes(&cert),
				),
			},
		},
	})
}

func TestAccAWSIAMServerCertificate_disappears(t *testing.T) {
	var cert iam.ServerCertificate

	testDestroyCert := func(*terraform.State) error {
		// reach out and DELETE the Cert
		conn := testAccProvider.Meta().(*AWSClient).iamconn
		_, err := conn.DeleteServerCertificate(&iam.DeleteServerCertificateInput{
			ServerCertificateName: cert.ServerCertificateMetadata.ServerCertificateName,
		})

		if err != nil {
			return fmt.Errorf("Error destorying cert in test: %s", err)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMServerCertificateDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccIAMServerCertConfig_random,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCertExists("aws_iam_server_certificate.test_cert", &cert),
					testAccCheckAWSServerCertAttributes(&cert),
					testDestroyCert,
				),
				ExpectNonEmptyPlan: true,
			},
			// Follow up plan w/ empty config should be empty, since the Cert is gone
			resource.TestStep{
				Config: "",
			},
		},
	})
}

func testAccCheckCertExists(n string, cert *iam.ServerCertificate) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Server Cert ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).iamconn
		describeOpts := &iam.GetServerCertificateInput{
			ServerCertificateName: aws.String(rs.Primary.Attributes["name"]),
		}
		resp, err := conn.GetServerCertificate(describeOpts)
		if err != nil {
			return err
		}

		*cert = *resp.ServerCertificate

		return nil
	}
}

func testAccCheckAWSServerCertAttributes(cert *iam.ServerCertificate) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if !strings.Contains(*cert.ServerCertificateMetadata.ServerCertificateName, "terraform-test-cert") {
			return fmt.Errorf("Bad Server Cert Name: %s", *cert.ServerCertificateMetadata.ServerCertificateName)
		}

		if *cert.CertificateBody != strings.TrimSpace(certBody) {
			return fmt.Errorf("Bad Server Cert body\n\t expected: %s\n\tgot: %s\n", certBody, *cert.CertificateBody)
		}
		return nil
	}
}

func testAccCheckIAMServerCertificateDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).iamconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iam_server_certificate" {
			continue
		}

		// Try to find the Cert
		opts := &iam.GetServerCertificateInput{
			ServerCertificateName: aws.String(rs.Primary.Attributes["name"]),
		}
		resp, err := conn.GetServerCertificate(opts)
		if err == nil {
			if resp.ServerCertificate != nil {
				return fmt.Errorf("Error: Server Cert still exists")
			}

			return nil
		}

	}

	return nil
}

var certBody = fmt.Sprintf(`
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
-----END CERTIFICATE-----`)

var testAccIAMServerCertConfig = fmt.Sprintf(`
resource "aws_iam_server_certificate" "test_cert" {
  name = "terraform-test-cert-%d"
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
`, rand.New(rand.NewSource(time.Now().UnixNano())).Int())

var testAccIAMServerCertConfig_random = `
resource "aws_iam_server_certificate" "test_cert" {
  name_prefix = "terraform-test-cert"
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
`
