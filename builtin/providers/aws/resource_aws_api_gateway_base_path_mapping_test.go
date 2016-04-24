package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAPIGatewayBasePath_basic(t *testing.T) {
	var conf apigateway.BasePathMapping
	name := domainNameFromTime()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayBasePathDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAPIGatewayBasePathConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayBasePathExists("aws_api_gateway_base_path.test", name, &conf),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewayBasePathExists(n string, name string, res *apigateway.BasePathMapping) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigateway

		req := &apigateway.GetBasePathMappingInput{
			DomainName: aws.String(name),
			BasePath:   aws.String("/mrtest"),
		}
		describe, err := conn.GetBasePathMapping(req)
		if err != nil {
			return err
		}

		if *describe.BasePath != "/mrtest" {
			return fmt.Errorf("APIGateway not found")
		}

		*res = *describe

		return nil
	}
}

func testAccCheckAWSAPIGatewayBasePathDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigateway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_rest_api" {
			continue
		}

		req := &apigateway.GetBasePathMappingsInput{}
		describe, err := conn.GetBasePathMappings(req)

		if err == nil {
			if len(describe.Items) != 0 &&
				*describe.Items[0].BasePath == "/mrtest" {
				return fmt.Errorf("Base path mapping still exists")
			}
		}

		return err
	}

	return nil
}

func testAccAWSAPIGatewayBasePathConfig(name string) string {
	return fmt.Sprintf(`
resource "aws_api_gateway_swagger_api" "test" {
  swagger = <<EOF
{
  "swagger": "2.0",
  "info": {
    "version": "1.0",
    "title": "Hello World API"
  },
  "paths": {
    "/hello/{user}": {
      "get": {
        "description": "Returns a greeting to the user!",
        "parameters": [
          {
            "name": "user",
            "in": "path",
            "type": "string",
            "required": true,
            "description": "The name of the user to greet."
          }
        ],
        "responses": {
          "200": {
            "description": "Returns the greeting.",
            "schema": {
              "type": "string"
            }
          },
          "400": {
            "description": "Invalid characters in \"user\" were provided."
          }
        }
      }
    }
  }
}
EOF
}

resource "aws_api_gateway_deployment" "test" {
  rest_api_id = "${aws_api_gateway_swagger_api.test.id}"
  stage_name = "test"
}

resource "aws_api_gateway_base_path_mapping" "test" {
  api_id = "${aws_api_gateway_swagger_api.test.id}"
  path = "/mrtest"
  stage = "${aws_api_gateway_deployment.test.stage_name}"
  domain_name = "${aws_api_gateway_domain.test.id}"
}

resource "aws_api_gateway_domain" "test" {
  domain_name = "%s"
  certificate_name = "test_api_cert"
  certificate_body = "${file("test-fixtures/apigateway.crt")}"
  certificate_private_key = "${file("test-fixtures/apigateway.key")}"
  certificate_chain = "${file("test-fixtures/apigateway.crt")}"
}
`, name)
}
