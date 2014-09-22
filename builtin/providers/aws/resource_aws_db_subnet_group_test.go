package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/rds"
)

func TestAccAWSDbSubnetGroup(t *testing.T) {
	var v rds.DBSubnetGroup

	testCheck := func(*terraform.State) error {
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDbSubnetGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDbSubnetGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDbSubnetGroupExists(
						"aws_db_subnet_group.foo", &v),
					testCheck,
				),
			},
		},
	})
}

func testAccCheckDbSubnetGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.rdsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_db_subnet_group" {
			continue
		}

		// Try to find the resource
		resp, err := conn.DescribeDBSubnetGroups(&rds.DescribeDBSubnetGroups{rs.Primary.ID})
		if err == nil {
			if len(resp.DBSubnetGroups) > 0 {
				return fmt.Errorf("still exist.")
			}

			return nil
		}

		// Verify the error is what we want
		rdserr, ok := err.(*rds.Error)
		if !ok {
			return err
		}
		if rdserr.Code != "DBSubnetGroupNotFoundFault" {
			return err
		}
	}

	return nil
}

func testAccCheckDbSubnetGroupExists(n string, v *rds.DBSubnetGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.rdsconn
		resp, err := conn.DescribeDBSubnetGroups(&rds.DescribeDBSubnetGroups{rs.Primary.ID})
		if err != nil {
			return err
		}
		if len(resp.DBSubnetGroups) == 0 {
			return fmt.Errorf("DbSubnetGroup not found")
		}

		*v = resp.DBSubnetGroups[0]

		return nil
	}
}

const testAccDbSubnetGroupConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_subnet" "bar" {
	cidr_block = "10.1.2.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_db_subnet_group" "foo" {
	name = "foo"
	description = "foo description"
	subnet_ids = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}"]
}
`
