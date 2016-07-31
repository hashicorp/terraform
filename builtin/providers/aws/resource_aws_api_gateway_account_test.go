package aws

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAPIGatewayAccount_basic(t *testing.T) {
	var conf apigateway.Account

	expectedRoleArn_first := regexp.MustCompile("[0-9]+")
	expectedRoleArn_second := regexp.MustCompile("[0-9]+")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayAccountDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAPIGatewayAccountConfig_updated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayAccountExists("aws_api_gateway_account.test", &conf),
					testAccCheckAWSAPIGatewayAccountCloudwatchRoleArn(&conf, expectedRoleArn_first),
					resource.TestMatchResourceAttr("aws_api_gateway_account.test", "cloudwatch_role_arn", expectedRoleArn_first),
				),
			},
			resource.TestStep{
				Config: testAccAWSAPIGatewayAccountConfig_updated2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayAccountExists("aws_api_gateway_account.test", &conf),
					testAccCheckAWSAPIGatewayAccountCloudwatchRoleArn(&conf, expectedRoleArn_second),
					resource.TestMatchResourceAttr("aws_api_gateway_account.test", "cloudwatch_role_arn", expectedRoleArn_second),
				),
			},
			resource.TestStep{
				Config: testAccAWSAPIGatewayAccountConfig_empty,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayAccountExists("aws_api_gateway_account.test", &conf),
					testAccCheckAWSAPIGatewayAccountCloudwatchRoleArn(&conf, expectedRoleArn_second),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewayAccountCloudwatchRoleArn(conf *apigateway.Account, expectedArn *regexp.Regexp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if expectedArn == nil && conf.CloudwatchRoleArn == nil {
			return nil
		}
		if expectedArn == nil && conf.CloudwatchRoleArn != nil {
			return fmt.Errorf("Expected empty CloudwatchRoleArn, given: %q", *conf.CloudwatchRoleArn)
		}
		if expectedArn != nil && conf.CloudwatchRoleArn == nil {
			return fmt.Errorf("Empty CloudwatchRoleArn, expected: %q", expectedArn)
		}
		if !expectedArn.MatchString(*conf.CloudwatchRoleArn) {
			return fmt.Errorf("CloudwatchRoleArn didn't match. Expected: %q, Given: %q", expectedArn, *conf.CloudwatchRoleArn)
		}
		return nil
	}
}

func testAccCheckAWSAPIGatewayAccountExists(n string, res *apigateway.Account) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway Account ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigateway

		req := &apigateway.GetAccountInput{}
		describe, err := conn.GetAccount(req)
		if err != nil {
			return err
		}
		if describe == nil {
			return fmt.Errorf("Got nil account ?!")
		}

		*res = *describe

		return nil
	}
}

func testAccCheckAWSAPIGatewayAccountDestroy(s *terraform.State) error {
	// Intentionally noop
	// as there is no API method for deleting or resetting account settings
	return nil
}

const testAccAWSAPIGatewayAccountConfig_empty = `
resource "aws_api_gateway_account" "test" {
}
`

const testAccAWSAPIGatewayAccountConfig_updated = `
resource "aws_api_gateway_account" "test" {
  cloudwatch_role_arn = "${aws_iam_role.cloudwatch.arn}"
}

resource "aws_iam_role" "cloudwatch" {
    name = "api_gateway_cloudwatch_global"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "apigateway.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "cloudwatch" {
    name = "default"
    role = "${aws_iam_role.cloudwatch.id}"
    policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:DescribeLogGroups",
                "logs:DescribeLogStreams",
                "logs:PutLogEvents",
                "logs:GetLogEvents",
                "logs:FilterLogEvents"
            ],
            "Resource": "*"
        }
    ]
}
EOF
}
`
const testAccAWSAPIGatewayAccountConfig_updated2 = `
resource "aws_api_gateway_account" "test" {
  cloudwatch_role_arn = "${aws_iam_role.second.arn}"
}

resource "aws_iam_role" "second" {
    name = "api_gateway_cloudwatch_global_modified"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "apigateway.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "cloudwatch" {
    name = "default"
    role = "${aws_iam_role.second.id}"
    policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:DescribeLogGroups",
                "logs:DescribeLogStreams",
                "logs:PutLogEvents",
                "logs:GetLogEvents",
                "logs:FilterLogEvents"
            ],
            "Resource": "*"
        }
    ]
}
EOF
}
`
