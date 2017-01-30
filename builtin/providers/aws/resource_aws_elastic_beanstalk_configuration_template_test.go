package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/hashicorp/terraform/helper/acctest"
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
				Config: testAccBeanstalkConfigurationTemplateConfig(acctest.RandString(5)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkConfigurationTemplateExists("aws_elastic_beanstalk_configuration_template.tf_template", &config),
				),
			},
		},
	})
}

func TestAccAWSBeanstalkConfigurationTemplate_VPC(t *testing.T) {
	var config elasticbeanstalk.ConfigurationSettingsDescription

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkConfigurationTemplateDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBeanstalkConfigurationTemplateConfig_VPC(acctest.RandString(5)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkConfigurationTemplateExists("aws_elastic_beanstalk_configuration_template.tf_template", &config),
				),
			},
		},
	})
}

func TestAccAWSBeanstalkConfigurationTemplate_Setting(t *testing.T) {
	var config elasticbeanstalk.ConfigurationSettingsDescription

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBeanstalkConfigurationTemplateDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBeanstalkConfigurationTemplateConfig_Setting(acctest.RandString(5)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBeanstalkConfigurationTemplateExists("aws_elastic_beanstalk_configuration_template.tf_template", &config),
					resource.TestCheckResourceAttr(
						"aws_elastic_beanstalk_configuration_template.tf_template", "setting.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_elastic_beanstalk_configuration_template.tf_template", "setting.4112217815.value", "m1.small"),
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

		switch {
		case ec2err.Code() == "InvalidBeanstalkConfigurationTemplateID.NotFound":
			return nil
		// This error can be returned when the beanstalk application no longer exists.
		case ec2err.Code() == "InvalidParameterValue":
			return nil
		default:
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

func testAccBeanstalkConfigurationTemplateConfig(r string) string {
	return fmt.Sprintf(`
resource "aws_elastic_beanstalk_application" "tftest" {
  name = "tf-test-%s"
  description = "tf-test-desc-%s"
}

resource "aws_elastic_beanstalk_configuration_template" "tf_template" {
  name = "tf-test-template-config"
  application = "${aws_elastic_beanstalk_application.tftest.name}"
  solution_stack_name = "64bit Amazon Linux running Python"
}`, r, r)
}

func testAccBeanstalkConfigurationTemplateConfig_VPC(name string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "tf_b_test" {
  cidr_block = "10.0.0.0/16"

  tags {
    Name = "beanstalk_crash"
  }
}

resource "aws_subnet" "main" {
  vpc_id     = "${aws_vpc.tf_b_test.id}"
  cidr_block = "10.0.0.0/24"

  tags {
    Name = "subnet-count-test"
  }
}

resource "aws_elastic_beanstalk_application" "tftest" {
  name        = "tf-test-%s"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_configuration_template" "tf_template" {
  name        = "tf-test-%s"
  application = "${aws_elastic_beanstalk_application.tftest.name}"

  solution_stack_name = "64bit Amazon Linux running Python"

  setting {
    namespace = "aws:ec2:vpc"
    name      = "VPCId"
    value     = "${aws_vpc.tf_b_test.id}"
  }

  setting {
    namespace = "aws:ec2:vpc"
    name      = "Subnets"
    value     = "${aws_subnet.main.id}"
  }
}
`, name, name)
}

func testAccBeanstalkConfigurationTemplateConfig_Setting(name string) string {
	return fmt.Sprintf(`
resource "aws_elastic_beanstalk_application" "tftest" {
  name        = "tf-test-%s"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_configuration_template" "tf_template" {
  name        = "tf-test-%s"
  application = "${aws_elastic_beanstalk_application.tftest.name}"

  solution_stack_name = "64bit Amazon Linux running Python"

  setting {
    namespace = "aws:autoscaling:launchconfiguration"
    name      = "InstanceType"
    value     = "m1.small"
  }

}
`, name, name)
}
