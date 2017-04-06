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

func TestAccAWSIAMOpenIDConnectProvider_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMOpenIDConnectProviderDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccIAMOpenIDConnectProviderConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMOpenIDConnectProvider("aws_iam_openid_connect_provider.goog"),
				),
			},
		},
	})
}

func testAccCheckIAMOpenIDConnectProviderDestroy(s *terraform.State) error {
	iamconn := testAccProvider.Meta().(*AWSClient).iamconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iam_openid_connect_provider" {
			continue
		}

		input := &iam.GetOpenIDConnectProviderInput{
			OpenIDConnectProviderArn: aws.String(rs.Primary.ID),
		}
		out, err := iamconn.GetOpenIDConnectProvider(input)
		if err != nil {
			if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" {
				// none found, that's good
				return nil
			}
			return fmt.Errorf("Error reading IAM OpenID Connect Provider, out: %s, err: %s", out, err)
		}

		if out != nil {
			return fmt.Errorf("Found IAM OpenID Connect Provider, expected none: %s", out)
		}
	}

	return nil
}

func testAccCheckIAMOpenIDConnectProvider(id string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("Not Found: %s", id)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		iamconn := testAccProvider.Meta().(*AWSClient).iamconn
		_, err := iamconn.GetOpenIDConnectProvider(&iam.GetOpenIDConnectProviderInput{
			OpenIDConnectProviderArn: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		return nil
	}
}

const testAccIAMOpenIDConnectProviderConfig = `
resource "aws_iam_openid_connect_provider" "goog" {
  url="https://accounts.google.com"
  client_id_list = [
     "266362248691-re108qaeld573ia0l6clj2i5ac7r7291.apps.googleusercontent.com"
  ]
  thumbprint_list = []
}
`
