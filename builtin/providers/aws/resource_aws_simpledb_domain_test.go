package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/simpledb"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSimpleDBDomain_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSimpleDBDomainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSimpleDBDomainConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSimpleDBDomainExists("aws_simpledb_domain.test_domain"),
				),
			},
		},
	})
}

func testAccCheckAWSSimpleDBDomainDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).simpledbconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_simpledb_domain" {
			continue
		}

		input := &simpledb.DomainMetadataInput{
			DomainName: aws.String(rs.Primary.ID),
		}
		_, err := conn.DomainMetadata(input)
		if err == nil {
			return fmt.Errorf("Domain exists when it should be destroyed!")
		}

		// Verify the error is an API error, not something else
		_, ok := err.(awserr.Error)
		if !ok {
			return err
		}
	}

	return nil
}

func testAccCheckAWSSimpleDBDomainExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SimpleDB domain with that name exists")
		}

		conn := testAccProvider.Meta().(*AWSClient).simpledbconn
		input := &simpledb.DomainMetadataInput{
			DomainName: aws.String(rs.Primary.ID),
		}
		_, err := conn.DomainMetadata(input)
		return err
	}
}

var testAccAWSSimpleDBDomainConfig = `
resource "aws_simpledb_domain" "test_domain" {
  name = "terraform-test-domain"
}
`
