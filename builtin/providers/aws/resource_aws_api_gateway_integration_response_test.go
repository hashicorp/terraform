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

func TestAccAWSAPIGatewayIntegrationResponse_basic(t *testing.T) {
	var conf apigateway.IntegrationResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayIntegrationResponseDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAPIGatewayIntegrationResponseConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationResponseExists("aws_api_gateway_integration_response.test", &conf),
					testAccCheckAWSAPIGatewayIntegrationResponseAttributes(&conf),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_integration_response.test", "response_templates.application/json", ""),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_integration_response.test", "response_templates.application/xml", "#set($inputRoot = $input.path('$'))\n{ }"),
					resource.TestCheckNoResourceAttr(
						"aws_api_gateway_integration_response.test", "content_handling"),
				),
			},

			resource.TestStep{
				Config: testAccAWSAPIGatewayIntegrationResponseConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationResponseExists("aws_api_gateway_integration_response.test", &conf),
					testAccCheckAWSAPIGatewayIntegrationResponseAttributesUpdate(&conf),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_integration_response.test", "response_templates.application/json", "$input.path('$')"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_integration_response.test", "response_templates.application/xml", ""),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_integration_response.test", "content_handling", "CONVERT_TO_BINARY"),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewayIntegrationResponseAttributes(conf *apigateway.IntegrationResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *conf.StatusCode != "400" {
			return fmt.Errorf("wrong StatusCode: %q", *conf.StatusCode)
		}
		if conf.ResponseTemplates["application/json"] != nil {
			return fmt.Errorf("wrong ResponseTemplate for application/json")
		}
		if *conf.ResponseTemplates["application/xml"] != "#set($inputRoot = $input.path('$'))\n{ }" {
			return fmt.Errorf("wrong ResponseTemplate for application/xml")
		}
		if conf.SelectionPattern == nil || *conf.SelectionPattern != ".*" {
			return fmt.Errorf("wrong SelectionPattern (expected .*)")
		}
		if *conf.ResponseParameters["method.response.header.Content-Type"] != "integration.response.body.type" {
			return fmt.Errorf("wrong ResponseParameters for header.Content-Type")
		}
		return nil
	}
}

func testAccCheckAWSAPIGatewayIntegrationResponseAttributesUpdate(conf *apigateway.IntegrationResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *conf.StatusCode != "400" {
			return fmt.Errorf("wrong StatusCode: %q", *conf.StatusCode)
		}
		if *conf.ResponseTemplates["application/json"] != "$input.path('$')" {
			return fmt.Errorf("wrong ResponseTemplate for application/json")
		}
		if conf.ResponseTemplates["application/xml"] != nil {
			return fmt.Errorf("wrong ResponseTemplate for application/xml")
		}
		if conf.SelectionPattern != nil {
			return fmt.Errorf("wrong SelectionPattern (expected nil)")
		}
		if conf.ResponseParameters["method.response.header.Content-Type"] != nil {
			return fmt.Errorf("ResponseParameters for header.Content-Type shouldnt exist")
		}

		return nil
	}
}

func testAccCheckAWSAPIGatewayIntegrationResponseExists(n string, res *apigateway.IntegrationResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway Method ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigateway

		req := &apigateway.GetIntegrationResponseInput{
			HttpMethod: aws.String("GET"),
			ResourceId: aws.String(s.RootModule().Resources["aws_api_gateway_resource.test"].Primary.ID),
			RestApiId:  aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
			StatusCode: aws.String(rs.Primary.Attributes["status_code"]),
		}
		describe, err := conn.GetIntegrationResponse(req)
		if err != nil {
			return err
		}

		*res = *describe

		return nil
	}
}

func testAccCheckAWSAPIGatewayIntegrationResponseDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigateway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_integration_response" {
			continue
		}

		req := &apigateway.GetIntegrationResponseInput{
			HttpMethod: aws.String("GET"),
			ResourceId: aws.String(s.RootModule().Resources["aws_api_gateway_resource.test"].Primary.ID),
			RestApiId:  aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
			StatusCode: aws.String(rs.Primary.Attributes["status_code"]),
		}
		_, err := conn.GetIntegrationResponse(req)

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

const testAccAWSAPIGatewayIntegrationResponseConfig = `
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

resource "aws_api_gateway_method_response" "error" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "${aws_api_gateway_method.test.http_method}"
  status_code = "400"

  response_models = {
    "application/json" = "Error"
  }

	response_parameters = {
		"method.response.header.Content-Type" = true
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

  type = "MOCK"
}

resource "aws_api_gateway_integration_response" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "${aws_api_gateway_method.test.http_method}"
  status_code = "${aws_api_gateway_method_response.error.status_code}"
  selection_pattern = ".*"

  response_templates = {
    "application/json" = ""
    "application/xml" = "#set($inputRoot = $input.path('$'))\n{ }"
  }

	response_parameters = {
		"method.response.header.Content-Type" = "integration.response.body.type"
	}
}
`

const testAccAWSAPIGatewayIntegrationResponseConfigUpdate = `
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

resource "aws_api_gateway_method_response" "error" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "${aws_api_gateway_method.test.http_method}"
  status_code = "400"

  response_models = {
    "application/json" = "Error"
  }

	response_parameters = {
		"method.response.header.Content-Type" = true
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

  type = "MOCK"
}

resource "aws_api_gateway_integration_response" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "${aws_api_gateway_method.test.http_method}"
  status_code = "${aws_api_gateway_method_response.error.status_code}"

  response_templates = {
    "application/json" = "$input.path('$')"
    "application/xml" = ""
  }

  content_handling = "CONVERT_TO_BINARY"

}
`
