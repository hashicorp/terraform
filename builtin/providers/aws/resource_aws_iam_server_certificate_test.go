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

func TestAccIAMServerCertificate_basic(t *testing.T) {
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
		if !strings.HasPrefix(*cert.ServerCertificateMetadata.ServerCertificateName, "terraform-test-cert") {
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
MIIExDCCA6ygAwIBAgIJALX7Jt7ddT3eMA0GCSqGSIb3DQEBBQUAMIGcMQswCQYD
VQQGEwJVUzERMA8GA1UECBMITWlzc291cmkxETAPBgNVBAcTCENvbHVtYmlhMRIw
EAYDVQQKEwlIYXNoaUNvcnAxEjAQBgNVBAsTCVRlcnJhZm9ybTEbMBkGA1UEAxMS
d3d3Lm5vdGV4YW1wbGUuY29tMSIwIAYJKoZIhvcNAQkBFhNjbGludEBoYXNoaWNv
cnAuY29tMB4XDTE1MDUyNjE0MzA1MloXDTE4MDUyNTE0MzA1MlowgZwxCzAJBgNV
BAYTAlVTMREwDwYDVQQIEwhNaXNzb3VyaTERMA8GA1UEBxMIQ29sdW1iaWExEjAQ
BgNVBAoTCUhhc2hpQ29ycDESMBAGA1UECxMJVGVycmFmb3JtMRswGQYDVQQDExJ3
d3cubm90ZXhhbXBsZS5jb20xIjAgBgkqhkiG9w0BCQEWE2NsaW50QGhhc2hpY29y
cC5jb20wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCownyOIKXBbYxh
PynVAw30eaJj2OmilFJagwGeFHMT0rErCodY8lAQsPz6gj83NC9D4MzDt1H+GmoR
MSDphJEUxxTxvaNWTTN5sZ9WvE+sbw5YkkTXc4DmVsVMoa3urQO20f0tcHXyULj0
sXbtG+q/QhKxqeFjYON46Z6l7x32d/cj4mIcXwLpIf+W2wpvXCKAc8851skJ+O9W
UW0/h/ivwwkKfzGfiObL16IUaq+fxwnkYt3fUI2Z4rSKAULMEcquzfKr3JR6wkeI
J66ZSb6fMNlCPGPcINDhzwSgGRpqRqeuRl4Z9m2fZaaYVltHqjwDH1tKr+3qXFnv
nZmq7pzJAgMBAAGjggEFMIIBATAdBgNVHQ4EFgQUO8bEvPq+V/rtnlhTxQDusR7o
n6QwgdEGA1UdIwSByTCBxoAUO8bEvPq+V/rtnlhTxQDusR7on6ShgaKkgZ8wgZwx
CzAJBgNVBAYTAlVTMREwDwYDVQQIEwhNaXNzb3VyaTERMA8GA1UEBxMIQ29sdW1i
aWExEjAQBgNVBAoTCUhhc2hpQ29ycDESMBAGA1UECxMJVGVycmFmb3JtMRswGQYD
VQQDExJ3d3cubm90ZXhhbXBsZS5jb20xIjAgBgkqhkiG9w0BCQEWE2NsaW50QGhh
c2hpY29ycC5jb22CCQC1+ybe3XU93jAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEB
BQUAA4IBAQBsJ/NP1uBYm+8ejrpUu2mipT5JfBahpiUxef5BeubSrSM3zmdrtLLA
+DdDkrt0AfOaasBXMTEwrR3NunBAmn/6PX0r/PAjlqk/tOVBnASC9t3cmi88fO10
gQw+se86MiCr/hTavq2YTQZ652+ksjxeQwyHIzKrYS/rRGPKKHX70H5Asb1CY44p
/GRyLvAckzZ1Gp64ym6XCLTS53wOur6wLX1/lqshBo2utUmm/2a/XF4psSDx/k2J
E2oHzGoJ2F/+QkiXHzvPcUXRFVhXkQnZDocCv/nhcEwNkN9Z1OxCNqsZw+FiJm2E
FVSdVaOstOHOVllblhWxvjm55a44feFX
-----END CERTIFICATE-----`)

var testAccIAMServerCertConfig = fmt.Sprintf(`
resource "aws_iam_server_certificate" "test_cert" {
  name = "terraform-test-cert-%d"
  certificate_body = <<EOF
-----BEGIN CERTIFICATE-----
MIIExDCCA6ygAwIBAgIJALX7Jt7ddT3eMA0GCSqGSIb3DQEBBQUAMIGcMQswCQYD
VQQGEwJVUzERMA8GA1UECBMITWlzc291cmkxETAPBgNVBAcTCENvbHVtYmlhMRIw
EAYDVQQKEwlIYXNoaUNvcnAxEjAQBgNVBAsTCVRlcnJhZm9ybTEbMBkGA1UEAxMS
d3d3Lm5vdGV4YW1wbGUuY29tMSIwIAYJKoZIhvcNAQkBFhNjbGludEBoYXNoaWNv
cnAuY29tMB4XDTE1MDUyNjE0MzA1MloXDTE4MDUyNTE0MzA1MlowgZwxCzAJBgNV
BAYTAlVTMREwDwYDVQQIEwhNaXNzb3VyaTERMA8GA1UEBxMIQ29sdW1iaWExEjAQ
BgNVBAoTCUhhc2hpQ29ycDESMBAGA1UECxMJVGVycmFmb3JtMRswGQYDVQQDExJ3
d3cubm90ZXhhbXBsZS5jb20xIjAgBgkqhkiG9w0BCQEWE2NsaW50QGhhc2hpY29y
cC5jb20wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCownyOIKXBbYxh
PynVAw30eaJj2OmilFJagwGeFHMT0rErCodY8lAQsPz6gj83NC9D4MzDt1H+GmoR
MSDphJEUxxTxvaNWTTN5sZ9WvE+sbw5YkkTXc4DmVsVMoa3urQO20f0tcHXyULj0
sXbtG+q/QhKxqeFjYON46Z6l7x32d/cj4mIcXwLpIf+W2wpvXCKAc8851skJ+O9W
UW0/h/ivwwkKfzGfiObL16IUaq+fxwnkYt3fUI2Z4rSKAULMEcquzfKr3JR6wkeI
J66ZSb6fMNlCPGPcINDhzwSgGRpqRqeuRl4Z9m2fZaaYVltHqjwDH1tKr+3qXFnv
nZmq7pzJAgMBAAGjggEFMIIBATAdBgNVHQ4EFgQUO8bEvPq+V/rtnlhTxQDusR7o
n6QwgdEGA1UdIwSByTCBxoAUO8bEvPq+V/rtnlhTxQDusR7on6ShgaKkgZ8wgZwx
CzAJBgNVBAYTAlVTMREwDwYDVQQIEwhNaXNzb3VyaTERMA8GA1UEBxMIQ29sdW1i
aWExEjAQBgNVBAoTCUhhc2hpQ29ycDESMBAGA1UECxMJVGVycmFmb3JtMRswGQYD
VQQDExJ3d3cubm90ZXhhbXBsZS5jb20xIjAgBgkqhkiG9w0BCQEWE2NsaW50QGhh
c2hpY29ycC5jb22CCQC1+ybe3XU93jAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEB
BQUAA4IBAQBsJ/NP1uBYm+8ejrpUu2mipT5JfBahpiUxef5BeubSrSM3zmdrtLLA
+DdDkrt0AfOaasBXMTEwrR3NunBAmn/6PX0r/PAjlqk/tOVBnASC9t3cmi88fO10
gQw+se86MiCr/hTavq2YTQZ652+ksjxeQwyHIzKrYS/rRGPKKHX70H5Asb1CY44p
/GRyLvAckzZ1Gp64ym6XCLTS53wOur6wLX1/lqshBo2utUmm/2a/XF4psSDx/k2J
E2oHzGoJ2F/+QkiXHzvPcUXRFVhXkQnZDocCv/nhcEwNkN9Z1OxCNqsZw+FiJm2E
FVSdVaOstOHOVllblhWxvjm55a44feFX
-----END CERTIFICATE-----
EOF

	private_key =  <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAqMJ8jiClwW2MYT8p1QMN9HmiY9jpopRSWoMBnhRzE9KxKwqH
WPJQELD8+oI/NzQvQ+DMw7dR/hpqETEg6YSRFMcU8b2jVk0zebGfVrxPrG8OWJJE
13OA5lbFTKGt7q0DttH9LXB18lC49LF27Rvqv0ISsanhY2DjeOmepe8d9nf3I+Ji
HF8C6SH/ltsKb1wigHPPOdbJCfjvVlFtP4f4r8MJCn8xn4jmy9eiFGqvn8cJ5GLd
31CNmeK0igFCzBHKrs3yq9yUesJHiCeumUm+nzDZQjxj3CDQ4c8EoBkaakanrkZe
GfZtn2WmmFZbR6o8Ax9bSq/t6lxZ752Zqu6cyQIDAQABAoIBAFq2kHVlnzPmSvtL
FJVn2ux7JYs+Yff+enYkzY3HuEQDkTBtrGtndRpDyPhvYsOtzWpTQD5EIFLSqAkt
u19K3yGoEd4P7ejJ/s1/aQMankk2OSPrHA4kDDnEkrGqhvAxGDoBjnIKbZwfQAxo
CGFUDE9amOnfQ0REJIIuMhVH/3coDbsLUQstf43xU0Hl8C7vEPZ0WjX0LKZJsQEg
TVp4kF/ohxCsl/rtasrrCTIEWLgsqzYoIs5EivxZR6FwudaM3C/Dlcad3rISHXCe
w+dBWhIKqr7c695wkbtKb9x44hAKYAZdPG2eupy+SlQhm1mkl3aV3g6ylwEwhMJW
8nV3WcECgYEA0phtwHvXhRuSaCjTHAMf3DBqi8GDg+x9oKDPpFNROjot/rUqkfX8
vpHR7roN5bqX/4nDe/CfJE5UmJBbGcKXburRkwRqteT30N3qx9X+Yla9lPXbKo2F
TE+jQO5oZ7IdKdycyPOoFvlB0RB8c7pLwXlwWaybvb/2sqEcL0a3zW0CgYEAzST7
6YXMKKNvbw9vivm+LenpV6BVk5SOLXbMWtKmOjKLmc2dlD38jL2ZTlWzTbbAncta
Kkd4hDwH7rnXueMWwJjhRXKweNG5BsNhSl0NyQeujzHmxT54wGS8nYk48xGZuMNa
F0tfodJ2OA2IzyQHHzZ7YJPt85533Wj9CPt4P00CgYBi6T7bIg9msD2CeHI2/Oyw
4XiZbWlUw/V5RS5hUtSa0Yqa0AJPjcaIxzpfsrkmRg5v8geDpc9JIRUwltSC89dm
PBn0wCVSi1ktm51TAJo7G9xtI1At20xZPCpEK/WThp+V8s0cwPwY1jdodyLMxBoi
o+P16lE3vPqkiXEQb1mSvQKBgQCRLddJkHLHX8KA6n+Z7tx0SdHlPYbShpOIAUbm
D6WsEhFRq34VZzjPsW5JTcUy/l6aTUtmGGZlzsYeYE8XMmrrqkXijCPvnRxAeQzl
P619033pwPr8JBX4slH5ex9ehdowM7ASRDlNoFAhoxJq5ahUoo317zq66i8R9jb8
oFqdEQKBgCKhmR2NCfeVuklmnGGKOqpHD5IUnirmsm6/AtIk4HOR7NO+YOEmw9QN
0bOSNJPRbWAAZtvWT9/eJi61cAm2UhGN8iTSEWM+ixAn8v+9G+KzcNjccZu39ZCf
GWrW9VPbefh6Bhfdh34IQexYbivSxvi4ZZB4lJ8ce0rPJihQILBl
-----END RSA PRIVATE KEY-----
EOF
}
`, rand.New(rand.NewSource(time.Now().UnixNano())).Int())
