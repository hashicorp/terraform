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
		CheckDestroy: destroyKinesisAnalytics,
		Steps: []resource.TestStep{
			{
				Config: kinesisAnalyticsBasicCreateConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					doesKinesisAnalyticsInstanceExist("aws_kinesis_analytics.test_application", &desc),
					areRootAttributesCorrect(&desc),
				),
			},
		},
	})
}

func TestAccAWSKinesisAnalytics_output_stream_connections(t *testing.T) {
	var desc kinesisanalytics.ApplicationDetail

	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: destroyKinesisAnalytics,
		Steps: []resource.TestStep{
			{
				Config: kinesisAnalyticsCreateWithOutputStreamsConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					doesKinesisAnalyticsInstanceExist("aws_kinesis_analytics.test_application", &desc),
					areRootAttributesCorrect(&desc),
				),
			},
		},
	})
}

func doesKinesisAnalyticsInstanceExist(n string, desc *kinesisanalytics.ApplicationDetail) resource.TestCheckFunc {
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

func areRootAttributesCorrect(desc *kinesisanalytics.ApplicationDetail) resource.TestCheckFunc {
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

func destroyKinesisAnalytics(s *terraform.State) error {
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

func roleConfig(rInt int) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "ka_test_role" {
	name = "terraform-kinesis-analytics-test-role-%d"
	description = "this role has no attached policy. it is just for testing instantiation kinesis analytics connections to other resources onCreate!"
	assume_role_policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
			"Principal": {
				"Service": "kinesisanalytics.amazonaws.com"
			},
			"Action": "sts:AssumeRole"
		}
	]
}
EOF
}
	`, rInt)
}

func kinesisAnalyticsCreateWithOutputStreamsConfig(rInt int) string {
	return roleConfig(rInt) + fmt.Sprintf(`
resource "aws_kinesis_stream" "test_output_stream_a" {
	name             = "terraform-kinesis-analytics-output-a-test-%d"
	shard_count      = 1
}

resource "aws_kinesis_stream" "test_output_stream_b" {
	name             = "terraform-kinesis-analytics-output-b-test-%d"
	shard_count      = 1
}

resource "aws_kinesis_analytics" "test_application" {
	name = "terraform-kinesis-analytics-test-%d"
	application_description = "test description"
	application_code = "SELECT 1\n"
	outputs {
		name = "DESTINATION_SQL_STREAM_A"
		record_format_type = "JSON"
		arn = "${aws_kinesis_stream.test_output_stream_a.arn}"
		role_arn = "${aws_iam_role.ka_test_role.arn}"
	}
	outputs {
		name = "DESTINATION_SQL_STREAM_B"
		record_format_type = "CSV"
		arn = "${aws_kinesis_stream.test_output_stream_b.arn}"
		role_arn = "${aws_iam_role.ka_test_role.arn}"
	}
}`, rInt, rInt, rInt)
}

func kinesisAnalyticsBasicCreateConfig(rInt int) string {
	return fmt.Sprintf(`
resource "aws_kinesis_analytics" "test_application" {
	name = "terraform-kinesis-analytics-test-%d"
	application_description = "test description"
	application_code = "SELECT 1\n"
}`, rInt)
}
