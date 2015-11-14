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
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKmsKeyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSKmsKey,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsKeyExists("aws_kms_key.foo"),
				),
			},
			resource.TestStep{
				Config: testAccAWSKmsKey_removedPolicy,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsKeyExists("aws_kms_key.foo"),
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

		if err == nil {
			return fmt.Errorf("KMS key still exists:\n%#v", out.KeyMetadata)
		}

		return err
	}

	return nil
}

func testAccCheckAWSKmsKeyExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
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
