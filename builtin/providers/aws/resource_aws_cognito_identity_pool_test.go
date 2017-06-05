package aws

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cognitoidentity"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCognitoIdentityPool_basic(t *testing.T) {
	name := fmt.Sprintf("%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	updatedName := fmt.Sprintf("%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoIdentityPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoIdentityPoolConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolExists("aws_cognito_identity_pool.main"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "identity_pool_name", fmt.Sprintf("identity pool %s", name)),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "allow_unauthenticated_identities", "false"),
				),
			},
			{
				Config: testAccAWSCognitoIdentityPoolConfig_basic(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolExists("aws_cognito_identity_pool.main"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "identity_pool_name", fmt.Sprintf("identity pool %s", updatedName)),
				),
			},
		},
	})
}

func TestAccAWSCognitoIdentityPool_supportedLoginProviders(t *testing.T) {
	name := fmt.Sprintf("%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoIdentityPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoIdentityPoolConfig_supportedLoginProviders(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolExists("aws_cognito_identity_pool.main"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "identity_pool_name", fmt.Sprintf("identity pool %s", name)),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "supported_login_providers.graph.facebook.com", "7346241598935555"),
				),
			},
			{
				Config: testAccAWSCognitoIdentityPoolConfig_supportedLoginProvidersModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolExists("aws_cognito_identity_pool.main"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "identity_pool_name", fmt.Sprintf("identity pool %s", name)),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "supported_login_providers.graph.facebook.com", "7346241598935552"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "supported_login_providers.accounts.google.com", "123456789012.apps.googleusercontent.com"),
				),
			},
			{
				Config: testAccAWSCognitoIdentityPoolConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolExists("aws_cognito_identity_pool.main"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "identity_pool_name", fmt.Sprintf("identity pool %s", name)),
				),
			},
		},
	})
}

func TestAccAWSCognitoIdentityPool_openidConnectProviderArns(t *testing.T) {
	name := fmt.Sprintf("%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoIdentityPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoIdentityPoolConfig_openidConnectProviderArns(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolExists("aws_cognito_identity_pool.main"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "identity_pool_name", fmt.Sprintf("identity pool %s", name)),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "openid_connect_provider_arns.#", "1"),
				),
			},
			{
				Config: testAccAWSCognitoIdentityPoolConfig_openidConnectProviderArnsModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolExists("aws_cognito_identity_pool.main"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "identity_pool_name", fmt.Sprintf("identity pool %s", name)),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "openid_connect_provider_arns.#", "2"),
				),
			},
			{
				Config: testAccAWSCognitoIdentityPoolConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolExists("aws_cognito_identity_pool.main"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "identity_pool_name", fmt.Sprintf("identity pool %s", name)),
				),
			},
		},
	})
}

func TestAccAWSCognitoIdentityPool_samlProviderArns(t *testing.T) {
	name := fmt.Sprintf("%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoIdentityPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoIdentityPoolConfig_samlProviderArns(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolExists("aws_cognito_identity_pool.main"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "identity_pool_name", fmt.Sprintf("identity pool %s", name)),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "saml_provider_arns.#", "1"),
				),
			},
			{
				Config: testAccAWSCognitoIdentityPoolConfig_samlProviderArnsModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolExists("aws_cognito_identity_pool.main"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "identity_pool_name", fmt.Sprintf("identity pool %s", name)),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "saml_provider_arns.#", "1"),
				),
			},
			{
				Config: testAccAWSCognitoIdentityPoolConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolExists("aws_cognito_identity_pool.main"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "identity_pool_name", fmt.Sprintf("identity pool %s", name)),
					resource.TestCheckNoResourceAttr("aws_cognito_identity_pool.main", "saml_provider_arns.#"),
				),
			},
		},
	})
}

func TestAccAWSCognitoIdentityPool_cognitoIdentityProviders(t *testing.T) {
	name := fmt.Sprintf("%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoIdentityPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoIdentityPoolConfig_cognitoIdentityProviders(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolExists("aws_cognito_identity_pool.main"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "identity_pool_name", fmt.Sprintf("identity pool %s", name)),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "cognito_identity_providers.66456389.client_id", "7lhlkkfbfb4q5kpp90urffao"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "cognito_identity_providers.66456389.provider_name", "cognito-idp.us-east-1.amazonaws.com/us-east-1_Zr231apJu"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "cognito_identity_providers.66456389.server_side_token_check", "false"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "cognito_identity_providers.3571192419.client_id", "7lhlkkfbfb4q5kpp90urffao"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "cognito_identity_providers.3571192419.provider_name", "cognito-idp.us-east-1.amazonaws.com/us-east-1_Ab129faBb"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "cognito_identity_providers.3571192419.server_side_token_check", "false"),
				),
			},
			{
				Config: testAccAWSCognitoIdentityPoolConfig_cognitoIdentityProvidersModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolExists("aws_cognito_identity_pool.main"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "identity_pool_name", fmt.Sprintf("identity pool %s", name)),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "cognito_identity_providers.3661724441.client_id", "6lhlkkfbfb4q5kpp90urffae"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "cognito_identity_providers.3661724441.provider_name", "cognito-idp.us-east-1.amazonaws.com/us-east-1_Zr231apJu"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "cognito_identity_providers.3661724441.server_side_token_check", "false"),
				),
			},
			{
				Config: testAccAWSCognitoIdentityPoolConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolExists("aws_cognito_identity_pool.main"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool.main", "identity_pool_name", fmt.Sprintf("identity pool %s", name)),
				),
			},
		},
	})
}

func testAccCheckAWSCognitoIdentityPoolExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("No Cognito Identity Pool ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).cognitoconn

		_, err := conn.DescribeIdentityPool(&cognitoidentity.DescribeIdentityPoolInput{
			IdentityPoolId: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckAWSCognitoIdentityPoolDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).cognitoconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_cognito_identity_pool" {
			continue
		}

		_, err := conn.DescribeIdentityPool(&cognitoidentity.DescribeIdentityPoolInput{
			IdentityPoolId: aws.String(rs.Primary.ID),
		})

		if err != nil {
			if wserr, ok := err.(awserr.Error); ok && wserr.Code() == "ResourceNotFoundException" {
				return nil
			}
			return err
		}
	}

	return nil
}

func testAccAWSCognitoIdentityPoolConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "aws_cognito_identity_pool" "main" {
  identity_pool_name               = "identity pool %s"
  allow_unauthenticated_identities = false
  developer_provider_name          = "my.developer"
}
`, name)
}

func testAccAWSCognitoIdentityPoolConfig_supportedLoginProviders(name string) string {
	return fmt.Sprintf(`
resource "aws_cognito_identity_pool" "main" {
  identity_pool_name               = "identity pool %s"
  allow_unauthenticated_identities = false

  supported_login_providers {
    "graph.facebook.com" = "7346241598935555"
  }
}
`, name)
}

func testAccAWSCognitoIdentityPoolConfig_supportedLoginProvidersModified(name string) string {
	return fmt.Sprintf(`
resource "aws_cognito_identity_pool" "main" {
  identity_pool_name               = "identity pool %s"
  allow_unauthenticated_identities = false

  supported_login_providers {
    "graph.facebook.com"  = "7346241598935552"
    "accounts.google.com" = "123456789012.apps.googleusercontent.com"
  }
}
`, name)
}

func testAccAWSCognitoIdentityPoolConfig_openidConnectProviderArns(name string) string {
	return fmt.Sprintf(`
resource "aws_cognito_identity_pool" "main" {
  identity_pool_name               = "identity pool %s"
  allow_unauthenticated_identities = false

  openid_connect_provider_arns = ["arn:aws:iam::123456789012:oidc-provider/server.example.com"]
}
`, name)
}

func testAccAWSCognitoIdentityPoolConfig_openidConnectProviderArnsModified(name string) string {
	return fmt.Sprintf(`
resource "aws_cognito_identity_pool" "main" {
  identity_pool_name               = "identity pool %s"
  allow_unauthenticated_identities = false

  openid_connect_provider_arns = ["arn:aws:iam::123456789012:oidc-provider/foo.example.com", "arn:aws:iam::123456789012:oidc-provider/bar.example.com"]
}
`, name)
}

func testAccAWSCognitoIdentityPoolConfig_samlProviderArns(name string) string {
	return fmt.Sprintf(`
resource "aws_iam_saml_provider" "default" {
  name                   = "myprovider-%s"
  saml_metadata_document = "${file("./test-fixtures/saml-metadata.xml")}"
}

resource "aws_cognito_identity_pool" "main" {
  identity_pool_name               = "identity pool %s"
  allow_unauthenticated_identities = false

  saml_provider_arns = ["${aws_iam_saml_provider.default.arn}"]
}
`, name, name)
}

func testAccAWSCognitoIdentityPoolConfig_samlProviderArnsModified(name string) string {
	return fmt.Sprintf(`
resource "aws_iam_saml_provider" "default" {
  name                   = "default-%s"
  saml_metadata_document = "${file("./test-fixtures/saml-metadata.xml")}"
}

resource "aws_iam_saml_provider" "secondary" {
  name                   = "secondary-%s"
  saml_metadata_document = "${file("./test-fixtures/saml-metadata.xml")}"
}

resource "aws_cognito_identity_pool" "main" {
  identity_pool_name               = "identity pool %s"
  allow_unauthenticated_identities = false

  saml_provider_arns = ["${aws_iam_saml_provider.secondary.arn}"]
}
`, name, name, name)
}

func testAccAWSCognitoIdentityPoolConfig_cognitoIdentityProviders(name string) string {
	return fmt.Sprintf(`
resource "aws_cognito_identity_pool" "main" {
  identity_pool_name               = "identity pool %s"
  allow_unauthenticated_identities = false

  cognito_identity_providers {
    client_id               = "7lhlkkfbfb4q5kpp90urffao"
    provider_name           = "cognito-idp.us-east-1.amazonaws.com/us-east-1_Ab129faBb"
    server_side_token_check = false
  }

  cognito_identity_providers {
    client_id               = "7lhlkkfbfb4q5kpp90urffao"
    provider_name           = "cognito-idp.us-east-1.amazonaws.com/us-east-1_Zr231apJu"
    server_side_token_check = false
  }
}
`, name)
}

func testAccAWSCognitoIdentityPoolConfig_cognitoIdentityProvidersModified(name string) string {
	return fmt.Sprintf(`
resource "aws_cognito_identity_pool" "main" {
  identity_pool_name               = "identity pool %s"
  allow_unauthenticated_identities = false

  cognito_identity_providers {
    client_id               = "6lhlkkfbfb4q5kpp90urffae"
    provider_name           = "cognito-idp.us-east-1.amazonaws.com/us-east-1_Zr231apJu"
    server_side_token_check = false
  }
}
`, name)
}
