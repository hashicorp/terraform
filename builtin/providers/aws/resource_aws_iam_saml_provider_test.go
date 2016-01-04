package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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
	iamconn := testAccProvider.Meta().(*AWSClient).iamconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iam_saml_provider" {
			continue
		}

		input := &iam.GetSAMLProviderInput{
			SAMLProviderArn: aws.String(rs.Primary.ID),
		}
		out, err := iamconn.GetSAMLProvider(input)
		if err != nil {
			if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" {
				// none found, that's good
				return nil
			}
			return fmt.Errorf("Error reading IAM SAML Provider, out: %s, err: %s", out, err)
		}

		if out != nil {
			return fmt.Errorf("Found IAM SAML Provider, expected none: %s", out)
		}
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
