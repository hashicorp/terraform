package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSQSQueue_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSQSQueueDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSQSConfigWithDefaults,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSQSExistsWithDefaults("aws_sqs_queue.queue-with-defaults"),
				),
			},
			resource.TestStep{
				Config: testAccAWSSQSConfigWithOverrides,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSQSExistsWithOverrides("aws_sqs_queue.queue-with-overrides"),
				),
			},
		},
	})
}

func testAccCheckAWSSQSQueueDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).sqsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_sqs_queue" {
			continue
		}

		// Check if queue exists by checking for its attributes
		params := &sqs.GetQueueAttributesInput{
			QueueUrl: aws.String(rs.Primary.ID),
		}
		_, err := conn.GetQueueAttributes(params)
		if err == nil {
			return fmt.Errorf("Queue %s still exists. Failing!", rs.Primary.ID)
		}

		// Verify the error is what we want
		_, ok := err.(awserr.Error)
		if !ok {
			return err
		}
	}

	return nil
}

func testAccCheckAWSSQSExistsWithDefaults(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Queue URL specified!")
		}

		conn := testAccProvider.Meta().(*AWSClient).sqsconn

		params := &sqs.GetQueueAttributesInput{
			QueueUrl:       aws.String(rs.Primary.ID),
			AttributeNames: []*string{aws.String("All")},
		}
		resp, err := conn.GetQueueAttributes(params)

		if err != nil {
			return err
		}

		// checking if attributes are defaults
		for k, v := range resp.Attributes {
			if k == "VisibilityTimeout" && *v != "30" {
				return fmt.Errorf("VisibilityTimeout (%s) was not set to 30", *v)
			}

			if k == "MessageRetentionPeriod" && *v != "345600" {
				return fmt.Errorf("MessageRetentionPeriod (%s) was not set to 345600", *v)
			}

			if k == "MaximumMessageSize" && *v != "262144" {
				return fmt.Errorf("MaximumMessageSize (%s) was not set to 262144", *v)
			}

			if k == "DelaySeconds" && *v != "0" {
				return fmt.Errorf("DelaySeconds (%s) was not set to 0", *v)
			}

			if k == "ReceiveMessageWaitTimeSeconds" && *v != "0" {
				return fmt.Errorf("ReceiveMessageWaitTimeSeconds (%s) was not set to 0", *v)
			}
		}

		return nil
	}
}

func testAccCheckAWSSQSExistsWithOverrides(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Queue URL specified!")
		}

		conn := testAccProvider.Meta().(*AWSClient).sqsconn

		params := &sqs.GetQueueAttributesInput{
			QueueUrl:       aws.String(rs.Primary.ID),
			AttributeNames: []*string{aws.String("All")},
		}
		resp, err := conn.GetQueueAttributes(params)

		if err != nil {
			return err
		}

		// checking if attributes match our overrides
		for k, v := range resp.Attributes {
			if k == "VisibilityTimeout" && *v != "60" {
				return fmt.Errorf("VisibilityTimeout (%s) was not set to 60", *v)
			}

			if k == "MessageRetentionPeriod" && *v != "86400" {
				return fmt.Errorf("MessageRetentionPeriod (%s) was not set to 86400", *v)
			}

			if k == "MaximumMessageSize" && *v != "2048" {
				return fmt.Errorf("MaximumMessageSize (%s) was not set to 2048", *v)
			}

			if k == "DelaySeconds" && *v != "90" {
				return fmt.Errorf("DelaySeconds (%s) was not set to 90", *v)
			}

			if k == "ReceiveMessageWaitTimeSeconds" && *v != "10" {
				return fmt.Errorf("ReceiveMessageWaitTimeSeconds (%s) was not set to 10", *v)
			}
		}

		return nil
	}
}

const testAccAWSSQSConfigWithDefaults = `
resource "aws_sqs_queue" "queue-with-defaults" {
    name = "test-sqs-queue-with-defaults"
}
`

const testAccAWSSQSConfigWithOverrides = `
resource "aws_sqs_queue" "queue-with-overrides" {
	name = "test-sqs-queue-with-overrides"
	delay_seconds = 90
  	max_message_size = 2048
  	message_retention_seconds = 86400
  	receive_wait_time_seconds = 10
  	visibility_timeout_seconds = 60
}
`
