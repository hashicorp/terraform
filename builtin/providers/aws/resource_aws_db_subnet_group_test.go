package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/rds"
)

func TestAccAWSDBSubnetGroup(t *testing.T) {
	var v rds.DBSubnetGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBSubnetGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDBSubnetGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBSubnetGroupExists("aws_db_subnet_group.foo", &v),
					testAccCheckAWSDBSubnetGroupAttributes(&v),
					resource.TestCheckResourceAttr(
						"aws_db_subnet_group.foo", "name", "subgroup-terraform"),
					resource.TestCheckResourceAttr(
						"aws_db_subnet_group.foo", "description", "just cuz"),
					// TODO check subnet ID contents
					resource.TestCheckResourceAttr(
						"aws_db_subnet_group.foo", "subnet_ids.#", "2"),
				),
			},
		},
	})
}

func testAccCheckAWSDBSubnetGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.rdsconn

	for _, rs := range s.Resources {
		if rs.Type != "aws_db_subnet_group" {
			continue
		}

		// Try to find the Group
		resp, err := conn.DescribeDBSubnetGroups(
			&rds.DescribeDBSubnetGroups{
				DBSubnetGroupName: rs.ID,
			})

		if err == nil {
			if len(resp.DBSubnetGroups) != 0 &&
				resp.DBSubnetGroups[0].Name == rs.ID {
				return fmt.Errorf("DB Subnet Group still exists")
			}
		}

		// Verify the error
		_, ok := err.(*rds.Error)
		if !ok {
			return err
		}
	}

	return nil
}

func testAccCheckAWSDBSubnetGroupAttributes(group *rds.DBSubnetGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(group.SubnetIds) == 0 {
			return fmt.Errorf("no subnets: %#v", group.SubnetIds)
		}

		if group.Name != "subgroup-terraform" {
			return fmt.Errorf("bad name: %#v", group.Name)
		}

		if group.Description != "just cuz" {
			return fmt.Errorf("bad description: %#v", group.Description)
		}

		return nil
	}
}

func testAccCheckAWSDBSubnetGroupExists(n string, v *rds.DBSubnetGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No DB Subnet Group ID is set")
		}

		conn := testAccProvider.rdsconn

		opts := rds.DescribeDBSubnetGroups{
			DBSubnetGroupName: rs.ID,
		}

		resp, err := conn.DescribeDBSubnetGroups(&opts)

		if err != nil {
			return err
		}

		if len(resp.DBSubnetGroups) != 1 ||
			resp.DBSubnetGroups[0].Name != rs.ID {
			return fmt.Errorf("DB Subnet Group not found")
		}

		*v = resp.DBSubnetGroups[0]

		return nil
	}
}

const testAccAWSDBSubnetGroupConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "foo" {
    cidr_block = "10.0.0.0/24"
    vpc_id = "${aws_vpc.foo.id}"
    availability_zone = "us-west-2a"
}

resource "aws_subnet" "bar" {
    cidr_block = "10.0.1.0/24"
    vpc_id = "${aws_vpc.foo.id}"
    availability_zone = "us-west-2b"
}

resource "aws_db_subnet_group" "foo" {
    name = "subgroup-terraform"
    description = "just cuz"
    subnet_ids = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}"]
}
`
