package aws

import (
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sns"
)

/**
 Before running this test, a few ENV variables must be set:
 GCM_API_KEY - Google Cloud Messaging API Key
 APNS_SANDBOX_CREDENTIAL - Apple Push Notification Sandbox Private Key
 APNS_SANDBOX_PRINCIPAL - Apple Push Notification Sandbox Certificate
**/

func TestAccAWSSNSApplication_gcm_create_update(t *testing.T) {

	if os.Getenv("GCM_API_KEY") == "" {
		log.Printf("Environment variable GCM_API_KEY not set. Tests cannot run.")
		os.Exit(1)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSNSApplicationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSNSApplicationGCMConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_sns_application.gcm_test", "name", "aws_sns_application_test"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.gcm_test", "platform", "GCM"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.gcm_test", "success_sample_rate", "100"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.gcm_test", "created_topic", "arn:aws:sns:us-east-1:638386993804:endpoint-created-topic"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.gcm_test", "updated_topic", "arn:aws:sns:us-east-1:638386993804:endpoint-updated-topic"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.gcm_test", "failure_topic", "arn:aws:sns:us-east-1:638386993804:endpoint-failure-topic"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.gcm_test", "deleted_topic", "arn:aws:sns:us-east-1:638386993804:endpoint-deleted-topic"),
				),
			},
			resource.TestStep{
				Config: testAccAWSSNSApplicationGCMConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_sns_application.gcm_test", "name", "aws_sns_application_test"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.gcm_test", "platform", "GCM"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.gcm_test", "success_sample_rate", "99"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.gcm_test", "created_topic", "arn:aws:sns:us-east-1:638386993804:endpoint-created-topic-update"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.gcm_test", "updated_topic", "arn:aws:sns:us-east-1:638386993804:endpoint-updated-topic-update"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.gcm_test", "failure_topic", "arn:aws:sns:us-east-1:638386993804:endpoint-failure-topic-update"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.gcm_test", "deleted_topic", "arn:aws:sns:us-east-1:638386993804:endpoint-deleted-topic-update"),
				),
			},
		},
	})
}

func TestAccAWSSNSApplication_apns_sandbox_create_update(t *testing.T) {

	if os.Getenv("APNS_SANDBOX_CREDENTIAL") == "" {
		log.Printf("Environment variable APNS_SANDBOX_CREDENTIAL not set. Tests cannot run.")
		os.Exit(1)
	}

	if os.Getenv("APNS_SANDBOX_PRINCIPAL") == "" {
		log.Printf("Environment variable APNS_SANDBOX_CREDENTIAL not set. Tests cannot run.")
		os.Exit(1)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSNSApplicationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSNSApplicationAPNSSandBoxConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_sns_application.apns_test", "name", "aws_sns_application_test"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.apns_test", "platform", "APNS_SANDBOX"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.apns_test", "success_sample_rate", "100"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.apns_test", "created_topic", "arn:aws:sns:us-east-1:638386993804:endpoint-created-topic"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.apns_test", "updated_topic", "arn:aws:sns:us-east-1:638386993804:endpoint-updated-topic"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.apns_test", "failure_topic", "arn:aws:sns:us-east-1:638386993804:endpoint-failure-topic"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.apns_test", "deleted_topic", "arn:aws:sns:us-east-1:638386993804:endpoint-deleted-topic"),
				),
			},
			resource.TestStep{
				Config: testAccAWSSNSApplicationAPNSSandBoxConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_sns_application.apns_test", "name", "aws_sns_application_test"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.apns_test", "platform", "APNS_SANDBOX"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.apns_test", "success_sample_rate", "99"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.apns_test", "created_topic", "arn:aws:sns:us-east-1:638386993804:endpoint-created-topic-update"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.apns_test", "updated_topic", "arn:aws:sns:us-east-1:638386993804:endpoint-updated-topic-update"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.apns_test", "failure_topic", "arn:aws:sns:us-east-1:638386993804:endpoint-failure-topic-update"),
					resource.TestCheckResourceAttr(
						"aws_sns_application.apns_test", "deleted_topic", "arn:aws:sns:us-east-1:638386993804:endpoint-deleted-topic-update"),
				),
			},
		},
	})
}

func testAccCheckAWSSNSApplicationDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).snsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_sns_application" {
			continue
		}
		_, err := conn.DeletePlatformApplication(&sns.DeletePlatformApplicationInput{
			PlatformApplicationArn: aws.String(rs.Primary.ID),
		})
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoSNSApplication" {
				return nil
			}
			return err
		}
	}
	return nil
}

var testAccAWSSNSApplicationGCMConfig = `
resource "aws_sns_application" "gcm_test" {
	name = "aws_sns_application_test"
	platform = "GCM"
	created_topic = "arn:aws:sns:us-east-1:638386993804:endpoint-created-topic"
	deleted_topic = "arn:aws:sns:us-east-1:638386993804:endpoint-deleted-topic"
	updated_topic = "arn:aws:sns:us-east-1:638386993804:endpoint-updated-topic"
	failure_topic = "arn:aws:sns:us-east-1:638386993804:endpoint-failure-topic"
	success_sample_rate = 100
	credential = "` + os.Getenv("GCM_API_KEY") + `"
}
`

var testAccAWSSNSApplicationGCMConfigUpdate = `
	resource "aws_sns_application" "gcm_test" {
	name = "aws_sns_application_test"
	platform = "GCM"
	created_topic = "arn:aws:sns:us-east-1:638386993804:endpoint-created-topic-update"
	deleted_topic = "arn:aws:sns:us-east-1:638386993804:endpoint-deleted-topic-update"
	updated_topic = "arn:aws:sns:us-east-1:638386993804:endpoint-updated-topic-update"
	failure_topic = "arn:aws:sns:us-east-1:638386993804:endpoint-failure-topic-update"
	success_sample_rate = 99
	credential = "` + os.Getenv("GCM_API_KEY") + `"
}
`

var testAccAWSSNSApplicationAPNSSandBoxConfig = `
resource "aws_sns_application" "apns_test" {
	name = "aws_sns_application_test"
	platform = "APNS_SANDBOX"
	created_topic = "arn:aws:sns:us-east-1:638386993804:endpoint-created-topic"
	deleted_topic = "arn:aws:sns:us-east-1:638386993804:endpoint-deleted-topic"
	updated_topic = "arn:aws:sns:us-east-1:638386993804:endpoint-updated-topic"
	failure_topic = "arn:aws:sns:us-east-1:638386993804:endpoint-failure-topic"
	success_sample_rate = 100
	credential = "` + os.Getenv("APNS_SANDBOX_CREDENTIAL") + `"
	principal = "` + os.Getenv("APNS_SANDBOX_PRINCIPAL") + `"
}
`

var testAccAWSSNSApplicationAPNSSandBoxConfigUpdate = `
	resource "aws_sns_application" "apns_test" {
	name = "aws_sns_application_test"
	platform = "APNS_SANDBOX"
	created_topic = "arn:aws:sns:us-east-1:638386993804:endpoint-created-topic-update"
	deleted_topic = "arn:aws:sns:us-east-1:638386993804:endpoint-deleted-topic-update"
	updated_topic = "arn:aws:sns:us-east-1:638386993804:endpoint-updated-topic-update"
	failure_topic = "arn:aws:sns:us-east-1:638386993804:endpoint-failure-topic-update"
	success_sample_rate = 99
	credential = "` + os.Getenv("APNS_SANDBOX_CREDENTIAL") + `"
	principal = "` + os.Getenv("APNS_SANDBOX_PRINCIPAL") + `"
}
`
