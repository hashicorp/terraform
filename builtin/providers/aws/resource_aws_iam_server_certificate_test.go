package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIAMServerCertificate_basic(t *testing.T) {
	var cert iam.ServerCertificate
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMServerCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIAMServerCertConfig(rInt),
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
			{
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
			return fmt.Errorf("Error destroying cert in test: %s", err)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMServerCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIAMServerCertConfig_random,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCertExists("aws_iam_server_certificate.test_cert", &cert),
					testAccCheckAWSServerCertAttributes(&cert),
					testDestroyCert,
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSIAMServerCertificate_file(t *testing.T) {
	var cert iam.ServerCertificate

	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMServerCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIAMServerCertConfig_file(rInt, "iam-ssl-unix-line-endings"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCertExists("aws_iam_server_certificate.test_cert", &cert),
					testAccCheckAWSServerCertAttributes(&cert),
				),
			},

			{
				Config: testAccIAMServerCertConfig_file(rInt, "iam-ssl-windows-line-endings"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCertExists("aws_iam_server_certificate.test_cert", &cert),
					testAccCheckAWSServerCertAttributes(&cert),
				),
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
-----END CERTIFICATE-----`)

func testAccIAMServerCertConfig(rInt int) string {
	return fmt.Sprintf(`
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
`, rInt)
}

var testAccIAMServerCertConfig_random = `
resource "aws_iam_server_certificate" "test_cert" {
  name_prefix = "terraform-test-cert"
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
`

// iam-ssl-unix-line-endings
func testAccIAMServerCertConfig_file(rInt int, fName string) string {
	return fmt.Sprintf(`
resource "aws_iam_server_certificate" "test_cert" {
  name = "terraform-test-cert-%d"
  certificate_body = "${file("test-fixtures/%s.pem")}"

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
`, rInt, fName)
}
