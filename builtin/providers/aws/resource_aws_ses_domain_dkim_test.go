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

func TestAccAwsSESDomainDkim_basic(t *testing.T) {
	domain := fmt.Sprintf(
		"%s.terraformtesting.com",
		acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAwsSESDomainDkimConfig, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsSESDomainDkimExists("aws_ses_domain_dkim.test"),
					testAccCheckAwsSESDomainDkimArn("aws_ses_domain_dkim.test", domain),
				),
			},
		},
	})
}

func testAccCheckAwsSESDomainDkimExists(n string) resource.TestCheckFunc {
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

		params := &ses.GetIdentityDkimAttributesInput{
			Identities: []*string{
				aws.String(domain),
			},
		}

		response, err := conn.GetIdentityDkimAttributes(params)
		if err != nil {
			return err
		}

		if response.DkimAttributes[domain] == nil {
			return fmt.Errorf("SES Domain DKIM %s not found in AWS", domain)
		}

		return nil
	}
}

func testAccCheckAwsSESDomainDkimArn(n string, domain string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, _ := s.RootModule().Resources[n]

		expected := fmt.Sprintf(
			"arn:%s:ses:%s:%s:dkim/%s",
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

const testAccAwsSESDomainDkimConfig = `
resource "aws_ses_domain_dkim" "test" {
	domain = "%s"
}
`
