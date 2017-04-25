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

func TestAccAWSAPIGatewayIntegration_basic(t *testing.T) {
	var conf apigateway.Integration

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAPIGatewayIntegrationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists("aws_api_gateway_integration.test", &conf),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "type", "HTTP"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "integration_http_method", "GET"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "uri", "https://www.google.de"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "passthrough_behavior", "WHEN_NO_MATCH"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "content_handling", "CONVERT_TO_TEXT"),
					resource.TestCheckNoResourceAttr("aws_api_gateway_integration.test", "credentials"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_parameters.%", "2"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_parameters.integration.request.header.X-Authorization", "'static'"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_parameters.integration.request.header.X-Foo", "'Bar'"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_templates.%", "2"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_templates.application/json", ""),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_templates.application/xml", "#set($inputRoot = $input.path('$'))\n{ }"),
				),
			},

			{
				Config: testAccAWSAPIGatewayIntegrationConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists("aws_api_gateway_integration.test", &conf),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "type", "HTTP"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "integration_http_method", "GET"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "uri", "https://www.google.de"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "passthrough_behavior", "WHEN_NO_MATCH"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "content_handling", "CONVERT_TO_TEXT"),
					resource.TestCheckNoResourceAttr("aws_api_gateway_integration.test", "credentials"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_parameters.%", "2"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_parameters.integration.request.header.X-Authorization", "'updated'"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_parameters.integration.request.header.X-FooBar", "'Baz'"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_templates.%", "2"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_templates.application/json", "{'foobar': 'bar}"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_templates.text/html", "<html>Foo</html>"),
				),
			},

			{
				Config: testAccAWSAPIGatewayIntegrationConfigUpdateNoTemplates,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists("aws_api_gateway_integration.test", &conf),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "type", "HTTP"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "integration_http_method", "GET"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "uri", "https://www.google.de"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "passthrough_behavior", "WHEN_NO_MATCH"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "content_handling", "CONVERT_TO_TEXT"),
					resource.TestCheckNoResourceAttr("aws_api_gateway_integration.test", "credentials"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_parameters.%", "0"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_templates.%", "0"),
				),
			},

			{
				Config: testAccAWSAPIGatewayIntegrationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists("aws_api_gateway_integration.test", &conf),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "type", "HTTP"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "integration_http_method", "GET"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "uri", "https://www.google.de"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "passthrough_behavior", "WHEN_NO_MATCH"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "content_handling", "CONVERT_TO_TEXT"),
					resource.TestCheckNoResourceAttr("aws_api_gateway_integration.test", "credentials"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_parameters.%", "2"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_parameters.integration.request.header.X-Authorization", "'static'"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_templates.%", "2"),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_templates.application/json", ""),
					resource.TestCheckResourceAttr("aws_api_gateway_integration.test", "request_templates.application/xml", "#set($inputRoot = $input.path('$'))\n{ }"),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewayIntegrationExists(n string, res *apigateway.Integration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway Method ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigateway

		req := &apigateway.GetIntegrationInput{
			HttpMethod: aws.String("GET"),
			ResourceId: aws.String(s.RootModule().Resources["aws_api_gateway_resource.test"].Primary.ID),
			RestApiId:  aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
		}
		describe, err := conn.GetIntegration(req)
		if err != nil {
			return err
		}

		*res = *describe

		return nil
	}
}

func testAccCheckAWSAPIGatewayIntegrationDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigateway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_integration" {
			continue
		}

		req := &apigateway.GetIntegrationInput{
			HttpMethod: aws.String("GET"),
			ResourceId: aws.String(s.RootModule().Resources["aws_api_gateway_resource.test"].Primary.ID),
			RestApiId:  aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
		}
		_, err := conn.GetIntegration(req)

		if err == nil {
			return fmt.Errorf("API Gateway Method still exists")
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

const testAccAWSAPIGatewayIntegrationConfig = `
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

  request_models = {
    "application/json" = "Error"
  }
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "${aws_api_gateway_method.test.http_method}"

  request_templates = {
    "application/json" = ""
    "application/xml" = "#set($inputRoot = $input.path('$'))\n{ }"
  }

  request_parameters = {
    "integration.request.header.X-Authorization" = "'static'"
    "integration.request.header.X-Foo" = "'Bar'"
  }

  type = "HTTP"
  uri = "https://www.google.de"
  integration_http_method = "GET"
  passthrough_behavior = "WHEN_NO_MATCH"
  content_handling = "CONVERT_TO_TEXT"
}
`

const testAccAWSAPIGatewayIntegrationConfigUpdate = `
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

  request_models = {
    "application/json" = "Error"
  }
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "${aws_api_gateway_method.test.http_method}"

  request_templates = {
    "application/json" = "{'foobar': 'bar}"
    "text/html" = "<html>Foo</html>"
  }

  request_parameters = {
    "integration.request.header.X-Authorization" = "'updated'"
    "integration.request.header.X-FooBar" = "'Baz'"
  }

  type = "HTTP"
  uri = "https://www.google.de"
  integration_http_method = "GET"
  passthrough_behavior = "WHEN_NO_MATCH"
  content_handling = "CONVERT_TO_TEXT"
}
`

const testAccAWSAPIGatewayIntegrationConfigUpdateNoTemplates = `
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

  request_models = {
    "application/json" = "Error"
  }
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "${aws_api_gateway_method.test.http_method}"

  type = "HTTP"
  uri = "https://www.google.de"
  integration_http_method = "GET"
  passthrough_behavior = "WHEN_NO_MATCH"
  content_handling = "CONVERT_TO_TEXT"
}
`
