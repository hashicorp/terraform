package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCloudWatchLogGroup_basic(t *testing.T) {
	var lg cloudwatchlogs.LogGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchLogGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudWatchLogGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchLogGroupExists("aws_cloudwatch_log_group.foobar", &lg),
					resource.TestCheckResourceAttr("aws_cloudwatch_log_group.foobar", "retention_in_days", "0"),
				),
			},
		},
	})
}

func TestAccAWSCloudWatchLogGroup_retentionPolicy(t *testing.T) {
	var lg cloudwatchlogs.LogGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchLogGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudWatchLogGroupConfig_withRetention,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchLogGroupExists("aws_cloudwatch_log_group.foobar", &lg),
					resource.TestCheckResourceAttr("aws_cloudwatch_log_group.foobar", "retention_in_days", "365"),
				),
			},
			resource.TestStep{
				Config: testAccAWSCloudWatchLogGroupConfigModified_withRetention,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchLogGroupExists("aws_cloudwatch_log_group.foobar", &lg),
					resource.TestCheckResourceAttr("aws_cloudwatch_log_group.foobar", "retention_in_days", "0"),
				),
			},
		},
	})
}

func TestAccAWSCloudWatchLogGroup_multiple(t *testing.T) {
	var lg cloudwatchlogs.LogGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchLogGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudWatchLogGroupConfig_multiple,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchLogGroupExists("aws_cloudwatch_log_group.alpha", &lg),
					resource.TestCheckResourceAttr("aws_cloudwatch_log_group.alpha", "retention_in_days", "14"),
					testAccCheckCloudWatchLogGroupExists("aws_cloudwatch_log_group.beta", &lg),
					resource.TestCheckResourceAttr("aws_cloudwatch_log_group.beta", "retention_in_days", "0"),
					testAccCheckCloudWatchLogGroupExists("aws_cloudwatch_log_group.charlie", &lg),
					resource.TestCheckResourceAttr("aws_cloudwatch_log_group.charlie", "retention_in_days", "3653"),
				),
			},
		},
	})
}

func testAccCheckCloudWatchLogGroupExists(n string, lg *cloudwatchlogs.LogGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).cloudwatchlogsconn
		logGroup, err := lookupCloudWatchLogGroup(conn, rs.Primary.ID, nil)
		if err != nil {
			return err
		}

		*lg = *logGroup

		return nil
	}
}

func testAccCheckAWSCloudWatchLogGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).cloudwatchlogsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_cloudwatch_log_group" {
			continue
		}

		_, err := lookupCloudWatchLogGroup(conn, rs.Primary.ID, nil)
		if err == nil {
			return fmt.Errorf("LogGroup Still Exists: %s", rs.Primary.ID)
		}
	}

	return nil
}

var testAccAWSCloudWatchLogGroupConfig = `
resource "aws_cloudwatch_log_group" "foobar" {
    name = "foo-bar"
}
`

var testAccAWSCloudWatchLogGroupConfig_withRetention = `
resource "aws_cloudwatch_log_group" "foobar" {
    name = "foo-bang"
    retention_in_days = 365
}
`

var testAccAWSCloudWatchLogGroupConfigModified_withRetention = `
resource "aws_cloudwatch_log_group" "foobar" {
    name = "foo-bang"
}
`

var testAccAWSCloudWatchLogGroupConfig_multiple = `
resource "aws_cloudwatch_log_group" "alpha" {
    name = "foo-bar"
    retention_in_days = 14
}
resource "aws_cloudwatch_log_group" "beta" {
    name = "foo-bara"
}
resource "aws_cloudwatch_log_group" "charlie" {
    name = "foo-baraa"
    retention_in_days = 3653
}
`
