package aws

import (
	"fmt"
	"testing"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/sqs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSQS_normal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSQSDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSQSConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSQSExists("aws_sqs.sqs-name"),
				),
			},
		},
	})
}


func testAccCheckAWSSQSDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).sqsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_sqs" {
			continue
		}

		// Try to find key pair
		params := &sqs.GetQueueAttributesInput{
		    QueueURL: aws.String(rs.Primary.ID), 
		}
		_, err := conn.GetQueueAttributes(params)
		if err == nil {
			return fmt.Errorf("still exist.")
		}

		// Verify the error is what we want
		_, ok := err.(aws.APIError)
		if !ok {
			return err
		}
	}

	return nil
}


func testAccCheckAWSSQSExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SQS Queue with that name is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).sqsconn

		params := &sqs.GetQueueAttributesInput{
		    QueueURL: aws.String(rs.Primary.ID), 
		}
		resp, err := conn.GetQueueAttributes(params)

		if err != nil {
			return err
		}

		// checking if attributes are defaults
		for k, v := range *resp.Attributes {
			if k == "VisibilityTimeout" && *v != "0" {
				return fmt.Errorf("VisibilityTimeout was not set to 0")
			}

			if k == "MessageRetentionPeriod" && *v != "345600" {
				return fmt.Errorf("MessageRetentionPeriod was not set to 345600")
			}

			if k == "MaximumMessageSize" && *v != "262144" {
				return fmt.Errorf("MaximumMessageSize was not set to 262144")
			}

			if k == "DelaySeconds" && *v != "0" {
				return fmt.Errorf("DelaySeconds was not set to 0")
			}

			if k == "ReceiveMessageWaitTimeSeconds" && *v != "0" {
				return fmt.Errorf("ReceiveMessageWaitTimeSeconds was not set to 0")
			}
		}

		return nil
	}
}

const testAccAWSSQSConfig = `
resource "aws_sqs" "test" {
    name = "sqs-name"
}

`

