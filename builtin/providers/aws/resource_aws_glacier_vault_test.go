package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/glacier"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSGlacierVault_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGlacierVaultDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGlacierVault_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGlacierVaultExists("aws_glacier_vault.test"),
				),
			},
		},
	})
}

func TestAccAWSGlacierVault_full(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGlacierVaultDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGlacierVault_full,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGlacierVaultExists("aws_glacier_vault.full"),
				),
			},
		},
	})
}

func TestAccAWSGlacierVault_RemoveNotifications(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGlacierVaultDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGlacierVault_full,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGlacierVaultExists("aws_glacier_vault.full"),
				),
			},
			resource.TestStep{
				Config: testAccGlacierVault_withoutNotification,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGlacierVaultExists("aws_glacier_vault.full"),
					testAccCheckVaultNotificationsMissing("aws_glacier_vault.full"),
				),
			},
		},
	})
}

func testAccCheckGlacierVaultExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		glacierconn := testAccProvider.Meta().(*AWSClient).glacierconn
		out, err := glacierconn.DescribeVault(&glacier.DescribeVaultInput{
			VaultName: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if out.VaultARN == nil {
			return fmt.Errorf("No Glacier Vault Found")
		}

		if *out.VaultName != rs.Primary.ID {
			return fmt.Errorf("Glacier Vault Mismatch - existing: %q, state: %q",
				*out.VaultName, rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckVaultNotificationsMissing(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		glacierconn := testAccProvider.Meta().(*AWSClient).glacierconn
		out, err := glacierconn.GetVaultNotifications(&glacier.GetVaultNotificationsInput{
			VaultName: aws.String(rs.Primary.ID),
		})

		if awserr, ok := err.(awserr.Error); ok && awserr.Code() != "ResourceNotFoundException" {
			return fmt.Errorf("Expected ResourceNotFoundException for Vault %s Notification Block but got %s", rs.Primary.ID, awserr.Code())
		}

		if out.VaultNotificationConfig != nil {
			return fmt.Errorf("Vault Notification Block has been found for %s", rs.Primary.ID)
		}

		return nil
	}

}

func testAccCheckGlacierVaultDestroy(s *terraform.State) error {
	if len(s.RootModule().Resources) > 0 {
		return fmt.Errorf("Expected all resources to be gone, but found: %#v",
			s.RootModule().Resources)
	}

	return nil
}

const testAccGlacierVault_basic = `
resource "aws_glacier_vault" "test" {
  name = "my_test_vault"
}
`

const testAccGlacierVault_full = `
resource "aws_sns_topic" "aws_sns_topic" {
  name = "glacier-sns-topic"
}

resource "aws_glacier_vault" "full" {
  name = "my_test_vault"
  notification {
  	sns_topic = "${aws_sns_topic.aws_sns_topic.arn}"
  	events = ["ArchiveRetrievalCompleted","InventoryRetrievalCompleted"]
  }
  tags {
    Test="Test1"
  }
}
`

const testAccGlacierVault_withoutNotification = `
resource "aws_sns_topic" "aws_sns_topic" {
  name = "glacier-sns-topic"
}

resource "aws_glacier_vault" "full" {
  name = "my_test_vault"
  tags {
    Test="Test1"
  }
}
`
