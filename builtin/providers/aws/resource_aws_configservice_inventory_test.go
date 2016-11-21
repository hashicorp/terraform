package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSConfigServiceInventory_basic(t *testing.T) {
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSConfigServiceInventoryDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSConfigServiceInventoryBasicConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSConfigServiceInventoryExists("aws_configservice_inventory.foo"),
				),
			},
		},
	})
}

func testAccCheckAWSConfigServiceInventoryExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ConfigService Inventory ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).configserviceconn

		_, err := conn.DescribeConfigurationRecorders(&configservice.DescribeConfigurationRecordersInput{
			ConfigurationRecorderNames: []*string{
				aws.String(rs.Primary.ID),
			},
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckAWSConfigServiceInventoryDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).configserviceconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_configservice_inventory" {
			continue
		}

		out, err := conn.DescribeConfigurationRecorders(&configservice.DescribeConfigurationRecordersInput{
			ConfigurationRecorderNames: []*string{
				aws.String(rs.Primary.Attributes["name"]),
			},
		})

		if err != nil {
			return err
		}

		if out != nil {
			return fmt.Errorf("Expected AWS ConfigService Inventory to be gone, but was still found")
		}

		return nil
	}

	return fmt.Errorf("Default error in ConfigService Inventory Test")
}

func testAccAWSConfigServiceInventoryBasicConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "test_role" {
  name = "test_role-%s"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "config.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_sns_topic" "inventory_updates" {
  name = "inventory-updates-topic-%s"
}

resource "aws_s3_bucket" "inventory_bucket" {
  bucket = "inventory-bucket-%s"
}

resource "aws_configservice_inventory" "foo" {
  name = "test_inventory-%s"
  role_arn = "${aws_iam_role.test_role.arn}"

  configuration_recorder {
    all_supported = "true"
  }

	delivery_channel {
		sns_topic_arn = "${aws_sns_topic.inventory_updates.arn}"
		s3_bucket_name = "${aws_s3_bucket.inventory_bucket.id}"
	}
}
`, rName, rName, rName, rName)
}
