package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/jen20/awspolicyequivalence"
)

func TestAccAWSSNSTopic_basic(t *testing.T) {
	rName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_sns_topic.test_topic",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSNSTopicDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSNSTopicConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSNSTopicExists("aws_sns_topic.test_topic"),
				),
			},
		},
	})
}

func TestAccAWSSNSTopic_policy(t *testing.T) {
	rName := acctest.RandString(10)
	expectedPolicy := `{"Statement":[{"Sid":"Stmt1445931846145","Effect":"Allow","Principal":{"AWS":"*"},"Action":"sns:Publish","Resource":"arn:aws:sns:us-west-2::example"}],"Version":"2012-10-17","Id":"Policy1445931846145"}`
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_sns_topic.test_topic",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSNSTopicDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSNSTopicWithPolicy(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSNSTopicExists("aws_sns_topic.test_topic"),
					testAccCheckAWSNSTopicHasPolicy("aws_sns_topic.test_topic", expectedPolicy),
				),
			},
		},
	})
}

func TestAccAWSSNSTopic_withIAMRole(t *testing.T) {
	rName := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_sns_topic.test_topic",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSNSTopicDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSNSTopicConfig_withIAMRole(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSNSTopicExists("aws_sns_topic.test_topic"),
				),
			},
		},
	})
}

func TestAccAWSSNSTopic_withDeliveryPolicy(t *testing.T) {
	rName := acctest.RandString(10)
	expectedPolicy := `{"http":{"defaultHealthyRetryPolicy": {"minDelayTarget": 20,"maxDelayTarget": 20,"numMaxDelayRetries": 0,"numRetries": 3,"numNoDelayRetries": 0,"numMinDelayRetries": 0,"backoffFunction": "linear"},"disableSubscriptionOverrides": false}}`
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_sns_topic.test_topic",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSSNSTopicDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSNSTopicConfig_withDeliveryPolicy(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSNSTopicExists("aws_sns_topic.test_topic"),
					testAccCheckAWSNSTopicHasDeliveryPolicy("aws_sns_topic.test_topic", expectedPolicy),
				),
			},
		},
	})
}

func testAccCheckAWSNSTopicHasPolicy(n string, expectedPolicyText string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Queue URL specified!")
		}

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SNS topic with that ARN exists")
		}

		conn := testAccProvider.Meta().(*AWSClient).snsconn

		params := &sns.GetTopicAttributesInput{
			TopicArn: aws.String(rs.Primary.ID),
		}
		resp, err := conn.GetTopicAttributes(params)
		if err != nil {
			return err
		}

		var actualPolicyText string
		for k, v := range resp.Attributes {
			if k == "Policy" {
				actualPolicyText = *v
				break
			}
		}

		equivalent, err := awspolicy.PoliciesAreEquivalent(actualPolicyText, expectedPolicyText)
		if err != nil {
			return fmt.Errorf("Error testing policy equivalence: %s", err)
		}
		if !equivalent {
			return fmt.Errorf("Non-equivalent policy error:\n\nexpected: %s\n\n     got: %s\n",
				expectedPolicyText, actualPolicyText)
		}

		return nil
	}
}

func testAccCheckAWSNSTopicHasDeliveryPolicy(n string, expectedPolicyText string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Queue URL specified!")
		}

		conn := testAccProvider.Meta().(*AWSClient).snsconn

		params := &sns.GetTopicAttributesInput{
			TopicArn: aws.String(rs.Primary.ID),
		}
		resp, err := conn.GetTopicAttributes(params)
		if err != nil {
			return err
		}

		var actualPolicyText string
		for k, v := range resp.Attributes {
			if k == "DeliveryPolicy" {
				actualPolicyText = *v
				break
			}
		}

		equivalent := suppressEquivalentJsonDiffs("", actualPolicyText, expectedPolicyText, nil)

		if !equivalent {
			return fmt.Errorf("Non-equivalent delivery policy error:\n\nexpected: %s\n\n     got: %s\n",
				expectedPolicyText, actualPolicyText)
		}

		return nil
	}
}

func testAccCheckAWSSNSTopicDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).snsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_sns_topic" {
			continue
		}

		// Check if the topic exists by fetching its attributes
		params := &sns.GetTopicAttributesInput{
			TopicArn: aws.String(rs.Primary.ID),
		}
		_, err := conn.GetTopicAttributes(params)
		if err == nil {
			return fmt.Errorf("Topic exists when it should be destroyed!")
		}

		// Verify the error is an API error, not something else
		_, ok := err.(awserr.Error)
		if !ok {
			return err
		}
	}

	return nil
}

func testAccCheckAWSSNSTopicExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SNS topic with that ARN exists")
		}

		conn := testAccProvider.Meta().(*AWSClient).snsconn

		params := &sns.GetTopicAttributesInput{
			TopicArn: aws.String(rs.Primary.ID),
		}
		_, err := conn.GetTopicAttributes(params)

		if err != nil {
			return err
		}

		return nil
	}
}

func testAccAWSSNSTopicConfig(r string) string {
	return fmt.Sprintf(`
resource "aws_sns_topic" "test_topic" {
    name = "terraform-test-topic-%s"
}
`, r)
}

func testAccAWSSNSTopicWithPolicy(r string) string {
	return fmt.Sprintf(`
resource "aws_sns_topic" "test_topic" {
  name = "example-%s"
  policy = <<EOF
{
  "Statement": [
    {
      "Sid": "Stmt1445931846145",
      "Effect": "Allow",
      "Principal": {
        "AWS": "*"
       },
      "Action": "sns:Publish",
      "Resource": "arn:aws:sns:us-west-2::example"
    }
  ],
  "Version": "2012-10-17",
  "Id": "Policy1445931846145"
}
EOF
}
`, r)
}

// Test for https://github.com/hashicorp/terraform/issues/3660
func testAccAWSSNSTopicConfig_withIAMRole(r string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "example" {
  name = "tf_acc_test_%s"
  path = "/test/"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_sns_topic" "test_topic" {
  name = "tf-acc-test-with-iam-role-%s"
  policy = <<EOF
{
  "Statement": [
    {
      "Sid": "Stmt1445931846145",
      "Effect": "Allow",
      "Principal": {
        "AWS": "${aws_iam_role.example.arn}"
			},
      "Action": "sns:Publish",
      "Resource": "arn:aws:sns:us-west-2::example"
    }
  ],
  "Version": "2012-10-17",
  "Id": "Policy1445931846145"
}
EOF
}
`, r, r)
}

// Test for https://github.com/hashicorp/terraform/issues/14024
func testAccAWSSNSTopicConfig_withDeliveryPolicy(r string) string {
	return fmt.Sprintf(`
resource "aws_sns_topic" "test_topic" {
  name = "tf_acc_test_delivery_policy_%s"
  delivery_policy = <<EOF
{
  "http": {
    "defaultHealthyRetryPolicy": {
      "minDelayTarget": 20,
      "maxDelayTarget": 20,
      "numRetries": 3,
      "numMaxDelayRetries": 0,
      "numNoDelayRetries": 0,
      "numMinDelayRetries": 0,
      "backoffFunction": "linear"
    },
    "disableSubscriptionOverrides": false
  }
}
EOF
}
`, r)
}
