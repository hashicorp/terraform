package aws

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAwsAcmCertificateDataSource_noMatchReturnsError(t *testing.T) {
	domain := "hashicorp.com"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckAwsAcmCertificateDataSourceConfig(domain),
				ExpectError: regexp.MustCompile(`No certificate for domain`),
			},
			{
				Config:      testAccCheckAwsAcmCertificateDataSourceConfigWithStatus(domain),
				ExpectError: regexp.MustCompile(`No certificate for domain`),
			},
			{
				Config:      testAccCheckAwsAcmCertificateDataSourceConfigWithTypes(domain),
				ExpectError: regexp.MustCompile(`No certificate for domain`),
			},
		},
	})
}

func testAccCheckAwsAcmCertificateDataSourceConfig(domain string) string {
	return fmt.Sprintf(`
data "aws_acm_certificate" "test" {
	domain = "%s"
}
`, domain)
}

func testAccCheckAwsAcmCertificateDataSourceConfigWithStatus(domain string) string {
	return fmt.Sprintf(`
data "aws_acm_certificate" "test" {
	domain = "%s"
	statuses = ["ISSUED"]
}
`, domain)
}

func testAccCheckAwsAcmCertificateDataSourceConfigWithTypes(domain string) string {
	return fmt.Sprintf(`
data "aws_acm_certificate" "test" {
	domain = "%s"
	types = ["IMPORTED"]
}
`, domain)
}
