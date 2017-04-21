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

func TestAccAWSAPIGatewayBasePath_basic(t *testing.T) {
	var conf apigateway.BasePathMapping

	// Our test cert is for a wildcard on this domain
	name := fmt.Sprintf("%s.tf-acc.invalid", resource.UniqueId())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayBasePathDestroy(name),
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAPIGatewayBasePathConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayBasePathExists("aws_api_gateway_base_path_mapping.test", name, &conf),
				),
			},
		},
	})
}

// https://github.com/hashicorp/terraform/issues/9212
func TestAccAWSAPIGatewayEmptyBasePath_basic(t *testing.T) {
	var conf apigateway.BasePathMapping

	// Our test cert is for a wildcard on this domain
	name := fmt.Sprintf("%s.tf-acc.invalid", resource.UniqueId())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayBasePathDestroy(name),
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAPIGatewayEmptyBasePathConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayEmptyBasePathExists("aws_api_gateway_base_path_mapping.test", name, &conf),
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
			BasePath:   aws.String("tf-acc"),
		}
		describe, err := conn.GetBasePathMapping(req)
		if err != nil {
			return err
		}

		if *describe.BasePath != "tf-acc" {
			return fmt.Errorf("base path mapping not found")
		}

		*res = *describe

		return nil
	}
}

func testAccCheckAWSAPIGatewayEmptyBasePathExists(n string, name string, res *apigateway.BasePathMapping) resource.TestCheckFunc {
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
			BasePath:   aws.String(""),
		}
		describe, err := conn.GetBasePathMapping(req)
		if err != nil {
			return err
		}

		*res = *describe

		return nil
	}
}

func testAccCheckAWSAPIGatewayBasePathDestroy(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).apigateway

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_api_gateway_rest_api" {
				continue
			}

			req := &apigateway.GetBasePathMappingsInput{
				DomainName: aws.String(name),
			}
			_, err := conn.GetBasePathMappings(req)

			if err != nil {
				if err, ok := err.(awserr.Error); ok && err.Code() == "NotFoundException" {
					return nil
				}
				return err
			}

			return fmt.Errorf("expected error reading deleted base path, but got success")
		}

		return nil
	}
}

func testAccAWSAPIGatewayBasePathConfig(name string) string {
	return fmt.Sprintf(`
resource "aws_api_gateway_rest_api" "test" {
  name = "tf-acc-apigateway-base-path-mapping"
  description = "Terraform Acceptance Tests"
}
# API gateway won't let us deploy an empty API
resource "aws_api_gateway_resource" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  parent_id = "${aws_api_gateway_rest_api.test.root_resource_id}"
  path_part = "tf-acc"
}
resource "aws_api_gateway_method" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "GET"
  authorization = "NONE"
}
resource "aws_api_gateway_integration" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "${aws_api_gateway_method.test.http_method}"
  type = "MOCK"
}
resource "aws_api_gateway_deployment" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  stage_name = "test"
  depends_on = ["aws_api_gateway_integration.test"]
}
resource "aws_api_gateway_base_path_mapping" "test" {
  api_id = "${aws_api_gateway_rest_api.test.id}"
  base_path = "tf-acc"
  stage_name = "${aws_api_gateway_deployment.test.stage_name}"
  domain_name = "${aws_api_gateway_domain_name.test.domain_name}"
}
resource "aws_api_gateway_domain_name" "test" {
  domain_name = "%s"
  certificate_name = "tf-apigateway-base-path-mapping-test"
  certificate_body = <<EOF
%vEOF
  certificate_chain = <<EOF
%vEOF
  certificate_private_key = <<EOF
%vEOF
}
`, name, testAccAWSAPIGatewayCertBody, testAccAWSAPIGatewayCertChain, testAccAWSAPIGatewayCertPrivateKey)
}

func testAccAWSAPIGatewayEmptyBasePathConfig(name string) string {
	return fmt.Sprintf(`
resource "aws_api_gateway_rest_api" "test" {
  name = "tf-acc-apigateway-base-path-mapping"
  description = "Terraform Acceptance Tests"
}
resource "aws_api_gateway_method" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_rest_api.test.root_resource_id}"
  http_method = "GET"
  authorization = "NONE"
}
resource "aws_api_gateway_integration" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_rest_api.test.root_resource_id}"
  http_method = "${aws_api_gateway_method.test.http_method}"
  type = "MOCK"
}
resource "aws_api_gateway_deployment" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  stage_name = "test"
  depends_on = ["aws_api_gateway_integration.test"]
}
resource "aws_api_gateway_base_path_mapping" "test" {
  api_id = "${aws_api_gateway_rest_api.test.id}"
  base_path = ""
  stage_name = "${aws_api_gateway_deployment.test.stage_name}"
  domain_name = "${aws_api_gateway_domain_name.test.domain_name}"
}
resource "aws_api_gateway_domain_name" "test" {
  domain_name = "%s"
  certificate_name = "tf-apigateway-base-path-mapping-test"
  certificate_body = <<EOF
%vEOF
  certificate_chain = <<EOF
%vEOF
  certificate_private_key = <<EOF
%vEOF
}
`, name, testAccAWSAPIGatewayCertBody, testAccAWSAPIGatewayCertChain, testAccAWSAPIGatewayCertPrivateKey)
}
