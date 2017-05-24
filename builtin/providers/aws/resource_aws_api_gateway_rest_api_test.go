package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAPIGatewayRestApi_basic(t *testing.T) {
	var conf apigateway.RestApi

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayRestAPIDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAPIGatewayRestAPIConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayRestAPIExists("aws_api_gateway_rest_api.test", &conf),
					testAccCheckAWSAPIGatewayRestAPINameAttribute(&conf, "bar"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_rest_api.test", "name", "bar"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_rest_api.test", "description", ""),
					resource.TestCheckResourceAttrSet(
						"aws_api_gateway_rest_api.test", "created_date"),
					resource.TestCheckNoResourceAttr(
						"aws_api_gateway_rest_api.test", "binary_media_types"),
				),
			},

			{
				Config: testAccAWSAPIGatewayRestAPIUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayRestAPIExists("aws_api_gateway_rest_api.test", &conf),
					testAccCheckAWSAPIGatewayRestAPINameAttribute(&conf, "test"),
					testAccCheckAWSAPIGatewayRestAPIDescriptionAttribute(&conf, "test"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_rest_api.test", "name", "test"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_rest_api.test", "description", "test"),
					resource.TestCheckResourceAttrSet(
						"aws_api_gateway_rest_api.test", "created_date"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_rest_api.test", "binary_media_types.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_rest_api.test", "binary_media_types.0", "application/octet-stream"),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewayRestAPINameAttribute(conf *apigateway.RestApi, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *conf.Name != name {
			return fmt.Errorf("Wrong Name: %q", *conf.Name)
		}

		return nil
	}
}

func testAccCheckAWSAPIGatewayRestAPIDescriptionAttribute(conf *apigateway.RestApi, description string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *conf.Description != description {
			return fmt.Errorf("Wrong Description: %q", *conf.Description)
		}

		return nil
	}
}

func testAccCheckAWSAPIGatewayRestAPIExists(n string, res *apigateway.RestApi) resource.TestCheckFunc {
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

func testAccCheckAWSAPIGatewayRestAPIDestroy(s *terraform.State) error {
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

const testAccAWSAPIGatewayRestAPIConfig = `
resource "aws_api_gateway_rest_api" "test" {
  name = "bar"
}
`

const testAccAWSAPIGatewayRestAPIUpdateConfig = `
resource "aws_api_gateway_rest_api" "test" {
  name = "test"
  description = "test"
  binary_media_types = ["application/octet-stream"]
}
`
