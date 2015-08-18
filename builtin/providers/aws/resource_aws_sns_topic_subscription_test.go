package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSNSTopicSubscription_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSNSTopicSubscriptionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSNSTopicSubscriptionConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSNSTopicExists("aws_sns_topic.test_topic"),
					testAccCheckAWSSNSTopicSubscriptionExists("aws_sns_topic_subscription.test_subscription"),
				),
			},
		},
	})
}

func testAccCheckAWSSNSTopicSubscriptionDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).snsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_sns_topic" {
			continue
		}

		// Try to find key pair
		req := &sns.GetSubscriptionAttributesInput{
			SubscriptionArn: aws.String(rs.Primary.ID),
		}

		_, err := conn.GetSubscriptionAttributes(req)

		if err == nil {
			return fmt.Errorf("Subscription still exists, can't continue.")
		}

		// Verify the error is an API error, not something else
		_, ok := err.(awserr.Error)
		if !ok {
			return err
		}
	}

	return nil
}

func testAccCheckAWSSNSTopicSubscriptionExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SNS subscription with that ARN exists")
		}

		conn := testAccProvider.Meta().(*AWSClient).snsconn

		params := &sns.GetSubscriptionAttributesInput{
			SubscriptionArn: aws.String(rs.Primary.ID),
		}
		_, err := conn.GetSubscriptionAttributes(params)

		if err != nil {
			return err
		}

		return nil
	}
}

const testAccAWSSNSTopicSubscriptionConfig = `
resource "aws_sns_topic" "test_topic" {
    name = "terraform-test-topic"
}

resource "aws_sqs_queue" "test_queue" {
	name = "terraform-subscription-test-queue"
}

resource "aws_sns_topic_subscription" "test_subscription" {
    topic_arn = "${aws_sns_topic.test_topic.arn}"
    protocol = "sqs"
    endpoint = "${aws_sqs_queue.test_queue.arn}"
}
`
