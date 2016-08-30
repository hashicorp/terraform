package aws

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSNSPlatformApplicationGCM(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAWSSNSPlatformApplcationGCMDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSNSPlaftformApplcationGCM,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSNSPlatformApplicationExists("aws_sns_platform_application_gcm.platform_application_android"),
				),
			},
		},
	})
}

func TestAccAWSSNSPlatformApplicationAPNS(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAWSSNSPlatformApplcationAPNSDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSNSPlaftformApplcationAPNS,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSNSPlatformApplicationExists("aws_sns_platform_application_gcm.platform_application_ios"),
				),
			},
		},
	})
}

func testAccAWSSNSPlatformApplcationAPNSDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).snsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_sns_platform_application_apns" {
			continue
		}

		params := &sns.GetPlatformApplicationAttributesInput{
			PlatformApplicationArn: aws.String(rs.Primary.ID),
		}

		//App deletes are not consistent so retry
		resource.Retry(30*time.Second, func() *resource.RetryError {
			r, err := conn.GetPlatformApplicationAttributes(params)
			if err == nil {
				return &resource.RetryError{
					Err:       fmt.Errorf("Platform application exists when it should be destroyed %s", r),
					Retryable: true,
				}
			}

			return nil
		})
	}

	return nil
}

func testAccAWSSNSPlatformApplcationGCMDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).snsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_sns_platform_application_gcm" {
			continue
		}

		params := &sns.GetPlatformApplicationAttributesInput{
			PlatformApplicationArn: aws.String(rs.Primary.ID),
		}

		//App deletes are not consistent so retry
		resource.Retry(30*time.Second, func() *resource.RetryError {
			r, err := conn.GetPlatformApplicationAttributes(params)
			if err == nil {
				return &resource.RetryError{
					Err:       fmt.Errorf("Platform application exists when it should be destroyed %s", r),
					Retryable: true,
				}
			}

			return nil
		})
	}

	return nil
}

//AWS does a fair amount of validation on platform credentials, so a simple test should do
func testAccCheckAWSSNSPlatformApplicationExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Platform application with that ARN exists")
		}

		conn := testAccProvider.Meta().(*AWSClient).snsconn

		params := &sns.GetPlatformApplicationAttributesInput{
			PlatformApplicationArn: aws.String(rs.Primary.ID),
		}
		_, err := conn.GetPlatformApplicationAttributes(params)

		if err != nil {
			return err
		}

		return nil
	}
}

const testAccAWSSNSPlaftformApplcationGCM = `
resource "aws_sns_platform_application_gcm" "platform_application_android" {
  name = "terraform_acc_plaform_application_android"
  platform_credential = "CREDS HERE"
}
`

const testAccAWSSNSPlaftformApplcationAPNS = `
resource "aws_sns_platform_application_apns" "platform_application_ios" {
  name = "terraform_acc_plaform_application_ios"
  type = "SANDBOX"
  platform_credential = "CREDS HERE"
  platform_principal = "CREDS HERE"
}
`
