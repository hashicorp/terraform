package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDBSecurityGroup_basic(t *testing.T) {
	var v rds.DBSecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDBSecurityGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBSecurityGroupExists("aws_db_security_group.bar", &v),
					testAccCheckAWSDBSecurityGroupAttributes(&v),
					resource.TestCheckResourceAttr(
						"aws_db_security_group.bar", "name", "secgroup-terraform"),
					resource.TestCheckResourceAttr(
						"aws_db_security_group.bar", "description", "just cuz"),
					resource.TestCheckResourceAttr(
						"aws_db_security_group.bar", "ingress.3363517775.cidr", "10.0.0.1/24"),
					resource.TestCheckResourceAttr(
						"aws_db_security_group.bar", "ingress.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_db_security_group.bar", "tags.#", "1"),
				),
			},
		},
	})
}

func testAccCheckAWSDBSecurityGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).rdsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_db_security_group" {
			continue
		}

		// Try to find the Group
		resp, err := conn.DescribeDBSecurityGroups(
			&rds.DescribeDBSecurityGroupsInput{
				DBSecurityGroupName: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if len(resp.DBSecurityGroups) != 0 &&
				*resp.DBSecurityGroups[0].DBSecurityGroupName == rs.Primary.ID {
				return fmt.Errorf("DB Security Group still exists")
			}
		}

		// Verify the error
		newerr, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if newerr.Code() != "DBSecurityGroupNotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSDBSecurityGroupAttributes(group *rds.DBSecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(group.IPRanges) == 0 {
			return fmt.Errorf("no cidr: %#v", group.IPRanges)
		}

		if *group.IPRanges[0].CIDRIP != "10.0.0.1/24" {
			return fmt.Errorf("bad cidr: %#v", group.IPRanges)
		}

		statuses := make([]string, 0, len(group.IPRanges))
		for _, ips := range group.IPRanges {
			statuses = append(statuses, *ips.Status)
		}

		if statuses[0] != "authorized" {
			return fmt.Errorf("bad status: %#v", statuses)
		}

		if *group.DBSecurityGroupName != "secgroup-terraform" {
			return fmt.Errorf("bad name: %#v", *group.DBSecurityGroupName)
		}

		if *group.DBSecurityGroupDescription != "just cuz" {
			return fmt.Errorf("bad description: %#v", *group.DBSecurityGroupDescription)
		}

		return nil
	}
}

func testAccCheckAWSDBSecurityGroupExists(n string, v *rds.DBSecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DB Security Group ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).rdsconn

		opts := rds.DescribeDBSecurityGroupsInput{
			DBSecurityGroupName: aws.String(rs.Primary.ID),
		}

		resp, err := conn.DescribeDBSecurityGroups(&opts)

		if err != nil {
			return err
		}

		if len(resp.DBSecurityGroups) != 1 ||
			*resp.DBSecurityGroups[0].DBSecurityGroupName != rs.Primary.ID {
			return fmt.Errorf("DB Security Group not found")
		}

		*v = *resp.DBSecurityGroups[0]

		return nil
	}
}

const testAccAWSDBSecurityGroupConfig = `
provider "aws" {
        region = "us-east-1"
}

resource "aws_db_security_group" "bar" {
    name = "secgroup-terraform"
    description = "just cuz"

    ingress {
        cidr = "10.0.0.1/24"
    }

    tags {
		foo = "bar"
    }
}
`
