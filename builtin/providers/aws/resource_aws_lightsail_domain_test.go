package aws

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSLightsailDomain_basic(t *testing.T) {
	var domain lightsail.Domain
	lightsailDomainName := fmt.Sprintf("tf-test-lightsail-%s.com", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLightsailDomainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLightsailDomainConfig_basic(lightsailDomainName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSLightsailDomainExists("aws_lightsail_domain.domain_test", &domain),
				),
			},
		},
	})
}

func TestAccAWSLightsailDomain_disappears(t *testing.T) {
	var domain lightsail.Domain
	lightsailDomainName := fmt.Sprintf("tf-test-lightsail-%s.com", acctest.RandString(5))

	domainDestroy := func(*terraform.State) error {

		conn := testAccProvider.Meta().(*AWSClient).lightsailconn
		_, err := conn.DeleteDomain(&lightsail.DeleteDomainInput{
			DomainName: aws.String(lightsailDomainName),
		})

		if err != nil {
			return fmt.Errorf("Error deleting Lightsail Domain in disapear test")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLightsailDomainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLightsailDomainConfig_basic(lightsailDomainName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSLightsailDomainExists("aws_lightsail_domain.domain_test", &domain),
					domainDestroy,
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAWSLightsailDomainExists(n string, domain *lightsail.Domain) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("No Lightsail Domain ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).lightsailconn

		resp, err := conn.GetDomain(&lightsail.GetDomainInput{
			DomainName: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if resp == nil || resp.Domain == nil {
			return fmt.Errorf("Domain (%s) not found", rs.Primary.ID)
		}
		*domain = *resp.Domain
		return nil
	}
}

func testAccCheckAWSLightsailDomainDestroy(s *terraform.State) error {

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_lightsail_domain" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).lightsailconn

		resp, err := conn.GetDomain(&lightsail.GetDomainInput{
			DomainName: aws.String(rs.Primary.ID),
		})

		if err == nil {
			if resp.Domain != nil {
				return fmt.Errorf("Lightsail Domain %q still exists", rs.Primary.ID)
			}
		}

		// Verify the error
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NotFoundException" {
				return nil
			}
		}
		return err
	}

	return nil
}

func testAccAWSLightsailDomainConfig_basic(lightsailDomainName string) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}
resource "aws_lightsail_domain" "domain_test" {
  domain_name = "%s"
}
`, lightsailDomainName)
}
