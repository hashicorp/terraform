package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIAMOpenIDConnectProvider_basic(t *testing.T) {
	rString := acctest.RandString(5)
	url := "accounts.google.com/" + rString

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMOpenIDConnectProviderDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccIAMOpenIDConnectProviderConfig(rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMOpenIDConnectProvider("aws_iam_openid_connect_provider.goog"),
					resource.TestCheckResourceAttr("aws_iam_openid_connect_provider.goog", "url", url),
					resource.TestCheckResourceAttr("aws_iam_openid_connect_provider.goog", "client_id_list.#", "1"),
					resource.TestCheckResourceAttr("aws_iam_openid_connect_provider.goog", "client_id_list.0",
						"266362248691-re108qaeld573ia0l6clj2i5ac7r7291.apps.googleusercontent.com"),
					resource.TestCheckResourceAttr("aws_iam_openid_connect_provider.goog", "thumbprint_list.#", "0"),
				),
			},
			resource.TestStep{
				Config: testAccIAMOpenIDConnectProviderConfig_modified(rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMOpenIDConnectProvider("aws_iam_openid_connect_provider.goog"),
					resource.TestCheckResourceAttr("aws_iam_openid_connect_provider.goog", "url", url),
					resource.TestCheckResourceAttr("aws_iam_openid_connect_provider.goog", "client_id_list.#", "1"),
					resource.TestCheckResourceAttr("aws_iam_openid_connect_provider.goog", "client_id_list.0",
						"266362248691-re108qaeld573ia0l6clj2i5ac7r7291.apps.googleusercontent.com"),
					resource.TestCheckResourceAttr("aws_iam_openid_connect_provider.goog", "thumbprint_list.#", "2"),
					resource.TestCheckResourceAttr("aws_iam_openid_connect_provider.goog", "thumbprint_list.0", "cf23df2207d99a74fbe169e3eba035e633b65d94"),
					resource.TestCheckResourceAttr("aws_iam_openid_connect_provider.goog", "thumbprint_list.1", "c784713d6f9cb67b55dd84f4e4af7832d42b8f55"),
				),
			},
		},
	})
}

func TestAccAWSIAMOpenIDConnectProvider_importBasic(t *testing.T) {
	resourceName := "aws_iam_openid_connect_provider.goog"
	rString := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMOpenIDConnectProviderDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccIAMOpenIDConnectProviderConfig_modified(rString),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSIAMOpenIDConnectProvider_disappears(t *testing.T) {
	rString := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMOpenIDConnectProviderDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccIAMOpenIDConnectProviderConfig(rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIAMOpenIDConnectProvider("aws_iam_openid_connect_provider.goog"),
					testAccCheckIAMOpenIDConnectProviderDisappears("aws_iam_openid_connect_provider.goog"),
				),
				ExpectNonEmptyPlan: true,
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

func testAccCheckIAMOpenIDConnectProviderDisappears(id string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("Not Found: %s", id)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		iamconn := testAccProvider.Meta().(*AWSClient).iamconn
		_, err := iamconn.DeleteOpenIDConnectProvider(&iam.DeleteOpenIDConnectProviderInput{
			OpenIDConnectProviderArn: aws.String(rs.Primary.ID),
		})
		return err
	}
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

func testAccIAMOpenIDConnectProviderConfig(rString string) string {
	return fmt.Sprintf(`
resource "aws_iam_openid_connect_provider" "goog" {
  url="https://accounts.google.com/%s"
  client_id_list = [
     "266362248691-re108qaeld573ia0l6clj2i5ac7r7291.apps.googleusercontent.com"
  ]
  thumbprint_list = []
}
`, rString)
}

func testAccIAMOpenIDConnectProviderConfig_modified(rString string) string {
	return fmt.Sprintf(`
resource "aws_iam_openid_connect_provider" "goog" {
  url="https://accounts.google.com/%s"
  client_id_list = [
     "266362248691-re108qaeld573ia0l6clj2i5ac7r7291.apps.googleusercontent.com"
  ]
  thumbprint_list = ["cf23df2207d99a74fbe169e3eba035e633b65d94", "c784713d6f9cb67b55dd84f4e4af7832d42b8f55"]
}
`, rString)
}
