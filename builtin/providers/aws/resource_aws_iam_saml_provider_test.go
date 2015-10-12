package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIAMSamlProvider_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMSamlProviderDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccIAMSamlProviderConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMSamlProvider("aws_iam_saml_provider.salesforce"),
				),
			},
			resource.TestStep{
				Config: testAccIAMSamlProviderConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMSamlProvider("aws_iam_saml_provider.salesforce"),
				),
			},
		},
	})
}

func testAccCheckIAMSamlProviderDestroy(s *terraform.State) error {
	if len(s.RootModule().Resources) > 0 {
		return fmt.Errorf("Expected all resources to be gone, but found: %#v", s.RootModule().Resources)
	}

	return nil
}

func testAccCheckIAMSamlProvider(id string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("Not Found: %s", id)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		iamconn := testAccProvider.Meta().(*AWSClient).iamconn
		_, err := iamconn.GetSAMLProvider(&iam.GetSAMLProviderInput{
			SAMLProviderArn: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		return nil
	}
}

const testAccIAMSamlProviderConfig = `
resource "aws_iam_saml_provider" "salesforce" {
    name = "tf-salesforce-test"
    saml_metadata_document = "${file("./test-fixtures/saml-metadata.xml")}"
}
`

const testAccIAMSamlProviderConfigUpdate = `
resource "aws_iam_saml_provider" "salesforce" {
    name = "tf-salesforce-test"
    saml_metadata_document = "${file("./test-fixtures/saml-metadata-modified.xml")}"
}
`
