package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDBEventSubscription_basicUpdate(t *testing.T) {
	var v rds.EventSubscription
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBEventSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDBEventSubscriptionConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBEventSubscriptionExists("aws_db_event_subscription.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "source_type", "db-instance"),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "name", fmt.Sprintf("tf-acc-test-rds-event-subs-%d", rInt)),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "tags.Name", "name"),
				),
			},
			{
				Config: testAccAWSDBEventSubscriptionConfigUpdate(rInt),
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

func TestAccAWSDBEventSubscription_withSourceIds(t *testing.T) {
	var v rds.EventSubscription
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBEventSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDBEventSubscriptionConfigWithSourceIds(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBEventSubscriptionExists("aws_db_event_subscription.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "source_type", "db-parameter-group"),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "name", fmt.Sprintf("tf-acc-test-rds-event-subs-with-ids-%d", rInt)),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "source_ids.#", "1"),
				),
			},
			{
				Config: testAccAWSDBEventSubscriptionConfigUpdateSourceIds(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBEventSubscriptionExists("aws_db_event_subscription.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "source_type", "db-parameter-group"),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "name", fmt.Sprintf("tf-acc-test-rds-event-subs-with-ids-%d", rInt)),
					resource.TestCheckResourceAttr(
						"aws_db_event_subscription.bar", "source_ids.#", "2"),
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

func testAccAWSDBEventSubscriptionConfig(rInt int) string {
	return fmt.Sprintf(`
resource "aws_sns_topic" "aws_sns_topic" {
  name = "tf-acc-test-rds-event-subs-sns-topic-%d"
}

resource "aws_db_event_subscription" "bar" {
  name = "tf-acc-test-rds-event-subs-%d"
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
}`, rInt, rInt)
}

func testAccAWSDBEventSubscriptionConfigUpdate(rInt int) string {
	return fmt.Sprintf(`
resource "aws_sns_topic" "aws_sns_topic" {
  name = "tf-acc-test-rds-event-subs-sns-topic-%d"
}

resource "aws_db_event_subscription" "bar" {
  name = "tf-acc-test-rds-event-subs-%d"
  sns_topic = "${aws_sns_topic.aws_sns_topic.arn}"
  enabled = false
  source_type = "db-parameter-group"
  event_categories = [
    "configuration change"
  ]
  tags {
    Name = "new-name"
  }
}`, rInt, rInt)
}

func testAccAWSDBEventSubscriptionConfigWithSourceIds(rInt int) string {
	return fmt.Sprintf(`
resource "aws_sns_topic" "aws_sns_topic" {
  name = "tf-acc-test-rds-event-subs-sns-topic-%d"
}

resource "aws_db_parameter_group" "bar" {
  name = "db-parameter-group-event-%d"
  family = "mysql5.6"
  description = "Test parameter group for terraform"
}

resource "aws_db_event_subscription" "bar" {
  name = "tf-acc-test-rds-event-subs-with-ids-%d"
  sns_topic = "${aws_sns_topic.aws_sns_topic.arn}"
  source_type = "db-parameter-group"
  source_ids = ["${aws_db_parameter_group.bar.id}"]
  event_categories = [
    "configuration change"
  ]
  tags {
    Name = "name"
  }
}`, rInt, rInt, rInt)
}

func testAccAWSDBEventSubscriptionConfigUpdateSourceIds(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_sns_topic" "aws_sns_topic" {
		name = "tf-acc-test-rds-event-subs-sns-topic-%d"
	}

	resource "aws_db_parameter_group" "bar" {
		name = "db-parameter-group-event-%d"
		family = "mysql5.6"
		description = "Test parameter group for terraform"
	}

	resource "aws_db_parameter_group" "foo" {
		name = "db-parameter-group-event-2-%d"
		family = "mysql5.6"
		description = "Test parameter group for terraform"
	}

	resource "aws_db_event_subscription" "bar" {
		name = "tf-acc-test-rds-event-subs-with-ids-%d"
		sns_topic = "${aws_sns_topic.aws_sns_topic.arn}"
		source_type = "db-parameter-group"
		source_ids = ["${aws_db_parameter_group.bar.id}","${aws_db_parameter_group.foo.id}"]
		event_categories = [
			"configuration change"
		]
		tags {
			Name = "name"
		}
	}`, rInt, rInt, rInt, rInt)
}
