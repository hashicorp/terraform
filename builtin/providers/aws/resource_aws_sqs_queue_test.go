package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/hashicorp/terraform/helper/acctest"
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

func TestAccAWSSQSQueue_redrivePolicy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSQSQueueDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSQSConfigWithRedrive(acctest.RandStringFromCharSet(5, acctest.CharSetAlpha)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSQSExistsWithDefaults("aws_sqs_queue.my_dead_letter_queue"),
				),
			},
		},
	})
}

// Tests formatting and compacting of Policy, Redrive json
func TestAccAWSSQSQueue_Policybasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSQSQueueDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSQSConfig_PolicyFormat,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSQSExistsWithOverrides("aws_sqs_queue.test-email-events"),
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
  name                       = "test-sqs-queue-with-overrides"
  delay_seconds              = 90
  max_message_size           = 2048
  message_retention_seconds  = 86400
  receive_wait_time_seconds  = 10
  visibility_timeout_seconds = 60
}
`

func testAccAWSSQSConfigWithRedrive(name string) string {
	return fmt.Sprintf(`
resource "aws_sqs_queue" "my_queue" {
  name                       = "tftestqueuq-%s"
  delay_seconds              = 0
  visibility_timeout_seconds = 300

  redrive_policy = <<EOF
{
    "maxReceiveCount": 3,
    "deadLetterTargetArn": "${aws_sqs_queue.my_dead_letter_queue.arn}"
}
EOF
}

resource "aws_sqs_queue" "my_dead_letter_queue" {
  name = "tfotherqueuq-%s"
}
`, name, name)
}

const testAccAWSSQSConfig_PolicyFormat = `
variable "sns_name" {
  default = "tf-test-name-2"
}

variable "sqs_name" {
  default = "tf-test-sqs-name-2"
}

resource "aws_sns_topic" "test_topic" {
  name = "${var.sns_name}"
}

resource "aws_sqs_queue" "test-email-events" {
  name                       = "${var.sqs_name}"
  depends_on                 = ["aws_sns_topic.test_topic"]
  delay_seconds              = 90
  max_message_size           = 2048
  message_retention_seconds  = 86400
  receive_wait_time_seconds  = 10
  visibility_timeout_seconds = 60

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Id": "sqspolicy",
  "Statement": [
    {
      "Sid": "Stmt1451501026839",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "sqs:SendMessage",
      "Resource": "arn:aws:sqs:us-west-2:470663696735:${var.sqs_name}",
      "Condition": {
        "ArnEquals": {
          "aws:SourceArn": "arn:aws:sns:us-west-2:470663696735:${var.sns_name}"
        }
      }
    }
  ]
}
EOF
}

resource "aws_sns_topic_subscription" "test_queue_target" {
  topic_arn = "${aws_sns_topic.test_topic.arn}"
  protocol  = "sqs"
  endpoint  = "${aws_sqs_queue.test-email-events.arn}"
}
`
