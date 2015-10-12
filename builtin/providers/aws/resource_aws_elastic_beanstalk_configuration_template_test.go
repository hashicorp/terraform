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

func TestAccAWSBeanstalkConfigurationTemplate_basic(t *testing.T) {
	var config elasticbeanstalk.ConfigurationSettingsDescription

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkConfigurationTemplateDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBeanstalkConfigurationTemplateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkConfigurationTemplateExists("aws_elastic_beanstalk_configuration_template.tf_template", &config),
				),
			},
		},
	})
}

func testAccCheckBeanstalkConfigurationTemplateDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elasticbeanstalkconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elastic_beanstalk_configuration_template" {
			continue
		}

		// Try to find the Configuration Template
		opts := elasticbeanstalk.DescribeConfigurationSettingsInput{
			TemplateName:    aws.String(rs.Primary.ID),
			ApplicationName: aws.String(rs.Primary.Attributes["application"]),
		}
		resp, err := conn.DescribeConfigurationSettings(&opts)
		if err == nil {
			if len(resp.ConfigurationSettings) > 0 {
				return fmt.Errorf("Elastic Beanstalk Application still exists.")
			}

			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidBeanstalkConfigurationTemplateID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckBeanstalkConfigurationTemplateExists(n string, config *elasticbeanstalk.ConfigurationSettingsDescription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).elasticbeanstalkconn
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Elastic Beanstalk config ID is not set")
		}

		opts := elasticbeanstalk.DescribeConfigurationSettingsInput{
			TemplateName:    aws.String(rs.Primary.ID),
			ApplicationName: aws.String(rs.Primary.Attributes["application"]),
		}
		resp, err := conn.DescribeConfigurationSettings(&opts)
		if err != nil {
			return err
		}
		if len(resp.ConfigurationSettings) == 0 {
			return fmt.Errorf("Elastic Beanstalk Configurations not found.")
		}

		*config = *resp.ConfigurationSettings[0]

		return nil
	}
}

const testAccBeanstalkConfigurationTemplateConfig = `
resource "aws_elastic_beanstalk_application" "tftest" {
  name = "tf-test-name"
  description = "tf-test-desc"
}

#resource "aws_elastic_beanstalk_environment" "tfenvtest" {
#  name = "tf-test-name"
#  application = "${aws_elastic_beanstalk_application.tftest.name}"
#  solution_stack_name = "64bit Amazon Linux 2015.03 v2.0.3 running Go 1.4"
#}

resource "aws_elastic_beanstalk_configuration_template" "tf_template" {
  name = "tf-test-template-config"
  application = "${aws_elastic_beanstalk_application.tftest.name}"
  solution_stack_name = "64bit Amazon Linux 2015.03 v2.0.3 running Go 1.4"
}
`
