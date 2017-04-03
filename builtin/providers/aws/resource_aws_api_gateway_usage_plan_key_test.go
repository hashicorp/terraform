package aws

import (
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAPIGatewayUsagePlanKey_basic(t *testing.T) {
	var conf apigateway.UsagePlanKey
	name := acctest.RandString(10)
	updatedName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayUsagePlanKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSApiGatewayUsagePlanKeyBasicConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayUsagePlanKeyExists("aws_api_gateway_usage_plan_key.main", &conf),
					resource.TestCheckResourceAttr("aws_api_gateway_usage_plan_key.main", "key_type", "API_KEY"),
					resource.TestCheckResourceAttrSet("aws_api_gateway_usage_plan_key.main", "key_id"),
					resource.TestCheckResourceAttrSet("aws_api_gateway_usage_plan_key.main", "key_type"),
					resource.TestCheckResourceAttrSet("aws_api_gateway_usage_plan_key.main", "usage_plan_id"),
					resource.TestCheckResourceAttrSet("aws_api_gateway_usage_plan_key.main", "name"),
					resource.TestCheckResourceAttr("aws_api_gateway_usage_plan_key.main", "value", ""),
				),
			},
			{
				Config: testAccAWSApiGatewayUsagePlanKeyBasicUpdatedConfig(updatedName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayUsagePlanKeyExists("aws_api_gateway_usage_plan_key.main", &conf),
					resource.TestCheckResourceAttr("aws_api_gateway_usage_plan_key.main", "key_type", "API_KEY"),
					resource.TestCheckResourceAttrSet("aws_api_gateway_usage_plan_key.main", "key_id"),
					resource.TestCheckResourceAttrSet("aws_api_gateway_usage_plan_key.main", "key_type"),
					resource.TestCheckResourceAttrSet("aws_api_gateway_usage_plan_key.main", "usage_plan_id"),
					resource.TestCheckResourceAttrSet("aws_api_gateway_usage_plan_key.main", "name"),
					resource.TestCheckResourceAttr("aws_api_gateway_usage_plan_key.main", "value", ""),
				),
			},
			{
				Config: testAccAWSApiGatewayUsagePlanKeyBasicConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayUsagePlanKeyExists("aws_api_gateway_usage_plan_key.main", &conf),
					resource.TestCheckResourceAttr("aws_api_gateway_usage_plan_key.main", "key_type", "API_KEY"),
					resource.TestCheckResourceAttrSet("aws_api_gateway_usage_plan_key.main", "key_id"),
					resource.TestCheckResourceAttrSet("aws_api_gateway_usage_plan_key.main", "key_type"),
					resource.TestCheckResourceAttrSet("aws_api_gateway_usage_plan_key.main", "usage_plan_id"),
					resource.TestCheckResourceAttrSet("aws_api_gateway_usage_plan_key.main", "name"),
					resource.TestCheckResourceAttr("aws_api_gateway_usage_plan_key.main", "value", ""),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewayUsagePlanKeyExists(n string, res *apigateway.UsagePlanKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway Usage Plan Key ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigateway

		req := &apigateway.GetUsagePlanKeyInput{
			UsagePlanId: aws.String(rs.Primary.Attributes["usage_plan_id"]),
			KeyId:       aws.String(rs.Primary.Attributes["key_id"]),
		}
		up, err := conn.GetUsagePlanKey(req)
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] Reading API Gateway Usage Plan Key: %#v", up)

		if *up.Id != rs.Primary.ID {
			return fmt.Errorf("API Gateway Usage Plan Key not found")
		}

		*res = *up

		return nil
	}
}

func testAccCheckAWSAPIGatewayUsagePlanKeyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigateway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_usage_plan_key" {
			continue
		}

		req := &apigateway.GetUsagePlanKeyInput{
			UsagePlanId: aws.String(rs.Primary.ID),
			KeyId:       aws.String(rs.Primary.Attributes["key_id"]),
		}
		describe, err := conn.GetUsagePlanKey(req)

		if err == nil {
			if describe.Id != nil && *describe.Id == rs.Primary.ID {
				return fmt.Errorf("API Gateway Usage Plan Key still exists")
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

const testAccAWSAPIGatewayUsagePlanKeyConfig = `
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

resource "aws_api_gateway_deployment" "foo" {
  depends_on = ["aws_api_gateway_integration.test"]

  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  stage_name = "foo"
  description = "This is a prod stage"
}

resource "aws_api_gateway_usage_plan" "main" {
  name = "%s"
}

resource "aws_api_gateway_usage_plan" "secondary" {
  name = "secondary-%s"
}

resource "aws_api_gateway_api_key" "mykey" {
  name = "demo-%s"

  stage_key {
    rest_api_id = "${aws_api_gateway_rest_api.test.id}"
    stage_name  = "${aws_api_gateway_deployment.foo.stage_name}"
  }
}
`

func testAccAWSApiGatewayUsagePlanKeyBasicConfig(rName string) string {
	return fmt.Sprintf(testAccAWSAPIGatewayUsagePlanKeyConfig+`
resource "aws_api_gateway_usage_plan_key" "main" {
  key_id        = "${aws_api_gateway_api_key.mykey.id}"
  key_type      = "API_KEY"
  usage_plan_id = "${aws_api_gateway_usage_plan.main.id}"
}
`, rName, rName, rName)
}

func testAccAWSApiGatewayUsagePlanKeyBasicUpdatedConfig(rName string) string {
	return fmt.Sprintf(testAccAWSAPIGatewayUsagePlanKeyConfig+`
resource "aws_api_gateway_usage_plan_key" "main" {
  key_id        = "${aws_api_gateway_api_key.mykey.id}"
  key_type      = "API_KEY"
  usage_plan_id = "${aws_api_gateway_usage_plan.secondary.id}"
}
`, rName, rName, rName)
}
