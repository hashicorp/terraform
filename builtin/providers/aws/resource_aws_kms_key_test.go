package aws

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSKmsKey_basic(t *testing.T) {
	var keyBefore, keyAfter kms.KeyMetadata

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKmsKeyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSKmsKey,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsKeyExists("aws_kms_key.foo", &keyBefore),
				),
			},
			resource.TestStep{
				Config: testAccAWSKmsKey_removedPolicy,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsKeyExists("aws_kms_key.foo", &keyAfter),
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
			resource.TestStep{
				Config: testAccAWSKmsKey_enabledRotation,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsKeyExists("aws_kms_key.bar", &key1),
					resource.TestCheckResourceAttr("aws_kms_key.bar", "is_enabled", "true"),
					testAccCheckAWSKmsKeyIsEnabled(&key1, true),
					resource.TestCheckResourceAttr("aws_kms_key.bar", "enable_key_rotation", "true"),
				),
			},
			resource.TestStep{
				Config: testAccAWSKmsKey_disabled,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsKeyExists("aws_kms_key.bar", &key2),
					resource.TestCheckResourceAttr("aws_kms_key.bar", "is_enabled", "false"),
					testAccCheckAWSKmsKeyIsEnabled(&key2, false),
					resource.TestCheckResourceAttr("aws_kms_key.bar", "enable_key_rotation", "false"),
				),
			},
			resource.TestStep{
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
