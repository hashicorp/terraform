package aws

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSpotDatafeedSubscription(t *testing.T) {
	cases := map[string]func(t *testing.T){
		"basic":      testAccAWSSpotDatafeedSubscription_basic,
		"disappears": testAccAWSSpotDatafeedSubscription_disappears,
		"import":     testAccAWSSpotDatafeedSubscription_importBasic,
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc(t)
		})
	}
}

func testAccAWSSpotDatafeedSubscription_basic(t *testing.T) {
	var subscription ec2.SpotDatafeedSubscription
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotDatafeedSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSpotDatafeedSubscription(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSpotDatafeedSubscriptionExists("aws_spot_datafeed_subscription.default", &subscription),
				),
			},
		},
	})
}

func testAccCheckAWSSpotDatafeedSubscriptionDisappears(subscription *ec2.SpotDatafeedSubscription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		_, err := conn.DeleteSpotDatafeedSubscription(&ec2.DeleteSpotDatafeedSubscriptionInput{})
		if err != nil {
			return err
		}

		return resource.Retry(40*time.Minute, func() *resource.RetryError {
			_, err := conn.DescribeSpotDatafeedSubscription(&ec2.DescribeSpotDatafeedSubscriptionInput{})
			if err != nil {
				cgw, ok := err.(awserr.Error)
				if ok && cgw.Code() == "InvalidSpotDatafeed.NotFound" {
					return nil
				}
				return resource.NonRetryableError(
					fmt.Errorf("Error retrieving Spot Datafeed Subscription: %s", err))
			}
			return resource.RetryableError(fmt.Errorf("Waiting for Spot Datafeed Subscription"))
		})
	}
}

func testAccAWSSpotDatafeedSubscription_disappears(t *testing.T) {
	var subscription ec2.SpotDatafeedSubscription
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotDatafeedSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSpotDatafeedSubscription(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSpotDatafeedSubscriptionExists("aws_spot_datafeed_subscription.default", &subscription),
					testAccCheckAWSSpotDatafeedSubscriptionDisappears(&subscription),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAWSSpotDatafeedSubscriptionExists(n string, subscription *ec2.SpotDatafeedSubscription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No policy ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		resp, err := conn.DescribeSpotDatafeedSubscription(&ec2.DescribeSpotDatafeedSubscriptionInput{})
		if err != nil {
			return err
		}

		*subscription = *resp.SpotDatafeedSubscription

		return nil
	}
}

func testAccCheckAWSSpotDatafeedSubscriptionDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_spot_datafeed_subscription" {
			continue
		}

		// Try to get subscription
		_, err := conn.DescribeSpotDatafeedSubscription(&ec2.DescribeSpotDatafeedSubscriptionInput{})
		if err == nil {
			return fmt.Errorf("still exist.")
		}

		awsErr, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if awsErr.Code() != "InvalidSpotDatafeed.NotFound" {
			return err
		}
	}

	return nil
}

func testAccAWSSpotDatafeedSubscription(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "default" {
	bucket = "tf-spot-datafeed-%d"
}

resource "aws_spot_datafeed_subscription" "default" {
	bucket = "${aws_s3_bucket.default.bucket}"
}
`, randInt)
}
