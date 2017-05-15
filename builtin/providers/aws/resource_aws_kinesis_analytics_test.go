package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesisanalytics"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSKinesisAnalytics_basic(t *testing.T) {
	var desc kinesisanalytics.ApplicationDetail

	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKinesisAnalyticsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKinesisAnalyticsConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKinesisAnalyticsExists("aws_kinesis_analytics.test_application", &desc),
					testAccCheckAWSKinesisAnalyticsAttributes(&desc),
				),
			},
		},
	})
}

func testAccCheckKinesisAnalyticsExists(n string, desc *kinesisanalytics.ApplicationDetail) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Kinesis Application ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).kinesisanalyticsconn
		describeOpts := &kinesisanalytics.DescribeApplicationInput{
			ApplicationName: aws.String(rs.Primary.Attributes["name"]),
		}

		resp, err := conn.DescribeApplication(describeOpts)
		if err != nil {
			return err
		}

		*desc = *resp.ApplicationDetail

		return nil
	}
}

func testAccCheckAWSKinesisAnalyticsAttributes(desc *kinesisanalytics.ApplicationDetail) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if !strings.HasPrefix(*desc.ApplicationName, "terraform-kinesis-analytics-test") {
			return fmt.Errorf("Bad Application name: %s", *desc.ApplicationName)
		}

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_kinesis_analytics" {
				continue
			}
			if *desc.ApplicationARN != rs.Primary.Attributes["arn"] {
				return fmt.Errorf("Bad Application ARN\n\t expected: %s\n\tgot: %s\n",
					rs.Primary.Attributes["arn"],
					*desc.ApplicationARN)
			}
			if *desc.ApplicationDescription != rs.Primary.Attributes["application_description"] {
				return fmt.Errorf("Bad Application Description\n\t expected: %s\n\tgot: %s\n",
					rs.Primary.Attributes["application_description"],
					*desc.ApplicationDescription)
			}
			if *desc.ApplicationCode != rs.Primary.Attributes["application_code"] {
				return fmt.Errorf("Bad Application Code\n\t expected: %s\n\tgot: %s\n",
					rs.Primary.Attributes["application_code"],
					desc.ApplicationCode)
			}
		}
		return nil
	}
}

func testAccCheckKinesisAnalyticsDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_kinesis_analytics" {
			continue
		}
		conn := testAccProvider.Meta().(*AWSClient).kinesisanalyticsconn
		describeOpts := &kinesisanalytics.DescribeApplicationInput{
			ApplicationName: aws.String(rs.Primary.Attributes["name"]),
		}
		resp, err := conn.DescribeApplication(describeOpts)
		if err == nil {
			if resp.ApplicationDetail.ApplicationStatus != nil && *resp.ApplicationDetail.ApplicationStatus != "DELETING" {
				return fmt.Errorf("Error: Application still exists")
			}
		}

		return nil

	}

	return nil
}

func testAccKinesisAnalyticsConfig(rInt int) string {
	return fmt.Sprintf(`
resource "aws_kinesis_analytics" "test_application" {
	name = "terraform-kinesis-analytics-test-%d"
	application_description = "test description"
	application_code = "SELECT 1\n"
}`, rInt)
}
