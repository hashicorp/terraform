package aws

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

//Domain names need to be unique for test purposes, AWS holds onto the domain->cloudfront live for a long time after destruction
func domainNameFromTime() string {
	now := time.Now()
	secs := now.Unix()
	return fmt.Sprintf("a%d.com", secs)
}

func TestAccAWSApiGatewayDomain_basic(t *testing.T) {
	var conf apigateway.DomainName
	name := domainNameFromTime()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSApiGatewayDomainDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSApiGatewayDomainConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSApiGatewayDomainExists("aws_api_gateway_domain.test", name, &conf),
				),
			},
		},
	})
}

func testAccCheckAWSApiGatewayDomainExists(n string, name string, res *apigateway.DomainName) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigateway

		req := &apigateway.GetDomainNameInput{
			DomainName: aws.String(rs.Primary.Attributes["domain_name"]),
		}
		describe, err := conn.GetDomainName(req)
		if err != nil {
			return err
		}

		if *describe.DomainName != rs.Primary.Attributes["domain_name"] {
			return fmt.Errorf("APIGateway domain not found")
		}

		*res = *describe

		return nil
	}
}

func testAccCheckAWSApiGatewayDomainDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigateway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_domain" {
			continue
		}

		req := &apigateway.GetDomainNamesInput{}
		describe, err := conn.GetDomainNames(req)

		if err == nil {
			if len(describe.Items) != 0 &&
				*describe.Items[0].DomainName == rs.Primary.Attributes["domain_name"] {
				return fmt.Errorf("API Gateway domain still exists")
			}
		}

		return err
	}

	return nil
}

func testAccAWSApiGatewayDomainConfig(name string) string {
	return fmt.Sprintf(`resource "aws_api_gateway_domain" "test" {
  domain_name = "%s"
  certificate_name = "test_api_cert"
  certificate_body = "${file("test-fixtures/apigateway.crt")}"
  certificate_private_key = "${file("test-fixtures/apigateway.key")}"
  certificate_chain = "${file("test-fixtures/apigateway.crt")}"
}
`, name)
}
