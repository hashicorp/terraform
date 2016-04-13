package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAPIGatewaySwaggerAPI_basic(t *testing.T) {
	var conf apigateway.RestApi

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewaySwaggerAPIDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAPIGatewaySwaggerAPIConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewaySwaggerAPIExists("aws_api_gateway_swagger_api.test", &conf),
				),
			},

			resource.TestStep{
				Config: testAccAWSAPIGatewaySwaggerAPIUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewaySwaggerAPIExists("aws_api_gateway_swagger_api.test", &conf),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewaySwaggerAPIExists(n string, res *apigateway.RestApi) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigateway

		req := &apigateway.GetRestApiInput{
			RestApiId: aws.String(rs.Primary.ID),
		}
		describe, err := conn.GetRestApi(req)
		if err != nil {
			return err
		}

		if *describe.Id != rs.Primary.ID {
			return fmt.Errorf("APIGateway not found")
		}

		*res = *describe

		return nil
	}
}

func testAccCheckAWSAPIGatewaySwaggerAPIDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigateway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_rest_api" {
			continue
		}

		req := &apigateway.GetRestApisInput{}
		describe, err := conn.GetRestApis(req)

		if err == nil {
			if len(describe.Items) != 0 &&
				*describe.Items[0].Id == rs.Primary.ID {
				return fmt.Errorf("API Gateway still exists")
			}
		}

		return err
	}

	return nil
}

const testAccAWSAPIGatewaySwaggerAPIConfig = `
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
`

const testAccAWSAPIGatewaySwaggerAPIUpdateConfig = `
resource "aws_api_gateway_swagger_api" "test" {
  swagger = <<EOF
{
  "swagger": "2.0",
  "info": {
    "version": "1.0",
    "title": "Hello World API 2"
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
`
