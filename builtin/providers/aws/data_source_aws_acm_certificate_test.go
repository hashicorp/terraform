package aws

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAwsAcmCertificateDataSource_basic(t *testing.T) {
	region := os.Getenv("AWS_ACM_TEST_REGION")
	domain := os.Getenv("AWS_ACM_TEST_DOMAIN")
	certArn := os.Getenv("AWS_ACM_TEST_CERT_ARN")
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if region == "" {
				t.Skip("AWS_ACM_TEST_REGION must be set to a region an ACM certificate pre-created for this test.")
			}
			if domain == "" {
				t.Skip("AWS_ACM_TEST_DOMAIN must be set to a domain with an ACM certificate pre-created for this test.")
			}
			if certArn == "" {
				t.Skip("AWS_ACM_TEST_CERT_ARN must be set to the ARN of an ACM cert pre-created for this test.")
			}
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsAcmCertificateDataSourceConfig(region, domain),
				Check:  testAccCheckAcmArnMatches("data.aws_acm_certificate.test", certArn),
			},
		},
	})
}

func testAccCheckAcmArnMatches(name, expectArn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ms := s.RootModule()
		rs, ok := ms.Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		gotArn := rs.Primary.Attributes["arn"]
		if gotArn != expectArn {
			return fmt.Errorf("Expected cert to have arn: %s, got: %s", expectArn, gotArn)
		}
		return nil
	}
}

func testAccCheckAwsAcmCertificateDataSourceConfig(region, domain string) string {
	return fmt.Sprintf(`
provider "aws" {
	region = "%s"
}
data "aws_acm_certificate" "test" {
	domain = "%s"
}
	`, region, domain)
}
