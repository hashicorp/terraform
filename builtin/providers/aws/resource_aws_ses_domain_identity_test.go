package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAwsSESDomainIdentity_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsSESDomainIdentityDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccAwsSESDomainIdentityConfig,
					acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum),
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsSESDomainIdentityExists("aws_ses_domain_identity.test"),
				),
			},
		},
	})
}

func testAccCheckAwsSESDomainIdentityDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).sesConn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ses_domain_identity" {
			continue
		}

		domain := rs.Primary.ID
		params := &ses.GetIdentityVerificationAttributesInput{
			Identities: []*string{
				aws.String(domain),
			},
		}

		response, err := conn.GetIdentityVerificationAttributes(params)
		if err != nil {
			return err
		}

		if response.VerificationAttributes[domain] != nil {
			return fmt.Errorf("SES Domain Identity %s still exists. Failing!", domain)
		}
	}

	return nil
}

func testAccCheckAwsSESDomainIdentityExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("SES Domain Identity not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("SES Domain Identity name not set")
		}

		domain := rs.Primary.ID
		conn := testAccProvider.Meta().(*AWSClient).sesConn

		params := &ses.GetIdentityVerificationAttributesInput{
			Identities: []*string{
				aws.String(domain),
			},
		}

		response, err := conn.GetIdentityVerificationAttributes(params)
		if err != nil {
			return err
		}

		if response.VerificationAttributes[domain] == nil {
			return fmt.Errorf("SES Domain Identity %s not found in AWS", domain)
		}

		return nil
	}
}

const testAccAwsSESDomainIdentityConfig = `
resource "aws_ses_domain_identity" "test" {
	domain = "%s.terraformtesting.com"
}
`
