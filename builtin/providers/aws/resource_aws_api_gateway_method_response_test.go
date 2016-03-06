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

func TestAccAWSAPIGatewayMethodResponse_basic(t *testing.T) {
	var conf apigateway.MethodResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayMethodResponseDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAPIGatewayMethodResponseConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayMethodResponseExists("aws_api_gateway_method_response.error", &conf),
					testAccCheckAWSAPIGatewayMethodResponseAttributes(&conf),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_method_response.error", "status_code", "400"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_method_response.error", "response_models.application/json", "Error"),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewayMethodResponseAttributes(conf *apigateway.MethodResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *conf.StatusCode == "" {
			return fmt.Errorf("empty StatusCode")
		}
		if val, ok := conf.ResponseModels["application/json"]; !ok {
			return fmt.Errorf("missing application/json ResponseModel")
		} else {
			if *val != "Error" {
				return fmt.Errorf("wrong application/json ResponseModel")
			}
		}
		return nil
	}
}

func testAccCheckAWSAPIGatewayMethodResponseExists(n string, res *apigateway.MethodResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway Method ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigateway

		req := &apigateway.GetMethodResponseInput{
			HttpMethod: aws.String("GET"),
			ResourceId: aws.String(s.RootModule().Resources["aws_api_gateway_resource.test"].Primary.ID),
			RestApiId:  aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
			StatusCode: aws.String(rs.Primary.Attributes["status_code"]),
		}
		describe, err := conn.GetMethodResponse(req)
		if err != nil {
			return err
		}

		*res = *describe

		return nil
	}
}

func testAccCheckAWSAPIGatewayMethodResponseDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigateway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_method_response" {
			continue
		}

		req := &apigateway.GetMethodResponseInput{
			HttpMethod: aws.String("GET"),
			ResourceId: aws.String(s.RootModule().Resources["aws_api_gateway_resource.test"].Primary.ID),
			RestApiId:  aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
			StatusCode: aws.String(rs.Primary.Attributes["status_code"]),
		}
		_, err := conn.GetMethodResponse(req)

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

const testAccAWSAPIGatewayMethodResponseConfig = `
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
}
`
