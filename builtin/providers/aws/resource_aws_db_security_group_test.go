package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/rds"
)

func TestAccAWSDBSecurityGroup(t *testing.T) {
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
						"aws_db_security_group.bar", "ingress.0.cidr", "10.0.0.1/24"),
					resource.TestCheckResourceAttr(
						"aws_db_security_group.bar", "ingress.#", "1"),
				),
			},
		},
	})
}

func testAccCheckAWSDBSecurityGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.rdsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_db_security_group" {
			continue
		}

		// Try to find the Group
		resp, err := conn.DescribeDBSecurityGroups(
			&rds.DescribeDBSecurityGroups{
				DBSecurityGroupName: rs.Primary.ID,
			})

		if err == nil {
			if len(resp.DBSecurityGroups) != 0 &&
				resp.DBSecurityGroups[0].Name == rs.Primary.ID {
				return fmt.Errorf("DB Security Group still exists")
			}
		}

		// Verify the error
		newerr, ok := err.(*rds.Error)
		if !ok {
			return err
		}
		if newerr.Code != "InvalidDBSecurityGroup.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSDBSecurityGroupAttributes(group *rds.DBSecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(group.CidrIps) == 0 {
			return fmt.Errorf("no cidr: %#v", group.CidrIps)
		}

		if group.CidrIps[0] != "10.0.0.1/24" {
			return fmt.Errorf("bad cidr: %#v", group.CidrIps)
		}

		if group.CidrStatuses[0] != "authorized" {
			return fmt.Errorf("bad status: %#v", group.CidrStatuses)
		}

		if group.Name != "secgroup-terraform" {
			return fmt.Errorf("bad name: %#v", group.Name)
		}

		if group.Description != "just cuz" {
			return fmt.Errorf("bad description: %#v", group.Description)
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

		conn := testAccProvider.rdsconn

		opts := rds.DescribeDBSecurityGroups{
			DBSecurityGroupName: rs.Primary.ID,
		}

		resp, err := conn.DescribeDBSecurityGroups(&opts)

		if err != nil {
			return err
		}

		if len(resp.DBSecurityGroups) != 1 ||
			resp.DBSecurityGroups[0].Name != rs.Primary.ID {
			return fmt.Errorf("DB Security Group not found")
		}

		*v = resp.DBSecurityGroups[0]

		return nil
	}
}

const testAccAWSDBSecurityGroupConfig = `
resource "aws_db_security_group" "bar" {
    name = "secgroup-terraform"
    description = "just cuz"

    ingress {
        cidr = "10.0.0.1/24"
    }
}
`
