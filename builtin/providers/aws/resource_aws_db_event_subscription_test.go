package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDBEventSubscription_basicUpdate(t *testing.T) {
	var v rds.EventSubscription

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBEventSubscriptionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDBEventSubscriptionConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBEventSubscriptionExists("aws_db_event_subscription.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "source_type", "db-instance"),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "name", "tf-acc-test-rds-event-subs"),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "tags.Name", "name"),
				),
			},
			resource.TestStep{
				Config: testAccAWSDBEventSubscriptionConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBEventSubscriptionExists("aws_db_event_subscription.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "enabled", "false"),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "source_type", "db-parameter-group"),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "tags.Name", "new-name"),
				),
			},
		},
	})
}

func testAccCheckAWSDBEventSubscriptionExists(n string, v *rds.EventSubscription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No RDS Event Subscription is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).rdsconn

		opts := rds.DescribeEventSubscriptionsInput{
			SubscriptionName: aws.String(rs.Primary.ID),
		}

		resp, err := conn.DescribeEventSubscriptions(&opts)

		if err != nil {
			return err
		}

		if len(resp.EventSubscriptionsList) != 1 ||
			*resp.EventSubscriptionsList[0].CustSubscriptionId != rs.Primary.ID {
			return fmt.Errorf("RDS Event Subscription not found")
		}

		*v = *resp.EventSubscriptionsList[0]
		return nil
	}
}

func testAccCheckAWSDBEventSubscriptionDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).rdsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_db_event_subscription" {
			continue
		}

		var err error
		resp, err := conn.DescribeEventSubscriptions(
			&rds.DescribeEventSubscriptionsInput{
				SubscriptionName: aws.String(rs.Primary.ID),
			})

		if ae, ok := err.(awserr.Error); ok && ae.Code() == "SubscriptionNotFound" {
			continue
		}

		if err == nil {
			if len(resp.EventSubscriptionsList) != 0 &&
				*resp.EventSubscriptionsList[0].CustSubscriptionId == rs.Primary.ID {
				return fmt.Errorf("Event Subscription still exists")
			}
		}

		// Verify the error
		newerr, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if newerr.Code() != "SubscriptionNotFound" {
			return err
		}
	}

	return nil
}

var testAccAWSDBEventSubscriptionConfig = `
resource "aws_sns_topic" "aws_sns_topic" {
  name = "tf-acc-test-rds-event-subs-sns-topic"
}

resource "aws_db_event_subscription" "bar" {
  name = "tf-acc-test-rds-event-subs"
  sns_topic = "${aws_sns_topic.aws_sns_topic.arn}"
  source_type = "db-instance"
  event_categories = [
    "availability",
    "backup",
    "creation",
    "deletion",
    "maintenance"
  ]
  tags {
    Name = "name"
  }
}
`

var testAccAWSDBEventSubscriptionConfigUpdate = `
resource "aws_sns_topic" "aws_sns_topic" {
  name = "tf-acc-test-rds-event-subs-sns-topic"
}

resource "aws_db_event_subscription" "bar" {
  name = "tf-acc-test-rds-event-subs"
  sns_topic = "${aws_sns_topic.aws_sns_topic.arn}"
  enabled = false
  source_type = "db-parameter-group"
  event_categories = [
    "configuration change"
  ]
  tags {
    Name = "new-name"
  }
}
`
