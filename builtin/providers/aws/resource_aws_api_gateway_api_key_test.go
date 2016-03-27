package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAPIGatewayApiKey_basic(t *testing.T) {
	var conf apigateway.ApiKey

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayApiKeyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAPIGatewayApiKeyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayApiKeyExists("aws_api_gateway_api_key.test", &conf),
					testAccCheckAWSAPIGatewayApiKeyStageKeyAttribute(&conf),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_api_key.test", "name", "foo"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_api_key.test", "description", "bar"),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewayApiKeyStageKeyAttribute(conf *apigateway.ApiKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(conf.StageKeys) != 1 {
			return fmt.Errorf("Expected one apikey. Got %d", len(conf.StageKeys))
		}
		if !strings.Contains(*conf.StageKeys[0], "test") {
			return fmt.Errorf("Expected apikey for test. Got %q", *conf.StageKeys[0])
		}
		return nil
	}
}

func testAccCheckAWSAPIGatewayApiKeyExists(n string, res *apigateway.ApiKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway ApiKey ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigateway

		req := &apigateway.GetApiKeyInput{
			ApiKey: aws.String(rs.Primary.ID),
		}
		describe, err := conn.GetApiKey(req)
		if err != nil {
			return err
		}

		if *describe.Id != rs.Primary.ID {
			return fmt.Errorf("APIGateway ApiKey not found")
		}

		*res = *describe

		return nil
	}
}

func testAccCheckAWSAPIGatewayApiKeyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigateway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_api_key" {
			continue
		}

		describe, err := conn.GetApiKeys(&apigateway.GetApiKeysInput{})

		if err == nil {
			if len(describe.Items) != 0 &&
				*describe.Items[0].Id == rs.Primary.ID {
				return fmt.Errorf("API Gateway ApiKey still exists")
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

const testAccAWSAPIGatewayApiKeyConfig = `
resource "aws_api_gateway_rest_api" "test" {
  name = "test"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  parent_id = "${aws_api_gateway_rest_api.test.root_resource_id}"
  path_part = "test"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "GET"
  authorization = "NONE"
}

resource "aws_api_gateway_method_response" "error" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "${aws_api_gateway_method.test.http_method}"
  status_code = "400"
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "${aws_api_gateway_method.test.http_method}"

  type = "HTTP"
  uri = "https://www.google.de"
  integration_http_method = "GET"
}

resource "aws_api_gateway_integration_response" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "${aws_api_gateway_integration.test.http_method}"
  status_code = "${aws_api_gateway_method_response.error.status_code}"
}

resource "aws_api_gateway_deployment" "test" {
  depends_on = ["aws_api_gateway_integration.test"]

  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  stage_name = "test"
  description = "This is a test"

  variables = {
    "a" = "2"
  }
}

resource "aws_api_gateway_api_key" "test" {
  name = "foo"
  description = "bar"
  enabled = true

  stage_key {
    rest_api_id = "${aws_api_gateway_rest_api.test.id}"
    stage_name = "${aws_api_gateway_deployment.test.stage_name}"
  }
}
`
