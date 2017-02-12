package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCloudWatchLogStream_basic(t *testing.T) {
	var ls cloudwatchlogs.LogStream
	rName := acctest.RandString(15)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchLogStreamDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudWatchLogStreamConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchLogStreamExists("aws_cloudwatch_log_stream.foobar", &ls),
				),
			},
		},
	})
}

func TestAccAWSCloudWatchLogStream_disappears(t *testing.T) {
	var ls cloudwatchlogs.LogStream
	rName := acctest.RandString(15)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchLogStreamDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudWatchLogStreamConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchLogStreamExists("aws_cloudwatch_log_stream.foobar", &ls),
					testAccCheckCloudWatchLogStreamDisappears(&ls, rName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckCloudWatchLogStreamDisappears(ls *cloudwatchlogs.LogStream, lgn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).cloudwatchlogsconn
		opts := &cloudwatchlogs.DeleteLogStreamInput{
			LogGroupName:  aws.String(lgn),
			LogStreamName: ls.LogStreamName,
		}
		if _, err := conn.DeleteLogStream(opts); err != nil {
			return err
		}
		return nil
	}
}

func testAccCheckCloudWatchLogStreamExists(n string, ls *cloudwatchlogs.LogStream) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		logGroupName := rs.Primary.Attributes["log_group_name"]
		conn := testAccProvider.Meta().(*AWSClient).cloudwatchlogsconn
		logGroup, exists, err := lookupCloudWatchLogStream(conn, rs.Primary.ID, logGroupName, nil)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("Bad: LogStream %q does not exist", rs.Primary.ID)
		}

		*ls = *logGroup

		return nil
	}
}

func testAccCheckAWSCloudWatchLogStreamDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).cloudwatchlogsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_cloudwatch_log_stream" {
			continue
		}

		logGroupName := rs.Primary.Attributes["log_group_name"]
		_, exists, err := lookupCloudWatchLogStream(conn, rs.Primary.ID, logGroupName, nil)
		if err != nil {
			return nil
		}

		if exists {
			return fmt.Errorf("Bad: LogStream still exists: %q", rs.Primary.ID)
		}

	}

	return nil
}

func TestValidateCloudWatchLogStreamName(t *testing.T) {
	validNames := []string{
		"test-log-stream",
		"my_sample_log_stream",
		"012345678",
		"logstream/1234",
	}
	for _, v := range validNames {
		_, errors := validateCloudWatchLogStreamName(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid CloudWatch LogStream name: %q", v, errors)
		}
	}

	invalidNames := []string{
		acctest.RandString(513),
		"",
		"stringwith:colon",
	}
	for _, v := range invalidNames {
		_, errors := validateCloudWatchLogStreamName(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid CloudWatch LogStream name", v)
		}
	}
}

func testAccAWSCloudWatchLogStreamConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_log_group" "foobar" {
    name = "%s"
}

resource "aws_cloudwatch_log_stream" "foobar" {
    name = "%s"
    log_group_name = "${aws_cloudwatch_log_group.foobar.id}"
}
`, rName, rName)
}
