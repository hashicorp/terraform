package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCodeDeployApp_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployAppDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCodeDeployApp,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployAppExists("aws_codedeploy_app.foo"),
				),
			},
			resource.TestStep{
				Config: testAccAWSCodeDeployAppModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployAppExists("aws_codedeploy_app.foo"),
				),
			},
		},
	})
}

func testAccCheckAWSCodeDeployAppDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).codedeployconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_codedeploy_app" {
			continue
		}

		resp, err := conn.GetApplication(&codedeploy.GetApplicationInput{
			ApplicationName: aws.String(rs.Primary.ID),
		})

		if err == nil {
			if resp.Application != nil {
				return fmt.Errorf("CodeDeploy app still exists:\n%#v", *resp.Application.ApplicationId)
			}
		}

		return err
	}

	return nil
}

func testAccCheckAWSCodeDeployAppExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		return nil
	}
}

var testAccAWSCodeDeployApp = `
resource "aws_codedeploy_app" "foo" {
	name = "foo"
}`

var testAccAWSCodeDeployAppModified = `
resource "aws_codedeploy_app" "foo" {
	name = "bar"
}`
