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

func TestAccAWSAPIGatewayStage_basic(t *testing.T) {
	var conf apigateway.Stage

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayStageDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAPIGatewayStageConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayStageExists("aws_api_gateway_stage.test", &conf),
					resource.TestCheckResourceAttr("aws_api_gateway_stage.test", "stage_name", "prod"),
					resource.TestCheckResourceAttr("aws_api_gateway_stage.test", "cache_cluster_enabled", "true"),
					resource.TestCheckResourceAttr("aws_api_gateway_stage.test", "cache_cluster_size", "0.5"),
				),
			},
			resource.TestStep{
				Config: testAccAWSAPIGatewayStageConfig_updated(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayStageExists("aws_api_gateway_stage.test", &conf),
					resource.TestCheckResourceAttr("aws_api_gateway_stage.test", "stage_name", "prod"),
					resource.TestCheckResourceAttr("aws_api_gateway_stage.test", "cache_cluster_enabled", "false"),
				),
			},
			resource.TestStep{
				Config: testAccAWSAPIGatewayStageConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayStageExists("aws_api_gateway_stage.test", &conf),
					resource.TestCheckResourceAttr("aws_api_gateway_stage.test", "stage_name", "prod"),
					resource.TestCheckResourceAttr("aws_api_gateway_stage.test", "cache_cluster_enabled", "true"),
					resource.TestCheckResourceAttr("aws_api_gateway_stage.test", "cache_cluster_size", "0.5"),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewayStageExists(n string, res *apigateway.Stage) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway Stage ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigateway

		req := &apigateway.GetStageInput{
			RestApiId: aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
			StageName: aws.String(rs.Primary.Attributes["stage_name"]),
		}
		out, err := conn.GetStage(req)
		if err != nil {
			return err
		}

		*res = *out

		return nil
	}
}

func testAccCheckAWSAPIGatewayStageDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigateway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_stage" {
			continue
		}

		req := &apigateway.GetStageInput{
			RestApiId: aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
			StageName: aws.String(rs.Primary.Attributes["stage_name"]),
		}
		out, err := conn.GetStage(req)
		if err == nil {
			return fmt.Errorf("API Gateway Stage still exists: %s", out)
		}

		awsErr, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if awsErr.Code() != "NotFoundException" {
			return err
		}

		return nil
	}

	return nil
}

const testAccAWSAPIGatewayStageConfig_base = `
resource "aws_api_gateway_rest_api" "test" {
  name = "tf-acc-test"
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
  uri = "https://www.google.co.uk"
  integration_http_method = "GET"
}

resource "aws_api_gateway_integration_response" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "${aws_api_gateway_integration.test.http_method}"
  status_code = "${aws_api_gateway_method_response.error.status_code}"
}

resource "aws_api_gateway_deployment" "dev" {
  depends_on = ["aws_api_gateway_integration.test"]

  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  stage_name = "dev"
  description = "This is a dev env"

  variables = {
    "a" = "2"
  }
}
`

func testAccAWSAPIGatewayStageConfig_basic() string {
	return testAccAWSAPIGatewayStageConfig_base + `
resource "aws_api_gateway_stage" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  stage_name = "prod"
  deployment_id = "${aws_api_gateway_deployment.dev.id}"
  cache_cluster_enabled = true
  cache_cluster_size = "0.5"
  variables {
    one = "1"
    two = "2"
  }
}
`
}

func testAccAWSAPIGatewayStageConfig_updated() string {
	return testAccAWSAPIGatewayStageConfig_base + `
resource "aws_api_gateway_stage" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  stage_name = "prod"
  deployment_id = "${aws_api_gateway_deployment.dev.id}"
  cache_cluster_enabled = false
  description = "Hello world"
  variables {
    one = "1"
    three = "3"
  }
}
`
}
