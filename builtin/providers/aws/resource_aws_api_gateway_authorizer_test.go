package aws

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAPIGatewayAuthorizer_basic(t *testing.T) {
	var conf apigateway.Authorizer

	expectedAuthUri := regexp.MustCompile("arn:aws:apigateway:region:lambda:path/2015-03-31/functions/" +
		"arn:aws:lambda:[a-z0-9-]+:[0-9]{12}:function:tf_acc_api_gateway_authorizer/invocations")
	expectedCreds := regexp.MustCompile("arn:aws:iam::[0-9]{12}:role/tf_acc_api_gateway_auth_invocation_role")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayAuthorizerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAPIGatewayAuthorizerConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayAuthorizerExists("aws_api_gateway_authorizer.test", &conf),
					testAccCheckAWSAPIGatewayAuthorizerAuthorizerUri(&conf, expectedAuthUri),
					resource.TestMatchResourceAttr("aws_api_gateway_authorizer.test", "authorizer_uri", expectedAuthUri),
					testAccCheckAWSAPIGatewayAuthorizerIdentitySource(&conf, "method.request.header.Authorization"),
					resource.TestCheckResourceAttr("aws_api_gateway_authorizer.test", "identity_source", "method.request.header.Authorization"),
					testAccCheckAWSAPIGatewayAuthorizerName(&conf, "tf-acc-test-authorizer"),
					resource.TestCheckResourceAttr("aws_api_gateway_authorizer.test", "name", "tf-acc-test-authorizer"),
					testAccCheckAWSAPIGatewayAuthorizerType(&conf, "TOKEN"),
					resource.TestCheckResourceAttr("aws_api_gateway_authorizer.test", "type", "TOKEN"),
					testAccCheckAWSAPIGatewayAuthorizerAuthorizerCredentials(&conf, expectedCreds),
					resource.TestMatchResourceAttr("aws_api_gateway_authorizer.test", "authorizer_credentials", expectedCreds),
					testAccCheckAWSAPIGatewayAuthorizerAuthorizerResultTtlInSeconds(&conf, nil),
					resource.TestCheckResourceAttr("aws_api_gateway_authorizer.test", "authorizer_result_ttl_in_seconds", "0"),
					testAccCheckAWSAPIGatewayAuthorizerIdentityValidationExpression(&conf, nil),
					resource.TestCheckResourceAttr("aws_api_gateway_authorizer.test", "identity_validation_expression", ""),
				),
			},
			resource.TestStep{
				Config: testAccAWSAPIGatewayAuthorizerUpdatedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayAuthorizerExists("aws_api_gateway_authorizer.test", &conf),
					testAccCheckAWSAPIGatewayAuthorizerAuthorizerUri(&conf, expectedAuthUri),
					resource.TestMatchResourceAttr("aws_api_gateway_authorizer.test", "authorizer_uri", expectedAuthUri),
					testAccCheckAWSAPIGatewayAuthorizerIdentitySource(&conf, "method.request.header.Authorization"),
					resource.TestCheckResourceAttr("aws_api_gateway_authorizer.test", "identity_source", "method.request.header.Authorization"),
					testAccCheckAWSAPIGatewayAuthorizerName(&conf, "tf-acc-test-authorizer_modified"),
					resource.TestCheckResourceAttr("aws_api_gateway_authorizer.test", "name", "tf-acc-test-authorizer_modified"),
					testAccCheckAWSAPIGatewayAuthorizerType(&conf, "TOKEN"),
					resource.TestCheckResourceAttr("aws_api_gateway_authorizer.test", "type", "TOKEN"),
					testAccCheckAWSAPIGatewayAuthorizerAuthorizerCredentials(&conf, expectedCreds),
					resource.TestMatchResourceAttr("aws_api_gateway_authorizer.test", "authorizer_credentials", expectedCreds),
					testAccCheckAWSAPIGatewayAuthorizerAuthorizerResultTtlInSeconds(&conf, aws.Int64(360)),
					resource.TestCheckResourceAttr("aws_api_gateway_authorizer.test", "authorizer_result_ttl_in_seconds", "360"),
					testAccCheckAWSAPIGatewayAuthorizerIdentityValidationExpression(&conf, aws.String(".*")),
					resource.TestCheckResourceAttr("aws_api_gateway_authorizer.test", "identity_validation_expression", ".*"),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewayAuthorizerAuthorizerUri(conf *apigateway.Authorizer, expectedUri *regexp.Regexp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if conf.AuthorizerUri == nil {
			return fmt.Errorf("Empty AuthorizerUri, expected: %q", expectedUri)
		}

		if !expectedUri.MatchString(*conf.AuthorizerUri) {
			return fmt.Errorf("AuthorizerUri didn't match. Expected: %q, Given: %q", expectedUri, *conf.AuthorizerUri)
		}
		return nil
	}
}

func testAccCheckAWSAPIGatewayAuthorizerIdentitySource(conf *apigateway.Authorizer, expectedSource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if conf.IdentitySource == nil {
			return fmt.Errorf("Empty IdentitySource, expected: %q", expectedSource)
		}
		if *conf.IdentitySource != expectedSource {
			return fmt.Errorf("IdentitySource didn't match. Expected: %q, Given: %q", expectedSource, *conf.IdentitySource)
		}
		return nil
	}
}

func testAccCheckAWSAPIGatewayAuthorizerName(conf *apigateway.Authorizer, expectedName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if conf.Name == nil {
			return fmt.Errorf("Empty Name, expected: %q", expectedName)
		}
		if *conf.Name != expectedName {
			return fmt.Errorf("Name didn't match. Expected: %q, Given: %q", expectedName, *conf.Name)
		}
		return nil
	}
}

func testAccCheckAWSAPIGatewayAuthorizerType(conf *apigateway.Authorizer, expectedType string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if conf.Type == nil {
			return fmt.Errorf("Empty Type, expected: %q", expectedType)
		}
		if *conf.Type != expectedType {
			return fmt.Errorf("Type didn't match. Expected: %q, Given: %q", expectedType, *conf.Type)
		}
		return nil
	}
}

func testAccCheckAWSAPIGatewayAuthorizerAuthorizerCredentials(conf *apigateway.Authorizer, expectedCreds *regexp.Regexp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if conf.AuthorizerCredentials == nil {
			return fmt.Errorf("Empty AuthorizerCredentials, expected: %q", expectedCreds)
		}
		if !expectedCreds.MatchString(*conf.AuthorizerCredentials) {
			return fmt.Errorf("AuthorizerCredentials didn't match. Expected: %q, Given: %q",
				expectedCreds, *conf.AuthorizerCredentials)
		}
		return nil
	}
}

func testAccCheckAWSAPIGatewayAuthorizerAuthorizerResultTtlInSeconds(conf *apigateway.Authorizer, expectedTtl *int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if expectedTtl == conf.AuthorizerResultTtlInSeconds {
			return nil
		}
		if expectedTtl == nil && conf.AuthorizerResultTtlInSeconds != nil {
			return fmt.Errorf("Expected empty AuthorizerResultTtlInSeconds, given: %d", *conf.AuthorizerResultTtlInSeconds)
		}
		if conf.AuthorizerResultTtlInSeconds == nil {
			return fmt.Errorf("Empty AuthorizerResultTtlInSeconds, expected: %d", expectedTtl)
		}
		if *conf.AuthorizerResultTtlInSeconds != *expectedTtl {
			return fmt.Errorf("AuthorizerResultTtlInSeconds didn't match. Expected: %d, Given: %d",
				*expectedTtl, *conf.AuthorizerResultTtlInSeconds)
		}
		return nil
	}
}

func testAccCheckAWSAPIGatewayAuthorizerIdentityValidationExpression(conf *apigateway.Authorizer, expectedExpression *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if expectedExpression == conf.IdentityValidationExpression {
			return nil
		}
		if expectedExpression == nil && conf.IdentityValidationExpression != nil {
			return fmt.Errorf("Expected empty IdentityValidationExpression, given: %q", *conf.IdentityValidationExpression)
		}
		if conf.IdentityValidationExpression == nil {
			return fmt.Errorf("Empty IdentityValidationExpression, expected: %q", *expectedExpression)
		}
		if *conf.IdentityValidationExpression != *expectedExpression {
			return fmt.Errorf("IdentityValidationExpression didn't match. Expected: %q, Given: %q",
				*expectedExpression, *conf.IdentityValidationExpression)
		}
		return nil
	}
}

func testAccCheckAWSAPIGatewayAuthorizerExists(n string, res *apigateway.Authorizer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway Authorizer ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigateway

		req := &apigateway.GetAuthorizerInput{
			AuthorizerId: aws.String(rs.Primary.ID),
			RestApiId:    aws.String(rs.Primary.Attributes["rest_api_id"]),
		}
		describe, err := conn.GetAuthorizer(req)
		if err != nil {
			return err
		}

		*res = *describe

		return nil
	}
}

func testAccCheckAWSAPIGatewayAuthorizerDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigateway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_authorizer" {
			continue
		}

		req := &apigateway.GetAuthorizerInput{
			AuthorizerId: aws.String(rs.Primary.ID),
			RestApiId:    aws.String(rs.Primary.Attributes["rest_api_id"]),
		}
		_, err := conn.GetAuthorizer(req)

		if err == nil {
			return fmt.Errorf("API Gateway Authorizer still exists")
		}

		aws2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if aws2err.Code() != "NotFoundException" {
			return err
		}

		return nil
	}

	return nil
}

const testAccAWSAPIGatewayAuthorizerConfig_base = `
resource "aws_api_gateway_rest_api" "test" {
  name = "tf-auth-test"
}

resource "aws_iam_role" "invocation_role" {
  name = "tf_acc_api_gateway_auth_invocation_role"
  path = "/"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "apigateway.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "invocation_policy" {
  name = "default"
  role = "${aws_iam_role.invocation_role.id}"
  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "lambda:InvokeFunction",
      "Effect": "Allow",
      "Resource": "${aws_lambda_function.authorizer.arn}"
    }
  ]
}
EOF
}

resource "aws_iam_role" "iam_for_lambda" {
  name = "tf_acc_iam_for_lambda_api_gateway_authorizer"
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

resource "aws_lambda_function" "authorizer" {
  filename = "test-fixtures/lambdatest.zip"
  source_code_hash = "${base64sha256(file("test-fixtures/lambdatest.zip"))}"
  function_name = "tf_acc_api_gateway_authorizer"
  role = "${aws_iam_role.iam_for_lambda.arn}"
  handler = "exports.example"
	runtime = "nodejs4.3"
}
`

const testAccAWSAPIGatewayAuthorizerConfig = testAccAWSAPIGatewayAuthorizerConfig_base + `
resource "aws_api_gateway_authorizer" "test" {
  name = "tf-acc-test-authorizer"
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  authorizer_uri = "arn:aws:apigateway:region:lambda:path/2015-03-31/functions/${aws_lambda_function.authorizer.arn}/invocations"
  authorizer_credentials = "${aws_iam_role.invocation_role.arn}"
}
`

const testAccAWSAPIGatewayAuthorizerUpdatedConfig = testAccAWSAPIGatewayAuthorizerConfig_base + `
resource "aws_api_gateway_authorizer" "test" {
  name = "tf-acc-test-authorizer_modified"
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  authorizer_uri = "arn:aws:apigateway:region:lambda:path/2015-03-31/functions/${aws_lambda_function.authorizer.arn}/invocations"
  authorizer_credentials = "${aws_iam_role.invocation_role.arn}"
  authorizer_result_ttl_in_seconds = 360
  identity_validation_expression = ".*"
}
`
