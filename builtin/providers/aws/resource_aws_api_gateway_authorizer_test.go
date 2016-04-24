package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAPIGatewayAuthorizer_basic(t *testing.T) {
	var conf apigateway.Authorizer

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayAuthorizerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAPIGatewayAuthorizerConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayAuthorizerExists("aws_api_gateway_authorizer.test", &conf),
					resource.TestCheckResourceAttr("aws_api_gateway_authorizer.test", "name", "test"),
					resource.TestCheckResourceAttr("aws_api_gateway_authorizer.test", "identity_source", "method.request.header.Authorization"),
					resource.TestCheckResourceAttr("aws_api_gateway_authorizer.test", "type", "TOKEN"),
					resource.TestCheckResourceAttr("aws_api_gateway_authorizer.test", "result_in_ttl", "300"),
				),
			},
		},
	})
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
			RestApiId:    aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
		}
		describe, err := conn.GetAuthorizer(req)
		if err != nil {
			return err
		}
		if *describe.Id != rs.Primary.ID {
			return fmt.Errorf("APIGateway Authorizer not found")
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

		req := &apigateway.GetAuthorizersInput{
			RestApiId: aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
		}
		describe, err := conn.GetAuthorizers(req)

		if err == nil {
			if len(describe.Items) != 0 &&
				*describe.Items[0].Id == rs.Primary.ID {
				return fmt.Errorf("API Gateway Authorizer still exists")
			}
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

const testAccAWSAPIGatewayAuthorizerConfig = `
resource "aws_api_gateway_rest_api" "test" {
    name = "test"
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

resource "aws_lambda_function" "lambda_auth" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "lambda_auth"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
}


resource "aws_api_gateway_authorizer" "test" {
    rest_api_id = "${aws_api_gateway_rest_api.test.id}"
    name = "test"
    authorizer_uri = "arn:aws:apigateway:eu-west-1:lambda:path/2015-03-31/functions/${aws_lambda_function.lambda_auth.arn}/invocations"
    credentials = "arn:aws:iam::108016248797:role/lambda_auth"
    identity_source = "method.request.header.Authorization"
    type = "TOKEN"
    result_in_ttl = 300

}
`
