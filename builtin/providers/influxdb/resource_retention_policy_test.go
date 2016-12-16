package influxdb

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/influxdata/influxdb/client"
)

func TestAccInfluxDBRetentionPolicy_Create(t *testing.T) {
	var dbName string
	var rpName string
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccDataSourceCheckDestroy(&rpName, &dbName),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRetentionPolicyConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccRetentionPolicyCheckExists("influxdb_retention_policy.basic", &rpName, &dbName),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "name", "basic",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "database", "terraform-rp-create",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "duration", "INF",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "is_default", "false",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "replication", "1",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "shard_duration", "",
					),
				),
			},
		},
	})
}

func TestAccInfluxDBRetentionPolicy_EachAttribute(t *testing.T) {
	var dbName string
	var rpName string
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRetentionPolicyConfigEachAttribute,
				Check: resource.ComposeTestCheckFunc(
					testAccRetentionPolicyCheckExists("influxdb_retention_policy.basic", &rpName, &dbName),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "name", "basic",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "database", "terraform-rp-eachattribute",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "duration", "12h",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "is_default", "true",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "replication", "2",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "shard_duration", "48h",
					),
				),
			},
		},
	})
}

func TestAccInfluxDBRetentionPolicy_Alter(t *testing.T) {
	var dbName string
	var rpName string
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRetentionPolicyConfigAlterInitial,
				Check: resource.ComposeTestCheckFunc(
					testAccRetentionPolicyCheckExists("influxdb_retention_policy.basic", &rpName, &dbName),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "name", "basic",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "database", "terraform-rp-alter",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "duration", "INF",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "is_default", "false",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "replication", "1",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "shard_duration", "",
					),
				),
			},
			resource.TestStep{
				Config: testAccRetentionPolicyConfigAlter,
				Check: resource.ComposeTestCheckFunc(
					testAccRetentionPolicyCheckExists("influxdb_retention_policy.basic", &rpName, &dbName),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "name", "basic",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "database", "terraform-rp-alter",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "duration", "48h",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "is_default", "true",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "replication", "1",
					),
					resource.TestCheckResourceAttr(
						"influxdb_retention_policy.basic", "shard_duration", "24h",
					),
				),
			},
		},
	})
}

func testAccRetentionPolicyCheckExists(n string, policy *string, database *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No RetentionPolicy id set")
		}

		conn := testAccProvider.Meta().(*client.Client)

		queryStr := fmt.Sprintf("SHOW RETENTION POLICIES ON %s", quoteIdentifier(rs.Primary.Attributes["database"]))
		query := client.Query{
			Command: queryStr,
		}

		*database = rs.Primary.Attributes["database"]

		resp, err := conn.Query(query)
		if err != nil {
			return err
		}

		if resp.Err != nil {
			return resp.Err
		}

		for _, series := range resp.Results[0].Series {
			for _, result := range series.Values {
				if result[0].(string) == rs.Primary.Attributes["name"] {
					*policy = rs.Primary.Attributes["name"]
					return nil
				}
			}
		}

		return fmt.Errorf("Retention policy %q does not exist", rs.Primary.Attributes["name"])
	}
}

func testAccDataSourceCheckDestroy(policy *string, database *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*client.Client)

		queryStr := fmt.Sprintf("SHOW RETENTION POLICIES ON %s", quoteIdentifier(*database))
		query := client.Query{
			Command: queryStr,
		}

		resp, err := conn.Query(query)
		if err != nil {
			return err
		}

		if resp.Err != nil {
			return resp.Err
		}

		for _, series := range resp.Results[0].Series {
			for _, result := range series.Values {
				if result[0].(string) == *policy {
					return fmt.Errorf("Retention policy still exists [%s]", *policy)
				}
			}
		}

		return nil
	}
}

var testAccRetentionPolicyConfigCreate = `

resource "influxdb_database" "test" {
  name = "terraform-rp-create"
}

resource "influxdb_retention_policy" "basic" {
  name = "basic"
  database = "${influxdb_database.test.name}"
}

`

var testAccRetentionPolicyConfigEachAttribute = `

resource "influxdb_database" "test" {
  name = "terraform-rp-eachattribute"
}

resource "influxdb_retention_policy" "basic" {
  name = "basic"
  database = "${influxdb_database.test.name}"
  duration = "12h"
  is_default = true
  replication = 2
  shard_duration = "48h"
}

`

var testAccRetentionPolicyConfigAlterInitial = `

resource "influxdb_database" "test" {
  name = "terraform-rp-alter"
}

resource "influxdb_retention_policy" "basic" {
  name = "basic"
  database = "${influxdb_database.test.name}"
}

`

var testAccRetentionPolicyConfigAlter = `

resource "influxdb_database" "test" {
  name = "terraform-rp-alter"
}

resource "influxdb_retention_policy" "basic" {
  name = "basic"
  database = "${influxdb_database.test.name}"
  duration = "48h"
  is_default = true
  replication = 1
  shard_duration = "24h"
}

`
