package alicloud

import (
	"fmt"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/rds"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"log"
	"strings"
	"testing"
)

func TestAccAlicloudDBInstance_basic(t *testing.T) {
	var instance rds.DBInstanceAttribute

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_db_instance.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDBInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDBInstanceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDBInstanceExists(
						"alicloud_db_instance.foo", &instance),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"port",
						"3306"),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"db_instance_storage",
						"10"),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"instance_network_type",
						"Classic"),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"db_instance_net_type",
						"Intranet"),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"engine_version",
						"5.6"),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"engine",
						"MySQL"),
				),
			},
		},
	})

}

func TestAccAlicloudDBInstance_vpc(t *testing.T) {
	var instance rds.DBInstanceAttribute

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_db_instance.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDBInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDBInstance_vpc,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDBInstanceExists(
						"alicloud_db_instance.foo", &instance),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"port",
						"3306"),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"db_instance_storage",
						"10"),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"instance_network_type",
						"VPC"),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"db_instance_net_type",
						"Intranet"),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"engine_version",
						"5.6"),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"engine",
						"MySQL"),
				),
			},
		},
	})

}

func TestC2CAlicloudDBInstance_prepaid_order(t *testing.T) {
	var instance rds.DBInstanceAttribute

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_db_instance.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDBInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDBInstance_prepaid_order,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDBInstanceExists(
						"alicloud_db_instance.foo", &instance),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"port",
						"3306"),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"db_instance_storage",
						"10"),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"instance_network_type",
						"VPC"),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"db_instance_net_type",
						"Intranet"),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"engine_version",
						"5.6"),
					resource.TestCheckResourceAttr(
						"alicloud_db_instance.foo",
						"engine",
						"MySQL"),
				),
			},
		},
	})

}

func TestAccAlicloudDBInstance_multiIZ(t *testing.T) {
	var instance rds.DBInstanceAttribute

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_db_instance.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDBInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDBInstance_multiIZ,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDBInstanceExists(
						"alicloud_db_instance.foo", &instance),
					testAccCheckDBInstanceMultiIZ(&instance),
				),
			},
		},
	})

}

func TestAccAlicloudDBInstance_database(t *testing.T) {
	var instance rds.DBInstanceAttribute

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_db_instance.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDBInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDBInstance_database,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDBInstanceExists(
						"alicloud_db_instance.foo", &instance),
					resource.TestCheckResourceAttr("alicloud_db_instance.foo", "db_mappings.#", "2"),
				),
			},

			resource.TestStep{
				Config: testAccDBInstance_database_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDBInstanceExists(
						"alicloud_db_instance.foo", &instance),
					resource.TestCheckResourceAttr("alicloud_db_instance.foo", "db_mappings.#", "3"),
				),
			},
		},
	})

}

func TestAccAlicloudDBInstance_account(t *testing.T) {
	var instance rds.DBInstanceAttribute

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_db_instance.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDBInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDBInstance_grantDatabasePrivilege2Account,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDBInstanceExists(
						"alicloud_db_instance.foo", &instance),
					resource.TestCheckResourceAttr("alicloud_db_instance.foo", "db_mappings.#", "2"),
					testAccCheckAccountHasPrivilege2Database("alicloud_db_instance.foo", "tester", "foo", "ReadWrite"),
				),
			},
		},
	})

}

func TestAccAlicloudDBInstance_allocatePublicConnection(t *testing.T) {
	var instance rds.DBInstanceAttribute

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_db_instance.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDBInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDBInstance_allocatePublicConnection,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDBInstanceExists(
						"alicloud_db_instance.foo", &instance),
					resource.TestCheckResourceAttr("alicloud_db_instance.foo", "connections.#", "2"),
					testAccCheckHasPublicConnection("alicloud_db_instance.foo"),
				),
			},
		},
	})

}

func TestAccAlicloudDBInstance_backupPolicy(t *testing.T) {
	var policies []map[string]interface{}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_db_instance.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDBInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDBInstance_backup,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBackupPolicyExists(
						"alicloud_db_instance.foo", policies),
					testAccCheckKeyValueInMaps(policies, "backup policy", "preferred_backup_period", "Wednesday,Thursday"),
					testAccCheckKeyValueInMaps(policies, "backup policy", "preferred_backup_time", "00:00Z-01:00Z"),
				),
			},
		},
	})

}

func TestAccAlicloudDBInstance_securityIps(t *testing.T) {
	var ips []map[string]interface{}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_db_instance.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDBInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDBInstance_securityIps,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityIpExists(
						"alicloud_db_instance.foo", ips),
					testAccCheckKeyValueInMaps(ips, "security ip", "security_ips", "127.0.0.1"),
				),
			},

			resource.TestStep{
				Config: testAccDBInstance_securityIpsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityIpExists(
						"alicloud_db_instance.foo", ips),
					testAccCheckKeyValueInMaps(ips, "security ip", "security_ips", "10.168.1.12,100.69.7.112"),
				),
			},
		},
	})

}

func TestAccAlicloudDBInstance_upgradeClass(t *testing.T) {
	var instance rds.DBInstanceAttribute

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_db_instance.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDBInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDBInstance_class,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDBInstanceExists(
						"alicloud_db_instance.foo", &instance),
					resource.TestCheckResourceAttr("alicloud_db_instance.foo", "db_instance_class", "rds.mysql.t1.small"),
				),
			},

			resource.TestStep{
				Config: testAccDBInstance_classUpgrade,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDBInstanceExists(
						"alicloud_db_instance.foo", &instance),
					resource.TestCheckResourceAttr("alicloud_db_instance.foo", "db_instance_class", "rds.mysql.s1.small"),
				),
			},
		},
	})

}

func testAccCheckSecurityIpExists(n string, ips []map[string]interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DB Instance ID is set")
		}

		conn := testAccProvider.Meta().(*AliyunClient).rdsconn
		args := rds.DescribeDBInstanceIPsArgs{
			DBInstanceId: rs.Primary.ID,
		}

		resp, err := conn.DescribeDBInstanceIPs(&args)
		log.Printf("[DEBUG] check instance %s security ip %#v", rs.Primary.ID, resp)

		if err != nil {
			return err
		}

		p := resp.Items.DBInstanceIPArray

		if len(p) < 1 {
			return fmt.Errorf("DB security ip not found")
		}

		ips = flattenDBSecurityIPs(p)
		return nil
	}
}

func testAccCheckDBInstanceMultiIZ(i *rds.DBInstanceAttribute) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if !strings.Contains(i.ZoneId, MULTI_IZ_SYMBOL) {
			return fmt.Errorf("Current region does not support multiIZ.")
		}
		return nil
	}
}

func testAccCheckAccountHasPrivilege2Database(n, accountName, dbName, privilege string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DB instance ID is set")
		}

		conn := testAccProvider.Meta().(*AliyunClient).rdsconn
		if err := conn.WaitForAccountPrivilege(rs.Primary.ID, accountName, dbName, rds.AccountPrivilege(privilege), 50); err != nil {
			return fmt.Errorf("Failed to grant database %s privilege to account %s: %v", dbName, accountName, err)
		}
		return nil
	}
}

func testAccCheckHasPublicConnection(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DB instance ID is set")
		}

		conn := testAccProvider.Meta().(*AliyunClient).rdsconn
		if err := conn.WaitForPublicConnection(rs.Primary.ID, 50); err != nil {
			return fmt.Errorf("Failed to allocate public connection: %v", err)
		}
		return nil
	}
}

func testAccCheckDBInstanceExists(n string, d *rds.DBInstanceAttribute) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DB Instance ID is set")
		}

		client := testAccProvider.Meta().(*AliyunClient)
		attr, err := client.DescribeDBInstanceById(rs.Primary.ID)
		log.Printf("[DEBUG] check instance %s attribute %#v", rs.Primary.ID, attr)

		if err != nil {
			return err
		}

		if attr == nil {
			return fmt.Errorf("DB Instance not found")
		}

		*d = *attr
		return nil
	}
}

func testAccCheckBackupPolicyExists(n string, ps []map[string]interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Backup policy not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DB Instance ID is set")
		}

		conn := testAccProvider.Meta().(*AliyunClient).rdsconn

		args := rds.DescribeBackupPolicyArgs{
			DBInstanceId: rs.Primary.ID,
		}
		resp, err := conn.DescribeBackupPolicy(&args)
		log.Printf("[DEBUG] check instance %s backup policy %#v", rs.Primary.ID, resp)

		if err != nil {
			return err
		}

		var bs []rds.BackupPolicy
		bs = append(bs, resp.BackupPolicy)
		ps = flattenDBBackup(bs)

		return nil
	}
}

func testAccCheckKeyValueInMaps(ps []map[string]interface{}, propName, key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, policy := range ps {
			if policy[key].(string) != value {
				return fmt.Errorf("DB %s attribute '%s' expected %#v, got %#v", propName, key, value, policy[key])
			}
		}
		return nil
	}
}

func testAccCheckDBInstanceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AliyunClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_db_instance" {
			continue
		}

		ins, err := client.DescribeDBInstanceById(rs.Primary.ID)

		if ins != nil {
			return fmt.Errorf("Error DB Instance still exist")
		}

		// Verify the error is what we want
		if err != nil {
			// Verify the error is what we want
			e, _ := err.(*common.Error)
			if e.ErrorResponse.Code == InstanceNotfound {
				continue
			}
			return err
		}
	}

	return nil
}

const testAccDBInstanceConfig = `
resource "alicloud_db_instance" "foo" {
	engine = "MySQL"
	engine_version = "5.6"
	db_instance_class = "rds.mysql.t1.small"
	db_instance_storage = "10"
	instance_charge_type = "Postpaid"
	db_instance_net_type = "Intranet"
}
`

const testAccDBInstance_vpc = `
data "alicloud_zones" "default" {
	"available_resource_creation"= "VSwitch"
}

resource "alicloud_vpc" "foo" {
	name = "tf_test_foo"
	cidr_block = "172.16.0.0/12"
}

resource "alicloud_vswitch" "foo" {
 	vpc_id = "${alicloud_vpc.foo.id}"
 	cidr_block = "172.16.0.0/21"
 	availability_zone = "${data.alicloud_zones.default.zones.0.id}"
}

resource "alicloud_db_instance" "foo" {
	engine = "MySQL"
	engine_version = "5.6"
	db_instance_class = "rds.mysql.t1.small"
	db_instance_storage = "10"
	instance_charge_type = "Postpaid"
	db_instance_net_type = "Intranet"

	vswitch_id = "${alicloud_vswitch.foo.id}"
}
`
const testAccDBInstance_multiIZ = `
resource "alicloud_db_instance" "foo" {
	engine = "MySQL"
	engine_version = "5.6"
	db_instance_class = "rds.mysql.t1.small"
	db_instance_storage = "10"
	db_instance_net_type = "Intranet"
	multi_az = true
}
`

const testAccDBInstance_prepaid_order = `
resource "alicloud_db_instance" "foo" {
	engine = "MySQL"
	engine_version = "5.6"
	db_instance_class = "rds.mysql.t1.small"
	db_instance_storage = "10"
	instance_charge_type = "Prepaid"
	db_instance_net_type = "Intranet"
}
`

const testAccDBInstance_database = `
resource "alicloud_db_instance" "foo" {
	engine = "MySQL"
	engine_version = "5.6"
	db_instance_class = "rds.mysql.t1.small"
	db_instance_storage = "10"
	instance_charge_type = "Postpaid"
	db_instance_net_type = "Intranet"

	db_mappings = [
	    {
	      "db_name" = "foo"
	      "character_set_name" = "utf8"
	      "db_description" = "tf"
	    },{
	      "db_name" = "bar"
	      "character_set_name" = "utf8"
	      "db_description" = "tf"
	    }]
}
`
const testAccDBInstance_database_update = `
resource "alicloud_db_instance" "foo" {
	engine = "MySQL"
	engine_version = "5.6"
	db_instance_class = "rds.mysql.t1.small"
	db_instance_storage = "10"
	instance_charge_type = "Postpaid"
	db_instance_net_type = "Intranet"

	db_mappings = [
	    {
	      "db_name" = "foo"
	      "character_set_name" = "utf8"
	      "db_description" = "tf"
	    },{
	      "db_name" = "bar"
	      "character_set_name" = "utf8"
	      "db_description" = "tf"
	    },{
	      "db_name" = "zzz"
	      "character_set_name" = "utf8"
	      "db_description" = "tf"
	    }]
}
`

const testAccDBInstance_grantDatabasePrivilege2Account = `
resource "alicloud_db_instance" "foo" {
	engine = "MySQL"
	engine_version = "5.6"
	db_instance_class = "rds.mysql.t1.small"
	db_instance_storage = "10"
	instance_charge_type = "Postpaid"
	db_instance_net_type = "Intranet"

	master_user_name = "tester"
	master_user_password = "Test12345"

	db_mappings = [
	    {
	      "db_name" = "foo"
	      "character_set_name" = "utf8"
	      "db_description" = "tf"
	    },{
	      "db_name" = "bar"
	      "character_set_name" = "utf8"
	      "db_description" = "tf"
	    }]
}
`

const testAccDBInstance_allocatePublicConnection = `
resource "alicloud_db_instance" "foo" {
	engine = "MySQL"
	engine_version = "5.6"
	db_instance_class = "rds.mysql.t1.small"
	db_instance_storage = "10"
	instance_charge_type = "Postpaid"
	db_instance_net_type = "Intranet"

	master_user_name = "tester"
	master_user_password = "Test12345"

	allocate_public_connection = true
}
`

const testAccDBInstance_backup = `
resource "alicloud_db_instance" "foo" {
	engine = "MySQL"
	engine_version = "5.6"
	db_instance_class = "rds.mysql.t1.small"
	db_instance_storage = "10"
	instance_charge_type = "Postpaid"
	db_instance_net_type = "Intranet"

	preferred_backup_period = ["Wednesday","Thursday"]
	preferred_backup_time = "00:00Z-01:00Z"
	backup_retention_period = 9
}
`

const testAccDBInstance_securityIps = `
resource "alicloud_db_instance" "foo" {
	engine = "MySQL"
	engine_version = "5.6"
	db_instance_class = "rds.mysql.t1.small"
	db_instance_storage = "10"
	instance_charge_type = "Postpaid"
	db_instance_net_type = "Intranet"
}
`
const testAccDBInstance_securityIpsConfig = `
resource "alicloud_db_instance" "foo" {
	engine = "MySQL"
	engine_version = "5.6"
	db_instance_class = "rds.mysql.t1.small"
	db_instance_storage = "10"
	instance_charge_type = "Postpaid"
	db_instance_net_type = "Intranet"

	security_ips = ["10.168.1.12", "100.69.7.112"]
}
`

const testAccDBInstance_class = `
resource "alicloud_db_instance" "foo" {
	engine = "MySQL"
	engine_version = "5.6"
	db_instance_class = "rds.mysql.t1.small"
	db_instance_storage = "10"
	db_instance_net_type = "Intranet"
}
`
const testAccDBInstance_classUpgrade = `
resource "alicloud_db_instance" "foo" {
	engine = "MySQL"
	engine_version = "5.6"
	db_instance_class = "rds.mysql.s1.small"
	db_instance_storage = "10"
	db_instance_net_type = "Intranet"
}
`
