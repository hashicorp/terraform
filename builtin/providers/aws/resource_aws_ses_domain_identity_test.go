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
	domain := fmt.Sprintf(
		"%s.terraformtesting.com",
		acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsSESDomainIdentityDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAwsSESDomainIdentityConfig, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsSESDomainIdentityExists("aws_ses_domain_identity.test"),
					testAccCheckAwsSESDomainIdentityArn("aws_ses_domain_identity.test", domain),
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

func testAccCheckAwsSESDomainIdentityArn(n string, domain string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, _ := s.RootModule().Resources[n]

		expected := fmt.Sprintf(
			"arn:%s:ses:%s:%s:identity/%s",
			testAccProvider.Meta().(*AWSClient).partition,
			testAccProvider.Meta().(*AWSClient).region,
			testAccProvider.Meta().(*AWSClient).accountid,
			domain)

		if rs.Primary.Attributes["arn"] != expected {
			return fmt.Errorf("Incorrect ARN: expected %q, got %q", expected, rs.Primary.Attributes["arn"])
		}

		return nil
	}
}

const testAccAwsSESDomainIdentityConfig = `
resource "aws_ses_domain_identity" "test" {
	domain = "%s"
}
`
