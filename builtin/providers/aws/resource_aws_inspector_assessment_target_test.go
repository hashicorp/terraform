package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/inspector"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSInspectorTarget_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSInspectorTargetAssessmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSInspectorTargetAssessment,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSInspectorTargetExists("aws_inspector_assessment_target.foo"),
				),
			},
			{
				Config: testAccCheckAWSInspectorTargetAssessmentModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSInspectorTargetExists("aws_inspector_assessment_target.foo"),
				),
			},
			{
				Config: testAccCheckAWSInspectorTargetAssessmentUpdatedResourceGroup,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSInspectorTargetExists("aws_inspector_assessment_target.foo"),
				),
			},
		},
	})
}

func testAccCheckAWSInspectorTargetAssessmentDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).inspectorconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_inspector_assessment_target" {
			continue
		}

		resp, err := conn.DescribeAssessmentTargets(&inspector.DescribeAssessmentTargetsInput{
			AssessmentTargetArns: []*string{
				aws.String(rs.Primary.ID),
			},
		})

		if err != nil {
			if inspectorerr, ok := err.(awserr.Error); ok && inspectorerr.Code() == "InvalidInputException" {
				return nil
			} else {
				return fmt.Errorf("Error finding Inspector Assessment Target: %s", err)
			}
		}

		if len(resp.AssessmentTargets) > 0 {
			return fmt.Errorf("Found Target, expected none: %s", resp)
		}
	}

	return nil
}

func testAccCheckAWSInspectorTargetExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		return nil
	}
}

var testAccAWSInspectorTargetAssessment = `

resource "aws_inspector_resource_group" "foo" {
	tags {
	  Name  = "bar"
  }
}

resource "aws_inspector_assessment_target" "foo" {
	name = "foo"
	resource_group_arn =  "${aws_inspector_resource_group.foo.arn}"
}`

var testAccCheckAWSInspectorTargetAssessmentModified = `

resource "aws_inspector_resource_group" "foo" {
	tags {
	  Name  = "bar"
  }
}

resource "aws_inspector_assessment_target" "foo" {
	name = "bar"
	resource_group_arn =  "${aws_inspector_resource_group.foo.arn}"
}`

var testAccCheckAWSInspectorTargetAssessmentUpdatedResourceGroup = `

resource "aws_inspector_resource_group" "foo" {
	tags {
	  Name  = "bar"
  }
}

resource "aws_inspector_resource_group" "bar" {
	tags {
	  Name  = "test"
  }
}

resource "aws_inspector_assessment_target" "foo" {
	name = "bar"
	resource_group_arn =  "${aws_inspector_resource_group.bar.arn}"
}`
