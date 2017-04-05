// make testacc TEST=./builtin/providers/aws/ TESTARGS='-run=TestAccAWSKmsKey_'
package aws

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/jen20/awspolicyequivalence"
)

func TestAccAWSKmsKey_basic(t *testing.T) {
	var keyBefore, keyAfter kms.KeyMetadata

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKmsKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSKmsKey,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsKeyExists("aws_kms_key.foo", &keyBefore),
				),
			},
			{
				Config: testAccAWSKmsKey_removedPolicy,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsKeyExists("aws_kms_key.foo", &keyAfter),
				),
			},
		},
	})
}

func TestAccAWSKmsKey_disappears(t *testing.T) {
	var key kms.KeyMetadata

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKmsKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSKmsKey,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsKeyExists("aws_kms_key.foo", &key),
				),
			},
			{
				Config:             testAccAWSKmsKey_other_region,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSKmsKey_policy(t *testing.T) {
	var key kms.KeyMetadata
	expectedPolicyText := `{"Version":"2012-10-17","Id":"kms-tf-1","Statement":[{"Sid":"Enable IAM User Permissions","Effect":"Allow","Principal":{"AWS":"*"},"Action":"kms:*","Resource":"*"}]}`

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKmsKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSKmsKey,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsKeyExists("aws_kms_key.foo", &key),
					testAccCheckAWSKmsKeyHasPolicy("aws_kms_key.foo", expectedPolicyText),
				),
			},
		},
	})
}

func TestAccAWSKmsKey_isEnabled(t *testing.T) {
	var key1, key2, key3 kms.KeyMetadata

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKmsKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSKmsKey_enabledRotation,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsKeyExists("aws_kms_key.bar", &key1),
					resource.TestCheckResourceAttr("aws_kms_key.bar", "is_enabled", "true"),
					testAccCheckAWSKmsKeyIsEnabled(&key1, true),
					resource.TestCheckResourceAttr("aws_kms_key.bar", "enable_key_rotation", "true"),
				),
			},
			{
				Config: testAccAWSKmsKey_disabled,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsKeyExists("aws_kms_key.bar", &key2),
					resource.TestCheckResourceAttr("aws_kms_key.bar", "is_enabled", "false"),
					testAccCheckAWSKmsKeyIsEnabled(&key2, false),
					resource.TestCheckResourceAttr("aws_kms_key.bar", "enable_key_rotation", "false"),
				),
			},
			{
				Config: testAccAWSKmsKey_enabled,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsKeyExists("aws_kms_key.bar", &key3),
					resource.TestCheckResourceAttr("aws_kms_key.bar", "is_enabled", "true"),
					testAccCheckAWSKmsKeyIsEnabled(&key3, true),
					resource.TestCheckResourceAttr("aws_kms_key.bar", "enable_key_rotation", "true"),
				),
			},
		},
	})
}

func TestAccAWSKmsKey_tags(t *testing.T) {
	var keyBefore kms.KeyMetadata

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKmsKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSKmsKey_tags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsKeyExists("aws_kms_key.foo", &keyBefore),
					resource.TestCheckResourceAttr("aws_kms_key.foo", "tags.%", "2"),
				),
			},
		},
	})
}

func testAccCheckAWSKmsKeyHasPolicy(name string, expectedPolicyText string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No KMS Key ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).kmsconn

		out, err := conn.GetKeyPolicy(&kms.GetKeyPolicyInput{
			KeyId:      aws.String(rs.Primary.ID),
			PolicyName: aws.String("default"),
		})
		if err != nil {
			return err
		}

		actualPolicyText := *out.Policy

		equivalent, err := awspolicy.PoliciesAreEquivalent(actualPolicyText, expectedPolicyText)
		if err != nil {
			return fmt.Errorf("Error testing policy equivalence: %s", err)
		}
		if !equivalent {
			return fmt.Errorf("Non-equivalent policy error:\n\nexpected: %s\n\n     got: %s\n",
				expectedPolicyText, actualPolicyText)
		}

		return nil
	}
}

func testAccCheckAWSKmsKeyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).kmsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_kms_key" {
			continue
		}

		out, err := conn.DescribeKey(&kms.DescribeKeyInput{
			KeyId: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if *out.KeyMetadata.KeyState == "PendingDeletion" {
			return nil
		}

		return fmt.Errorf("KMS key still exists:\n%#v", out.KeyMetadata)
	}

	return nil
}

func testAccCheckAWSKmsKeyExists(name string, key *kms.KeyMetadata) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No KMS Key ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).kmsconn

		out, err := conn.DescribeKey(&kms.DescribeKeyInput{
			KeyId: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return err
		}

		*key = *out.KeyMetadata

		return nil
	}
}

func testAccCheckAWSKmsKeyIsEnabled(key *kms.KeyMetadata, isEnabled bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *key.Enabled != isEnabled {
			return fmt.Errorf("Expected key %q to have is_enabled=%t, given %t",
				*key.Arn, isEnabled, *key.Enabled)
		}

		return nil
	}
}

var kmsTimestamp = time.Now().Format(time.RFC1123)
var testAccAWSKmsKey = fmt.Sprintf(`
resource "aws_kms_key" "foo" {
    description = "Terraform acc test %s"
    deletion_window_in_days = 7
    policy = <<POLICY
{
  "Version": "2012-10-17",
  "Id": "kms-tf-1",
  "Statement": [
    {
      "Sid": "Enable IAM User Permissions",
      "Effect": "Allow",
      "Principal": {
        "AWS": "*"
      },
      "Action": "kms:*",
      "Resource": "*"
    }
  ]
}
POLICY
}`, kmsTimestamp)

var testAccAWSKmsKey_other_region = fmt.Sprintf(`
provider "aws" { 
	region = "us-east-1"
}
resource "aws_kms_key" "foo" {
    description = "Terraform acc test %s"
    deletion_window_in_days = 7
    policy = <<POLICY
{
  "Version": "2012-10-17",
  "Id": "kms-tf-1",
  "Statement": [
    {
      "Sid": "Enable IAM User Permissions",
      "Effect": "Allow",
      "Principal": {
        "AWS": "*"
      },
      "Action": "kms:*",
      "Resource": "*"
    }
  ]
}
POLICY
}`, kmsTimestamp)

var testAccAWSKmsKey_removedPolicy = fmt.Sprintf(`
resource "aws_kms_key" "foo" {
    description = "Terraform acc test %s"
    deletion_window_in_days = 7
}`, kmsTimestamp)

var testAccAWSKmsKey_enabledRotation = fmt.Sprintf(`
resource "aws_kms_key" "bar" {
    description = "Terraform acc test is_enabled %s"
    deletion_window_in_days = 7
    enable_key_rotation = true
}`, kmsTimestamp)
var testAccAWSKmsKey_disabled = fmt.Sprintf(`
resource "aws_kms_key" "bar" {
    description = "Terraform acc test is_enabled %s"
    deletion_window_in_days = 7
    enable_key_rotation = false
    is_enabled = false
}`, kmsTimestamp)
var testAccAWSKmsKey_enabled = fmt.Sprintf(`
resource "aws_kms_key" "bar" {
    description = "Terraform acc test is_enabled %s"
    deletion_window_in_days = 7
    enable_key_rotation = true
    is_enabled = true
}`, kmsTimestamp)

var testAccAWSKmsKey_tags = fmt.Sprintf(`
resource "aws_kms_key" "foo" {
    description = "Terraform acc test %s"
	tags {
		Key1 = "Value One"
		Description = "Very interesting"
	}
}`, kmsTimestamp)
