package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccComputeSslCertificate_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeSslCertificateDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeSslCertificate_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeSslCertificateExists(
						"google_compute_ssl_certificate.foobar"),
				),
			},
		},
	})
}

func TestAccComputeSslCertificate_no_name(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeSslCertificateDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeSslCertificate_no_name,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeSslCertificateExists(
						"google_compute_ssl_certificate.foobar"),
				),
			},
		},
	})
}

func TestAccComputeSslCertificate_name_prefix(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeSslCertificateDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeSslCertificate_name_prefix,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeSslCertificateExists(
						"google_compute_ssl_certificate.foobar"),
				),
			},
		},
	})
}

func testAccCheckComputeSslCertificateDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_ssl_certificate" {
			continue
		}

		_, err := config.clientCompute.SslCertificates.Get(
			config.Project, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("SslCertificate still exists")
		}
	}

	return nil
}

func testAccCheckComputeSslCertificateExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.SslCertificates.Get(
			config.Project, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Certificate not found")
		}

		return nil
	}
}

var testAccComputeSslCertificate_basic = fmt.Sprintf(`
resource "google_compute_ssl_certificate" "foobar" {
	name = "sslcert-test-%s"
	description = "very descriptive"
	private_key = "${file("test-fixtures/ssl_cert/test.key")}"
	certificate = "${file("test-fixtures/ssl_cert/test.crt")}"
}
`, acctest.RandString(10))

var testAccComputeSslCertificate_no_name = fmt.Sprintf(`
resource "google_compute_ssl_certificate" "foobar" {
	description = "really descriptive"
	private_key = "${file("test-fixtures/ssl_cert/test.key")}"
	certificate = "${file("test-fixtures/ssl_cert/test.crt")}"
}
`)

var testAccComputeSslCertificate_name_prefix = fmt.Sprintf(`
resource "google_compute_ssl_certificate" "foobar" {
	name_prefix = "sslcert-test-%s-"
	description = "extremely descriptive"
	private_key = "${file("test-fixtures/ssl_cert/test.key")}"
	certificate = "${file("test-fixtures/ssl_cert/test.crt")}"
}
`, acctest.RandString(10))
