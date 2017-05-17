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

func TestAccAWSAPIGatewayResource_basic(t *testing.T) {
	var conf apigateway.Resource

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAPIGatewayResourceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayResourceExists("aws_api_gateway_resource.test", &conf),
					testAccCheckAWSAPIGatewayResourceAttributes(&conf, "/test"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_resource.test", "path_part", "test"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_resource.test", "path", "/test"),
				),
			},
		},
	})
}

func TestAccAWSAPIGatewayResource_update(t *testing.T) {
	var conf apigateway.Resource

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAPIGatewayResourceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayResourceExists("aws_api_gateway_resource.test", &conf),
					testAccCheckAWSAPIGatewayResourceAttributes(&conf, "/test"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_resource.test", "path_part", "test"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_resource.test", "path", "/test"),
				),
			},

			resource.TestStep{
				Config: testAccAWSAPIGatewayResourceConfig_updatePathPart,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayResourceExists("aws_api_gateway_resource.test", &conf),
					testAccCheckAWSAPIGatewayResourceAttributes(&conf, "/test_changed"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_resource.test", "path_part", "test_changed"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_resource.test", "path", "/test_changed"),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewayResourceAttributes(conf *apigateway.Resource, path string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *conf.Path != path {
			return fmt.Errorf("Wrong Path: %q", conf.Path)
		}

		return nil
	}
}

func testAccCheckAWSAPIGatewayResourceExists(n string, res *apigateway.Resource) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway Resource ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigateway

		req := &apigateway.GetResourceInput{
			ResourceId: aws.String(rs.Primary.ID),
			RestApiId:  aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
		}
		describe, err := conn.GetResource(req)
		if err != nil {
			return err
		}

		if *describe.Id != rs.Primary.ID {
			return fmt.Errorf("APIGateway Resource not found")
		}

		*res = *describe

		return nil
	}
}

func testAccCheckAWSAPIGatewayResourceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigateway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_resource" {
			continue
		}

		req := &apigateway.GetResourcesInput{
			RestApiId: aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
		}
		describe, err := conn.GetResources(req)

		if err == nil {
			if len(describe.Items) != 0 &&
				*describe.Items[0].Id == rs.Primary.ID {
				return fmt.Errorf("API Gateway Resource still exists")
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

const testAccAWSAPIGatewayResourceConfig = `
resource "aws_api_gateway_rest_api" "test" {
  name = "test"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  parent_id = "${aws_api_gateway_rest_api.test.root_resource_id}"
  path_part = "test"
}
`

const testAccAWSAPIGatewayResourceConfig_updatePathPart = `
resource "aws_api_gateway_rest_api" "test" {
  name = "test"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  parent_id = "${aws_api_gateway_rest_api.test.root_resource_id}"
  path_part = "test_changed"
}
`
