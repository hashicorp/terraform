package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func TestAccAWSPlacementGroup_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSPlacementGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSPlacementGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSPlacementGroupExists("aws_placement_group.pg"),
				),
			},
		},
	})
}

func testAccCheckAWSPlacementGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_placement_group" {
			continue
		}

		_, err := conn.DescribePlacementGroups(&ec2.DescribePlacementGroupsInput{
			GroupNames: []*string{aws.String(rs.Primary.Attributes["name"])},
		})
		if err != nil {
			// Verify the error is what we want
			if ae, ok := err.(awserr.Error); ok && ae.Code() == "InvalidPlacementGroup.Unknown" {
				continue
			}
			return err
		}

		return fmt.Errorf("still exists")
	}
	return nil
}

func testAccCheckAWSPlacementGroupExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Placement Group ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		_, err := conn.DescribePlacementGroups(&ec2.DescribePlacementGroupsInput{
			GroupNames: []*string{aws.String(rs.Primary.ID)},
		})

		if err != nil {
			return fmt.Errorf("Placement Group error: %v", err)
		}
		return nil
	}
}

func testAccCheckAWSDestroyPlacementGroup(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Placement Group ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		_, err := conn.DeletePlacementGroup(&ec2.DeletePlacementGroupInput{
			GroupName: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return fmt.Errorf("Error destroying Placement Group (%s): %s", rs.Primary.ID, err)
		}
		return nil
	}
}

var testAccAWSPlacementGroupConfig = `
resource "aws_placement_group" "pg" {
	name = "tf-test-pg"
	strategy = "cluster"
}
`
