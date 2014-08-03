package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/rds"
)

func TestAccAWSDBInstance(t *testing.T) {
	var v rds.DBInstance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDBInstanceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBInstanceExists("aws_db_instance.bar", &v),
					testAccCheckAWSDBInstanceAttributes(&v),
					resource.TestCheckResourceAttr(
						"aws_db_instance.bar", "identifier", "foobarbaz-test-terraform"),
					resource.TestCheckResourceAttr(
						"aws_db_instance.bar", "allocated_storage", "10"),
					resource.TestCheckResourceAttr(
						"aws_db_instance.bar", "engine", "mysql"),
					resource.TestCheckResourceAttr(
						"aws_db_instance.bar", "engine_version", "5.6.17"),
					resource.TestCheckResourceAttr(
						"aws_db_instance.bar", "instance_class", "db.t1.micro"),
					resource.TestCheckResourceAttr(
						"aws_db_instance.bar", "name", "baz"),
					resource.TestCheckResourceAttr(
						// Shouldn't save password to state
						"aws_db_instance.bar", "password", ""),
					resource.TestCheckResourceAttr(
						"aws_db_instance.bar", "username", "foo"),
					resource.TestCheckResourceAttr(
						"aws_db_instance.bar", "skip_final_snapshot", "true"),
					resource.TestCheckResourceAttr(
						"aws_db_instance.bar", "security_group_names.0", "secfoobarbaz-test-terraform"),
				),
			},
		},
	})
}

func testAccCheckAWSDBInstanceDestroy(s *terraform.State) error {
	conn := testAccProvider.rdsconn

	for _, rs := range s.Resources {
		if rs.Type != "aws_db_instance" {
			continue
		}

		// Try to find the Group
		resp, err := conn.DescribeDBInstances(
			&rds.DescribeDBInstances{
				DBInstanceIdentifier: rs.ID,
			})

		if err == nil {
			if len(resp.DBInstances) != 0 &&
				resp.DBInstances[0].DBInstanceIdentifier == rs.ID {
				return fmt.Errorf("DB Instance still exists")
			}
		}

		// Verify the error
		newerr, ok := err.(*rds.Error)
		if !ok {
			return err
		}
		if newerr.Code != "InvalidDBInstance.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSDBInstanceAttributes(v *rds.DBInstance) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if len(v.DBSecurityGroupNames) == 0 {
			return fmt.Errorf("no sec names: %#v", v.DBSecurityGroupNames)
		}

		if v.Engine != "mysql" {
			return fmt.Errorf("bad engine: %#v", v.Engine)
		}

		if v.EngineVersion != "5.6.17" {
			return fmt.Errorf("bad engine_version: %#v", v.EngineVersion)
		}

		return nil
	}
}

func testAccCheckAWSDBInstanceExists(n string, v *rds.DBInstance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No DB Instance ID is set")
		}

		conn := testAccProvider.rdsconn

		opts := rds.DescribeDBInstances{
			DBInstanceIdentifier: rs.ID,
		}

		resp, err := conn.DescribeDBInstances(&opts)

		if err != nil {
			return err
		}

		if len(resp.DBInstances) != 1 ||
			resp.DBInstances[0].DBInstanceIdentifier != rs.ID {
			return fmt.Errorf("DB Instance not found")
		}

		*v = resp.DBInstances[0]

		return nil
	}
}

const testAccAWSDBInstanceConfig = `
resource "aws_db_security_group" "bar" {
	name = "secfoobarbaz-test-terraform"
	description = "just cuz"

	ingress {
		cidr = "10.0.0.1/24"
	}
}

resource "aws_db_instance" "bar" {
	identifier = "foobarbaz-test-terraform"

	allocated_storage = 10
	engine = "mysql"
	engine_version = "5.6.17"
	instance_class = "db.t1.micro"
	name = "baz"
	password = "barbarbarbar"
	username = "foo"

	skip_final_snapshot = true

	security_group_names = [${aws_db_security_group.bar.name}]
}
`
