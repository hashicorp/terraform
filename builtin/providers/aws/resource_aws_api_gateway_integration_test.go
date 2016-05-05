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
			resource.TestStep{
				Config: testAccAWSAPIGatewayIntegrationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists("aws_api_gateway_integration.test", &conf),
					testAccCheckAWSAPIGatewayIntegrationAttributes(&conf),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_integration.test", "type", "HTTP"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_integration.test", "integration_http_method", "GET"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_integration.test", "uri", "https://www.google.de"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_integration.test", "request_templates.application/json", ""),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_integration.test", "request_templates.application/xml", "#set($inputRoot = $input.path('$'))\n{ }"),
				),
			},

			resource.TestStep{
				Config: testAccAWSAPIGatewayIntegrationConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists("aws_api_gateway_integration.test", &conf),
					testAccCheckAWSAPIGatewayMockIntegrationAttributes(&conf),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_integration.test", "type", "MOCK"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_integration.test", "integration_http_method", ""),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_integration.test", "uri", ""),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewayMockIntegrationAttributes(conf *apigateway.Integration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *conf.Type != "MOCK" {
			return fmt.Errorf("Wrong Type: %q", *conf.Type)
		}
		if *conf.RequestParameters["integration.request.header.X-Authorization"] != "'updated'" {
			return fmt.Errorf("wrong updated RequestParameters for header.X-Authorization")
		}
		return nil
	}
}

func testAccCheckAWSAPIGatewayIntegrationAttributes(conf *apigateway.Integration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *conf.HttpMethod == "" {
			return fmt.Errorf("empty HttpMethod")
		}
		if *conf.Uri != "https://www.google.de" {
			return fmt.Errorf("wrong Uri")
		}
		if *conf.Type != "HTTP" {
			return fmt.Errorf("wrong Type")
		}
		if conf.RequestTemplates["application/json"] != nil {
			return fmt.Errorf("wrong RequestTemplate for application/json")
		}
		if *conf.RequestTemplates["application/xml"] != "#set($inputRoot = $input.path('$'))\n{ }" {
			return fmt.Errorf("wrong RequestTemplate for application/xml")
		}
		if *conf.RequestParameters["integration.request.header.X-Authorization"] != "'static'" {
			return fmt.Errorf("wrong RequestParameters for header.X-Authorization")
		}
		return nil
	}
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

  request_parameters_in_json = <<PARAMS
  {
	  "integration.request.header.X-Authorization": "'static'"
  }
  PARAMS

  type = "HTTP"
  uri = "https://www.google.de"
  integration_http_method = "GET"
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

  request_parameters_in_json = <<PARAMS
  {
	  "integration.request.header.X-Authorization": "'updated'"
  }
  PARAMS

  type = "MOCK"
}
`
