package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCloudWatchLogGroup_basic(t *testing.T) {
	var lg cloudwatchlogs.LogGroup
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchLogGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudWatchLogGroupConfig(rInt),
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
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchLogGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudWatchLogGroupConfig_withRetention(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchLogGroupExists("aws_cloudwatch_log_group.foobar", &lg),
					resource.TestCheckResourceAttr("aws_cloudwatch_log_group.foobar", "retention_in_days", "365"),
				),
			},
			resource.TestStep{
				Config: testAccAWSCloudWatchLogGroupConfigModified_withRetention(rInt),
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
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchLogGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudWatchLogGroupConfig_multiple(rInt),
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

func TestAccAWSCloudWatchLogGroup_disappears(t *testing.T) {
	var lg cloudwatchlogs.LogGroup
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchLogGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudWatchLogGroupConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchLogGroupExists("aws_cloudwatch_log_group.foobar", &lg),
					testAccCheckCloudWatchLogGroupDisappears(&lg),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckCloudWatchLogGroupDisappears(lg *cloudwatchlogs.LogGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).cloudwatchlogsconn
		opts := &cloudwatchlogs.DeleteLogGroupInput{
			LogGroupName: lg.LogGroupName,
		}
		if _, err := conn.DeleteLogGroup(opts); err != nil {
			return err
		}
		return nil
	}
}

func testAccCheckCloudWatchLogGroupExists(n string, lg *cloudwatchlogs.LogGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).cloudwatchlogsconn
		logGroup, exists, err := lookupCloudWatchLogGroup(conn, rs.Primary.ID, nil)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("Bad: LogGroup %q does not exist", rs.Primary.ID)
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
		_, exists, err := lookupCloudWatchLogGroup(conn, rs.Primary.ID, nil)
		if err != nil {
			return nil
		}

		if exists {
			return fmt.Errorf("Bad: LogGroup still exists: %q", rs.Primary.ID)
		}

	}

	return nil
}

func testAccAWSCloudWatchLogGroupConfig(rInt int) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_log_group" "foobar" {
    name = "foo-bar-%d"
}
`, rInt)
}

func testAccAWSCloudWatchLogGroupConfig_withRetention(rInt int) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_log_group" "foobar" {
    name = "foo-bar-%d"
    retention_in_days = 365
}
`, rInt)
}

func testAccAWSCloudWatchLogGroupConfigModified_withRetention(rInt int) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_log_group" "foobar" {
    name = "foo-bar-%d"
}
`, rInt)
}

func testAccAWSCloudWatchLogGroupConfig_multiple(rInt int) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_log_group" "alpha" {
    name = "foo-bar-%d"
    retention_in_days = 14
}
resource "aws_cloudwatch_log_group" "beta" {
    name = "foo-bar-%d"
}
resource "aws_cloudwatch_log_group" "charlie" {
    name = "foo-bar-%d"
    retention_in_days = 3653
}
`, rInt, rInt, rInt)
}
