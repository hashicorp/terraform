package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSfnActivity_basic(t *testing.T) {
	name := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSfnActivityDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSfnActivityBasicConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSfnActivityExists("aws_sfn_activity.foo"),
					resource.TestCheckResourceAttr("aws_sfn_activity.foo", "name", name),
					resource.TestCheckResourceAttrSet("aws_sfn_activity.foo", "creation_date"),
				),
			},
		},
	})
}

func testAccCheckAWSSfnActivityExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Step Function ID set")
		}

		conn := testAccProvider.Meta().(*AWSClient).sfnconn

		_, err := conn.DescribeActivity(&sfn.DescribeActivityInput{
			ActivityArn: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckAWSSfnActivityDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).sfnconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_sfn_activity" {
			continue
		}

		out, err := conn.DescribeActivity(&sfn.DescribeActivityInput{
			ActivityArn: aws.String(rs.Primary.ID),
		})

		if err != nil {
			if wserr, ok := err.(awserr.Error); ok && wserr.Code() == "ActivityDoesNotExist" {
				return nil
			}
			return err
		}

		if out != nil {
			return fmt.Errorf("Expected AWS Step Function Activity to be destroyed, but was still found")
		}

		return nil
	}

	return fmt.Errorf("Default error in Step Function Test")
}

func testAccAWSSfnActivityBasicConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_sfn_activity" "foo" {
  name = "%s"
}
`, rName)
}
