package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAPIGatewayMethodSettings_basic(t *testing.T) {
	var stage apigateway.Stage
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayMethodSettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAPIGatewayMethodSettingsConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayMethodSettingsExists("aws_api_gateway_method_settings.test", &stage),
					testAccCheckAWSAPIGatewayMethodSettings_metricsEnabled(&stage, "test/GET", true),
					testAccCheckAWSAPIGatewayMethodSettings_loggingLevel(&stage, "test/GET", "INFO"),
					resource.TestCheckResourceAttr("aws_api_gateway_method_settings.test", "settings.#", "1"),
					resource.TestCheckResourceAttr("aws_api_gateway_method_settings.test", "settings.0.metrics_enabled", "true"),
					resource.TestCheckResourceAttr("aws_api_gateway_method_settings.test", "settings.0.logging_level", "INFO"),
				),
			},

			{
				Config: testAccAWSAPIGatewayMethodSettingsConfigUpdate(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayMethodSettingsExists("aws_api_gateway_method_settings.test", &stage),
					testAccCheckAWSAPIGatewayMethodSettings_metricsEnabled(&stage, "test/GET", false),
					testAccCheckAWSAPIGatewayMethodSettings_loggingLevel(&stage, "test/GET", "OFF"),
					resource.TestCheckResourceAttr("aws_api_gateway_method_settings.test", "settings.#", "1"),
					resource.TestCheckResourceAttr("aws_api_gateway_method_settings.test", "settings.0.metrics_enabled", "false"),
					resource.TestCheckResourceAttr("aws_api_gateway_method_settings.test", "settings.0.logging_level", "OFF"),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewayMethodSettings_metricsEnabled(conf *apigateway.Stage, path string, expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		settings, ok := conf.MethodSettings[path]
		if !ok {
			return fmt.Errorf("Expected to find method settings for %q", path)
		}

		if expected && *settings.MetricsEnabled != expected {
			return fmt.Errorf("Expected metrics to be enabled, got %t", *settings.MetricsEnabled)
		}
		if !expected && *settings.MetricsEnabled != expected {
			return fmt.Errorf("Expected metrics to be disabled, got %t", *settings.MetricsEnabled)
		}

		return nil
	}
}

func testAccCheckAWSAPIGatewayMethodSettings_loggingLevel(conf *apigateway.Stage, path string, expectedLevel string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		settings, ok := conf.MethodSettings[path]
		if !ok {
			return fmt.Errorf("Expected to find method settings for %q", path)
		}

		if *settings.LoggingLevel != expectedLevel {
			return fmt.Errorf("Expected logging level to match %q, got %q", expectedLevel, *settings.LoggingLevel)
		}

		return nil
	}
}

func testAccCheckAWSAPIGatewayMethodSettingsExists(n string, res *apigateway.Stage) resource.TestCheckFunc {
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
			StageName: aws.String(s.RootModule().Resources["aws_api_gateway_deployment.test"].Primary.Attributes["stage_name"]),
			RestApiId: aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
		}
		out, err := conn.GetStage(req)
		if err != nil {
			return err
		}

		*res = *out

		return nil
	}
}

func testAccCheckAWSAPIGatewayMethodSettingsDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigateway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_method_settings" {
			continue
		}

		req := &apigateway.GetStageInput{
			StageName: aws.String(s.RootModule().Resources["aws_api_gateway_deployment.test"].Primary.Attributes["stage_name"]),
			RestApiId: aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
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

func testAccAWSAPIGatewayMethodSettingsConfig(rInt int) string {
	return fmt.Sprintf(`
resource "aws_api_gateway_rest_api" "test" {
  name = "tf-acc-test-apig-method-%d"
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

  request_parameters = {
    "method.request.header.Content-Type" = false,
	  "method.request.querystring.page" = true
  }
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "${aws_api_gateway_method.test.http_method}"
  type        = "MOCK"

  request_templates {
    "application/xml" = <<EOF
{
   "body" : $input.json('$')
}
EOF
  }
}

resource "aws_api_gateway_deployment" "test" {
  depends_on = ["aws_api_gateway_integration.test"]
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  stage_name = "dev"
}

resource "aws_api_gateway_method_settings" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  stage_name  = "${aws_api_gateway_deployment.test.stage_name}"
  method_path = "${aws_api_gateway_resource.test.path_part}/${aws_api_gateway_method.test.http_method}"

  settings {
  	metrics_enabled = true
	logging_level = "INFO"
  }
}
`, rInt)
}

func testAccAWSAPIGatewayMethodSettingsConfigUpdate(rInt int) string {
	return fmt.Sprintf(`
resource "aws_api_gateway_rest_api" "test" {
  name = "tf-acc-test-apig-method-%d"
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

  request_parameters = {
    "method.request.header.Content-Type" = false,
	  "method.request.querystring.page" = true
  }
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "${aws_api_gateway_method.test.http_method}"
  type        = "MOCK"

  request_templates {
    "application/xml" = <<EOF
{
   "body" : $input.json('$')
}
EOF
  }
}

resource "aws_api_gateway_deployment" "test" {
  depends_on = ["aws_api_gateway_integration.test"]
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  stage_name = "dev"
}

resource "aws_api_gateway_method_settings" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  stage_name  = "${aws_api_gateway_deployment.test.stage_name}"
  method_path = "${aws_api_gateway_resource.test.path_part}/${aws_api_gateway_method.test.http_method}"

  settings {
  	metrics_enabled = false
	logging_level = "OFF"
  }
}
`, rInt)
}
