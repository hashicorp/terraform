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

func TestAccAWSAPIGatewayModel_basic(t *testing.T) {
	var conf apigateway.Model

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayModelDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAPIGatewayModelConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayModelExists("aws_api_gateway_model.test", &conf),
					testAccCheckAWSAPIGatewayModelAttributes(&conf),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_model.test", "name", "test"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_model.test", "description", "a test schema"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_model.test", "content_type", "application/json"),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewayModelAttributes(conf *apigateway.Model) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *conf.Name != "test" {
			return fmt.Errorf("Wrong Name: %q", *conf.Name)
		}
		if *conf.Description != "a test schema" {
			return fmt.Errorf("Wrong Description: %q", *conf.Description)
		}
		if *conf.ContentType != "application/json" {
			return fmt.Errorf("Wrong ContentType: %q", *conf.ContentType)
		}

		return nil
	}
}

func testAccCheckAWSAPIGatewayModelExists(n string, res *apigateway.Model) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway Model ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigateway

		req := &apigateway.GetModelInput{
			ModelName: aws.String("test"),
			RestApiId: aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
		}
		describe, err := conn.GetModel(req)
		if err != nil {
			return err
		}
		if *describe.Id != rs.Primary.ID {
			return fmt.Errorf("APIGateway Model not found")
		}

		*res = *describe

		return nil
	}
}

func testAccCheckAWSAPIGatewayModelDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigateway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_model" {
			continue
		}

		req := &apigateway.GetModelsInput{
			RestApiId: aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
		}
		describe, err := conn.GetModels(req)

		if err == nil {
			if len(describe.Items) != 0 &&
				*describe.Items[0].Id == rs.Primary.ID {
				return fmt.Errorf("API Gateway Model still exists")
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

const testAccAWSAPIGatewayModelConfig = `
resource "aws_api_gateway_rest_api" "test" {
  name = "test"
}

resource "aws_api_gateway_model" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  name = "test"
  description = "a test schema"
  content_type = "application/json"
  schema = <<EOF
{
  "type": "object"
}
EOF
}
`
