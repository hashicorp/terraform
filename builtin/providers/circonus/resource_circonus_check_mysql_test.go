package circonus

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCirconusCheckMySQL_basic(t *testing.T) {
	checkName := fmt.Sprintf("MySQL binlog total - %s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusCheckBundle,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCirconusCheckMySQLConfigFmt, checkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_check.table_ops", "active", "true"),
					resource.TestCheckResourceAttr("circonus_check.table_ops", "collector.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.table_ops", "collector.2388330941.id", "/broker/1"),
					resource.TestCheckResourceAttr("circonus_check.table_ops", "mysql.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.table_ops", "mysql.3110376931.dsn", "user=mysql host=mydb1.example.org port=3306 password=12345 sslmode=require"),
					resource.TestCheckResourceAttr("circonus_check.table_ops", "mysql.3110376931.query", `select 'binlog', total from (select variable_value as total from information_schema.global_status where variable_name='BINLOG_CACHE_USE') total`),
					resource.TestCheckResourceAttr("circonus_check.table_ops", "name", checkName),
					resource.TestCheckResourceAttr("circonus_check.table_ops", "period", "300s"),
					resource.TestCheckResourceAttr("circonus_check.table_ops", "metric.#", "1"),

					resource.TestCheckResourceAttr("circonus_check.table_ops", "metric.885029470.name", "binlog`total"),
					resource.TestCheckResourceAttr("circonus_check.table_ops", "metric.885029470.tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.table_ops", "metric.885029470.tags.2087084518", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_check.table_ops", "metric.885029470.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.table_ops", "metric.885029470.type", "numeric"),

					resource.TestCheckResourceAttr("circonus_check.table_ops", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.table_ops", "tags.2087084518", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_check.table_ops", "tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.table_ops", "target", "mydb.example.org"),
					resource.TestCheckResourceAttr("circonus_check.table_ops", "type", "mysql"),
				),
			},
		},
	})
}

const testAccCirconusCheckMySQLConfigFmt = `
variable "test_tags" {
  type = "list"
  default = [ "author:terraform", "lifecycle:unittest" ]
}

resource "circonus_check" "table_ops" {
  active = true
  name = "%s"
  period = "300s"

  collector {
    id = "/broker/1"
  }

  mysql {
    dsn = "user=mysql host=mydb1.example.org port=3306 password=12345 sslmode=require"
    query = <<EOF
select 'binlog', total from (select variable_value as total from information_schema.global_status where variable_name='BINLOG_CACHE_USE') total
EOF
  }

  metric {
    name = "binlog` + "`" + `total"
    tags = [ "${var.test_tags}" ]
    type = "numeric"
  }

  tags = [ "${var.test_tags}" ]
  target = "mydb.example.org"
}
`
