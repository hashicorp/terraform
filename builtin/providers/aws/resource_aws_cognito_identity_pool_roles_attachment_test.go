package aws

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cognitoidentity"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCognitoIdentityPoolRolesAttachment_basic(t *testing.T) {
	name := fmt.Sprintf("%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	updatedName := fmt.Sprintf("%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoIdentityPoolRolesAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoIdentityPoolRolesAttachmentConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolRolesAttachmentExists("aws_cognito_identity_pool_roles_attachment.main"),
					resource.TestCheckResourceAttrSet("aws_cognito_identity_pool_roles_attachment.main", "identity_pool_id"),
					resource.TestCheckResourceAttrSet("aws_cognito_identity_pool_roles_attachment.main", "roles.authenticated"),
				),
			},
			{
				Config: testAccAWSCognitoIdentityPoolRolesAttachmentConfig_basic(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolRolesAttachmentExists("aws_cognito_identity_pool_roles_attachment.main"),
					resource.TestCheckResourceAttrSet("aws_cognito_identity_pool_roles_attachment.main", "identity_pool_id"),
					resource.TestCheckResourceAttrSet("aws_cognito_identity_pool_roles_attachment.main", "roles.authenticated"),
				),
			},
		},
	})
}

func TestAccAWSCognitoIdentityPoolRolesAttachment_roleMappings(t *testing.T) {
	name := fmt.Sprintf("%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoIdentityPoolRolesAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCognitoIdentityPoolRolesAttachmentConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolRolesAttachmentExists("aws_cognito_identity_pool_roles_attachment.main"),
					resource.TestCheckResourceAttrSet("aws_cognito_identity_pool_roles_attachment.main", "identity_pool_id"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool_roles_attachment.main", "role_mappings.#", "0"),
					resource.TestCheckResourceAttrSet("aws_cognito_identity_pool_roles_attachment.main", "roles.authenticated"),
				),
			},
			{
				Config: testAccAWSCognitoIdentityPoolRolesAttachmentConfig_roleMappings(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolRolesAttachmentExists("aws_cognito_identity_pool_roles_attachment.main"),
					resource.TestCheckResourceAttrSet("aws_cognito_identity_pool_roles_attachment.main", "identity_pool_id"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool_roles_attachment.main", "role_mappings.#", "1"),
					resource.TestCheckResourceAttrSet("aws_cognito_identity_pool_roles_attachment.main", "roles.authenticated"),
				),
			},
			{
				Config: testAccAWSCognitoIdentityPoolRolesAttachmentConfig_roleMappingsUpdated(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolRolesAttachmentExists("aws_cognito_identity_pool_roles_attachment.main"),
					resource.TestCheckResourceAttrSet("aws_cognito_identity_pool_roles_attachment.main", "identity_pool_id"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool_roles_attachment.main", "role_mappings.#", "1"),
					resource.TestCheckResourceAttrSet("aws_cognito_identity_pool_roles_attachment.main", "roles.authenticated"),
				),
			},
			{
				Config: testAccAWSCognitoIdentityPoolRolesAttachmentConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSCognitoIdentityPoolRolesAttachmentExists("aws_cognito_identity_pool_roles_attachment.main"),
					resource.TestCheckResourceAttrSet("aws_cognito_identity_pool_roles_attachment.main", "identity_pool_id"),
					resource.TestCheckResourceAttr("aws_cognito_identity_pool_roles_attachment.main", "role_mappings.#", "0"),
					resource.TestCheckResourceAttrSet("aws_cognito_identity_pool_roles_attachment.main", "roles.authenticated"),
				),
			},
		},
	})
}

func TestAccAWSCognitoIdentityPoolRolesAttachment_roleMappingsWithAmbiguousRoleResolutionError(t *testing.T) {
	name := fmt.Sprintf("%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoIdentityPoolRolesAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAWSCognitoIdentityPoolRolesAttachmentConfig_roleMappingsWithAmbiguousRoleResolutionError(name),
				ExpectError: regexp.MustCompile(`Error validating ambiguous role resolution`),
			},
		},
	})
}

func TestAccAWSCognitoIdentityPoolRolesAttachment_roleMappingsWithRulesTypeError(t *testing.T) {
	name := fmt.Sprintf("%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoIdentityPoolRolesAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAWSCognitoIdentityPoolRolesAttachmentConfig_roleMappingsWithRulesTypeError(name),
				ExpectError: regexp.MustCompile(`rules_configuration is required for Rules`),
			},
		},
	})
}

func TestAccAWSCognitoIdentityPoolRolesAttachment_roleMappingsWithTokenTypeError(t *testing.T) {
	name := fmt.Sprintf("%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCognitoIdentityPoolRolesAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAWSCognitoIdentityPoolRolesAttachmentConfig_roleMappingsWithTokenTypeError(name),
				ExpectError: regexp.MustCompile(`rules_configuration must not be set for Token based role mapping`),
			},
		},
	})
}

func testAccCheckAWSCognitoIdentityPoolRolesAttachmentExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("No Cognito Identity Pool Roles Attachment ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).cognitoconn

		_, err := conn.GetIdentityPoolRoles(&cognitoidentity.GetIdentityPoolRolesInput{
			IdentityPoolId: aws.String(rs.Primary.Attributes["identity_pool_id"]),
		})

		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckAWSCognitoIdentityPoolRolesAttachmentDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).cognitoconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_cognito_identity_pool_roles_attachment" {
			continue
		}

		_, err := conn.GetIdentityPoolRoles(&cognitoidentity.GetIdentityPoolRolesInput{
			IdentityPoolId: aws.String(rs.Primary.Attributes["identity_pool_id"]),
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

const baseAWSCognitoIdentityPoolRolesAttachmentConfig = `
resource "aws_cognito_identity_pool" "main" {
  identity_pool_name               = "identity pool %s"
  allow_unauthenticated_identities = false

  supported_login_providers {
    "graph.facebook.com" = "7346241598935555"
  }
}

# Unauthenticated Role
resource "aws_iam_role" "unauthenticated" {
  name = "cognito_unauthenticated_%s"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "cognito-identity.amazonaws.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "cognito-identity.amazonaws.com:aud": "${aws_cognito_identity_pool.main.id}"
        },
        "ForAnyValue:StringLike": {
          "cognito-identity.amazonaws.com:amr": "unauthenticated"
        }
      }
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "unauthenticated" {
  name = "unauthenticated_policy_%s"
  role = "${aws_iam_role.unauthenticated.id}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "mobileanalytics:PutEvents",
        "cognito-sync:*"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}
EOF
}

# Authenticated Role
resource "aws_iam_role" "authenticated" {
  name = "cognito_authenticated_%s"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "cognito-identity.amazonaws.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "cognito-identity.amazonaws.com:aud": "${aws_cognito_identity_pool.main.id}"
        },
        "ForAnyValue:StringLike": {
          "cognito-identity.amazonaws.com:amr": "authenticated"
        }
      }
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "authenticated" {
  name = "authenticated_policy_%s"
  role = "${aws_iam_role.authenticated.id}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "mobileanalytics:PutEvents",
        "cognito-sync:*",
        "cognito-identity:*"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}
EOF
}
`

func testAccAWSCognitoIdentityPoolRolesAttachmentConfig_basic(name string) string {
	return fmt.Sprintf(baseAWSCognitoIdentityPoolRolesAttachmentConfig+`
resource "aws_cognito_identity_pool_roles_attachment" "main" {
  identity_pool_id = "${aws_cognito_identity_pool.main.id}"
  roles {
    "authenticated" = "${aws_iam_role.authenticated.arn}"
  }
}
`, name, name, name, name, name)
}

func testAccAWSCognitoIdentityPoolRolesAttachmentConfig_roleMappings(name string) string {
	return fmt.Sprintf(baseAWSCognitoIdentityPoolRolesAttachmentConfig+`
resource "aws_cognito_identity_pool_roles_attachment" "main" {
  identity_pool_id = "${aws_cognito_identity_pool.main.id}"

  role_mappings {
    role_mapping {
      key   = "graph.facebook.com"
      value = {
        ambiguous_role_resolution = "AuthenticatedRole"
        type                      = "Rules"

        rules_configuration {
          rules {
            claim      = "isAdmin"
            match_type = "Equals"
            role_arn   = "${aws_iam_role.authenticated.arn}"
            value      = "paid"
          }
        }
      }
    }
  }

  roles {
    "authenticated" = "${aws_iam_role.authenticated.arn}"
  }
}
`, name, name, name, name, name)
}

func testAccAWSCognitoIdentityPoolRolesAttachmentConfig_roleMappingsUpdated(name string) string {
	return fmt.Sprintf(baseAWSCognitoIdentityPoolRolesAttachmentConfig+`
resource "aws_cognito_identity_pool_roles_attachment" "main" {
  identity_pool_id = "${aws_cognito_identity_pool.main.id}"

  role_mappings {
    role_mapping {
      key   = "graph.facebook.com"
      value = {
        ambiguous_role_resolution = "AuthenticatedRole"
        type                      = "Rules"

        rules_configuration {
          rules {
            claim      = "isPaid"
            match_type = "Equals"
            role_arn   = "${aws_iam_role.authenticated.arn}"
            value      = "unpaid"
          }
        }
      }
    }
  }

  roles {
    "authenticated" = "${aws_iam_role.authenticated.arn}"
  }
}
`, name, name, name, name, name)
}

func testAccAWSCognitoIdentityPoolRolesAttachmentConfig_roleMappingsWithAmbiguousRoleResolutionError(name string) string {
	return fmt.Sprintf(baseAWSCognitoIdentityPoolRolesAttachmentConfig+`
resource "aws_cognito_identity_pool_roles_attachment" "main" {
  identity_pool_id = "${aws_cognito_identity_pool.main.id}"

  role_mappings {
    role_mapping {
      key   = "graph.facebook.com"
      value = {
        type = "Rules"

        rules_configuration {
          rules {
            claim      = "isAdmin"
            match_type = "Equals"
            role_arn   = "${aws_iam_role.authenticated.arn}"
            value      = "paid"
          }
        }
      }
    }
  }

  roles {
    "authenticated" = "${aws_iam_role.authenticated.arn}"
  }
}
`, name, name, name, name, name)
}

func testAccAWSCognitoIdentityPoolRolesAttachmentConfig_roleMappingsWithRulesTypeError(name string) string {
	return fmt.Sprintf(baseAWSCognitoIdentityPoolRolesAttachmentConfig+`
resource "aws_cognito_identity_pool_roles_attachment" "main" {
  identity_pool_id = "${aws_cognito_identity_pool.main.id}"

  role_mappings {
    role_mapping {
      key   = "graph.facebook.com"
      value = {
        ambiguous_role_resolution = "AuthenticatedRole"
        type = "Rules"
      }
    }
  }

  roles {
    "authenticated" = "${aws_iam_role.authenticated.arn}"
  }
}
`, name, name, name, name, name)
}

func testAccAWSCognitoIdentityPoolRolesAttachmentConfig_roleMappingsWithTokenTypeError(name string) string {
	return fmt.Sprintf(baseAWSCognitoIdentityPoolRolesAttachmentConfig+`
resource "aws_cognito_identity_pool_roles_attachment" "main" {
  identity_pool_id = "${aws_cognito_identity_pool.main.id}"

  role_mappings {
    role_mapping {
      key   = "graph.facebook.com"
      value = {
        ambiguous_role_resolution = "AuthenticatedRole"
        type = "Token"

        rules_configuration {
          rules {
            claim      = "isAdmin"
            match_type = "Equals"
            role_arn   = "${aws_iam_role.authenticated.arn}"
            value      = "paid"
          }
        }
      }
    }
  }

  roles {
    "authenticated" = "${aws_iam_role.authenticated.arn}"
  }
}
`, name, name, name, name, name)
}
