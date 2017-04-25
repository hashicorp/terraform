package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSDataDbInstance_basic(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDBInstanceConfigWithDataSource(rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.aws_db_instance.bar", "allocated_storage"),
					resource.TestCheckResourceAttrSet("data.aws_db_instance.bar", "engine"),
					resource.TestCheckResourceAttrSet("data.aws_db_instance.bar", "db_instance_class"),
					resource.TestCheckResourceAttrSet("data.aws_db_instance.bar", "db_name"),
					resource.TestCheckResourceAttrSet("data.aws_db_instance.bar", "master_username"),
				),
			},
		},
	})
}

func TestAccAWSDataDbInstance_endpoint(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDBInstanceConfigWithDataSource(rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.aws_db_instance.bar", "address"),
					resource.TestCheckResourceAttrSet("data.aws_db_instance.bar", "port"),
					resource.TestCheckResourceAttrSet("data.aws_db_instance.bar", "hosted_zone_id"),
					resource.TestCheckResourceAttrSet("data.aws_db_instance.bar", "endpoint"),
				),
			},
		},
	})
}

func testAccAWSDBInstanceConfigWithDataSource(rInt int) string {
	return fmt.Sprintf(`
resource "aws_db_instance" "bar" {
	identifier = "datasource-test-terraform-%d"

	allocated_storage = 10
	engine = "MySQL"
	instance_class = "db.m1.small"
	name = "baz"
	password = "barbarbarbar"
	username = "foo"

	backup_retention_period = 0
	skip_final_snapshot = true
}

data "aws_db_instance" "bar" {
	db_instance_identifier = "${aws_db_instance.bar.identifier}"
}

`, rInt)
}
