package aws

import (
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSBeanstalkAppVersion_basic(t *testing.T) {

	var appVersion elasticbeanstalk.ApplicationVersionDescription

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckApplicationVersionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBeanstalkApplicationVersionConfig(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckApplicationVersionExists("aws_elastic_beanstalk_application_version.default", &appVersion),
				),
			},
		},
	})
}

func testAccCheckApplicationVersionDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elasticbeanstalkconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elastic_beanstalk_application_version" {
			continue
		}

		describeApplicationVersionOpts := &elasticbeanstalk.DescribeApplicationVersionsInput{
			VersionLabels: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeApplicationVersions(describeApplicationVersionOpts)
		if err == nil {
			if len(resp.ApplicationVersions) > 0 {
				return fmt.Errorf("Elastic Beanstalk Application Verson still exists.")
			}

			return nil
		}
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidParameterValue" {
			return err
		}
	}

	return nil
}

func testAccCheckApplicationVersionExists(n string, app *elasticbeanstalk.ApplicationVersionDescription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Elastic Beanstalk Application Version is not set")
		}

		conn := testAccProvider.Meta().(*AWSClient).elasticbeanstalkconn
		describeApplicationVersionOpts := &elasticbeanstalk.DescribeApplicationVersionsInput{
			VersionLabels: []*string{aws.String(rs.Primary.ID)},
		}

		log.Printf("[DEBUG] Elastic Beanstalk Application Version TEST describe opts: %s", describeApplicationVersionOpts)

		resp, err := conn.DescribeApplicationVersions(describeApplicationVersionOpts)
		if err != nil {
			return err
		}
		if len(resp.ApplicationVersions) == 0 {
			return fmt.Errorf("Elastic Beanstalk Application Version not found.")
		}

		*app = *resp.ApplicationVersions[0]

		return nil
	}
}

func testAccBeanstalkApplicationVersionConfig(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "default" {
  bucket = "tftest.applicationversion.bucket-%d"
}

resource "aws_s3_bucket_object" "default" {
  bucket = "${aws_s3_bucket.default.id}"
  key = "beanstalk/python-v1.zip"
  source = "test-fixtures/python-v1.zip"
}

resource "aws_elastic_beanstalk_application" "default" {
  name = "tf-test-name-%d"
  description = "tf-test-desc"
}

resource "aws_elastic_beanstalk_application_version" "default" {
  application = "${aws_elastic_beanstalk_application.default.name}"
  name = "tf-test-version-label"
  bucket = "${aws_s3_bucket.default.id}"
  key = "${aws_s3_bucket_object.default.id}"
 }
 `, randInt, randInt)
}
