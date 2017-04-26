package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIoTCertificate_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSIoTCertificateDestroy_basic,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSIoTCertificate_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSIoTCertificateExists_basic("aws_iot_certificate.foo_cert"),
				),
			},
		},
	})
}

func testAccCheckAWSIoTCertificateDestroy_basic(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).iotconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iot_certificate" {
			continue
		}

		out, err := conn.ListCertificates(&iot.ListCertificatesInput{})

		if err != nil {
			return err
		}

		for _, t := range out.Certificates {
			if *t.CertificateId == rs.Primary.ID {
				return fmt.Errorf("IoT certificate still exists:\n%s", t)
			}
		}

	}

	return nil
}

func testAccCheckAWSIoTCertificateExists_basic(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		return nil
	}
}

var testAccAWSIoTCertificate_basic = `
resource "aws_iot_certificate" "foo_cert" {
	csr = "${file("test-fixtures/csr.pem")}"
  active = true
}
`
