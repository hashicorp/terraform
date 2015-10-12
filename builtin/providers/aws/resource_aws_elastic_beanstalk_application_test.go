package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSBeanstalkApp_basic(t *testing.T) {
	var app elasticbeanstalk.ApplicationDescription

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkAppDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBeanstalkAppConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkAppExists("aws_elastic_beanstalk_application.tftest", &app),
				),
			},
		},
	})
}

func testAccCheckBeanstalkAppDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elasticbeanstalkconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elastic_beanstalk_application" {
			continue
		}

		// Try to find the application
		DescribeBeanstalkAppOpts := &elasticbeanstalk.DescribeApplicationsInput{
			ApplicationNames: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeApplications(DescribeBeanstalkAppOpts)
		if err == nil {
			if len(resp.Applications) > 0 {
				return fmt.Errorf("Elastic Beanstalk Application still exists.")
			}

			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidBeanstalkAppID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckBeanstalkAppExists(n string, app *elasticbeanstalk.ApplicationDescription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Elastic Beanstalk app ID is not set")
		}

		conn := testAccProvider.Meta().(*AWSClient).elasticbeanstalkconn
		DescribeBeanstalkAppOpts := &elasticbeanstalk.DescribeApplicationsInput{
			ApplicationNames: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeApplications(DescribeBeanstalkAppOpts)
		if err != nil {
			return err
		}
		if len(resp.Applications) == 0 {
			return fmt.Errorf("Elastic Beanstalk Application not found.")
		}

		*app = *resp.Applications[0]

		return nil
	}
}

const testAccBeanstalkAppConfig = `
resource "aws_elastic_beanstalk_application" "tftest" {
  name = "tf-test-name"
  description = "tf-test-desc"
}
`
