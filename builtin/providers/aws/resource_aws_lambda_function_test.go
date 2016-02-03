package aws

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSLambdaFunction_basic(t *testing.T) {
	var conf lambda.GetFunctionOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSLambdaConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", "example_lambda_name", &conf),
					testAccCheckAWSLambdaAttributes("example_lambda_name", &conf),
				),
			},
			resource.TestStep{
				Config: testAccAWSLambdaConfigS3,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_s3test", "example_lambda_name_s3", &conf),
					testAccCheckAWSLambdaAttributes("example_lambda_name_s3", &conf),
				),
			},
		},
	})
}

func testAccCheckLambdaFunctionDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).lambdaconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_lambda_function" {
			continue
		}

		_, err := conn.GetFunction(&lambda.GetFunctionInput{
			FunctionName: aws.String(rs.Primary.ID),
		})

		if err == nil {
			return fmt.Errorf("Lambda Function still exists")
		}

	}

	return nil

}

func testAccCheckAwsLambdaFunctionExists(res, fname string, function *lambda.GetFunctionOutput) resource.TestCheckFunc {
	// Wait for IAM role
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[res]
		if !ok {
			return fmt.Errorf("Lambda function not found: %s", res)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Lambda function ID not set")
		}

		conn := testAccProvider.Meta().(*AWSClient).lambdaconn

		params := &lambda.GetFunctionInput{
			FunctionName: aws.String(fname),
		}

		getFunction, err := conn.GetFunction(params)
		if err != nil {
			return err
		}

		*function = *getFunction

		return nil
	}
}

func testAccCheckAWSLambdaAttributes(fname string, function *lambda.GetFunctionOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := function.Configuration
		if *c.FunctionName != fname {
			return fmt.Errorf("Expected function name %s, got %s", fname, *c.FunctionName)
		}

		if *c.FunctionArn == "" {
			return fmt.Errorf("Could not read Lambda Function's ARN")
		}

		if strings.HasSuffix(fname, "s3") && *c.Version != "$LATEST" {
			return fmt.Errorf("Expected version $LATEST, got %s", *c.Version)
		}

		return nil
	}
}

const testAccAWSLambdaConfig = `
resource "aws_iam_role" "iam_for_lambda" {
    name = "iam_for_lambda"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "example_lambda_name"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
}
`

var testAccAWSLambdaConfigS3 = fmt.Sprintf(`
resource "aws_s3_bucket" "lambda_bucket" {
  bucket = "tf-test-bucket-%d"
}

resource "aws_s3_bucket_object" "lambda_code" {
  bucket = "${aws_s3_bucket.lambda_bucket.id}"
  key = "lambdatest.zip"
  source = "test-fixtures/lambdatest.zip"
}

resource "aws_iam_role" "iam_for_lambda" {
    name = "iam_for_lambda"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_lambda_function" "lambda_function_s3test" {
    s3_bucket = "${aws_s3_bucket.lambda_bucket.id}"
    s3_key = "${aws_s3_bucket_object.lambda_code.id}"
    function_name = "example_lambda_name_s3"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
}
`, rand.New(rand.NewSource(time.Now().UnixNano())).Int())
