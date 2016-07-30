package aws

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSKmsSecretsDataSource_basic(t *testing.T) {
	// Run a resource test to setup our KMS key
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsKmsSecretsDataSourceKey,
				Check: func(s *terraform.State) error {
					encryptedPayload, err := testAccCheckAwsKmsSecretsDataSourceCheckKeySetup(s)
					if err != nil {
						return err
					}

					// We run the actual test on our data source nested in the
					// Check function of the KMS key so we can access the
					// encrypted output, above, and so that the key will be
					// deleted at the end of the test
					resource.Test(t, resource.TestCase{
						PreCheck:  func() { testAccPreCheck(t) },
						Providers: testAccProviders,
						Steps: []resource.TestStep{
							{
								Config: fmt.Sprintf(testAccCheckAwsKmsSecretsDataSourceSecrets, encryptedPayload),
								Check: resource.ComposeTestCheckFunc(
									resource.TestCheckResourceAttr("data.aws_kms_secrets.testing", "secret_name", "PAYLOAD"),
								),
							},
						},
					})

					return nil
				},
			},
		},
	})

}

func testAccCheckAwsKmsSecretsDataSourceCheckKeySetup(s *terraform.State) (string, error) {
	rs, ok := s.RootModule().Resources["aws_kms_key.terraform_data_source_testing"]
	if !ok {
		return "", fmt.Errorf("Failed to setup a KMS key for data source testing!")
	}

	// Now that the key is setup encrypt a string using it
	// XXX TODO: Set up and test with grants
	params := &kms.EncryptInput{
		KeyId:     aws.String(rs.Primary.Attributes["arn"]),
		Plaintext: []byte("PAYLOAD"),
		EncryptionContext: map[string]*string{
			"name": aws.String("value"),
		},
	}

	kmsconn := testAccProvider.Meta().(*AWSClient).kmsconn
	resp, err := kmsconn.Encrypt(params)
	if err != nil {
		return "", fmt.Errorf("Failed encrypting string with KMS for data source testing: %s", err)
	}

	return base64.StdEncoding.EncodeToString(resp.CiphertextBlob), nil
}

const testAccCheckAwsKmsSecretsDataSourceKey = `
resource "aws_kms_key" "terraform_data_source_testing" {
    description = "Testing the Terraform AWS KMS Secrets data_source"
}
`

const testAccCheckAwsKmsSecretsDataSourceSecrets = `
data "aws_kms_secrets" "testing" {
    secret {
        name = "secret_name"
        payload = "%s"

        context {
            name = "value"
        }
    }
}
`
