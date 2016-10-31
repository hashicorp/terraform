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

func TestAccAWSAPIGatewayClientCertificate_basic(t *testing.T) {
	var conf apigateway.ClientCertificate

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayClientCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAPIGatewayClientCertificateConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayClientCertificateExists("aws_api_gateway_client_certificate.cow", &conf),
					resource.TestCheckResourceAttr("aws_api_gateway_client_certificate.cow", "description", "Hello from TF acceptance test"),
				),
			},
			{
				Config: testAccAWSAPIGatewayClientCertificateConfig_basic_updated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayClientCertificateExists("aws_api_gateway_client_certificate.cow", &conf),
					resource.TestCheckResourceAttr("aws_api_gateway_client_certificate.cow", "description", "Hello from TF acceptance test - updated"),
				),
			},
		},
	})
}

func TestAccAWSAPIGatewayClientCertificate_importBasic(t *testing.T) {
	resourceName := "aws_api_gateway_client_certificate.cow"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayClientCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAPIGatewayClientCertificateConfig_basic,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckAWSAPIGatewayClientCertificateExists(n string, res *apigateway.ClientCertificate) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway Client Certificate ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigateway

		req := &apigateway.GetClientCertificateInput{
			ClientCertificateId: aws.String(rs.Primary.ID),
		}
		out, err := conn.GetClientCertificate(req)
		if err != nil {
			return err
		}

		*res = *out

		return nil
	}
}

func testAccCheckAWSAPIGatewayClientCertificateDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigateway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_client_certificate" {
			continue
		}

		req := &apigateway.GetClientCertificateInput{
			ClientCertificateId: aws.String(rs.Primary.ID),
		}
		out, err := conn.GetClientCertificate(req)
		if err == nil {
			return fmt.Errorf("API Gateway Client Certificate still exists: %s", out)
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

const testAccAWSAPIGatewayClientCertificateConfig_basic = `
resource "aws_api_gateway_client_certificate" "cow" {
  description = "Hello from TF acceptance test"
}
`

const testAccAWSAPIGatewayClientCertificateConfig_basic_updated = `
resource "aws_api_gateway_client_certificate" "cow" {
  description = "Hello from TF acceptance test - updated"
}
`
