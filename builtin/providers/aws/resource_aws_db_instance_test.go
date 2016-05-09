package aws

import (
	"fmt"
	"log"
	"strings"

	"math/rand"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
)

func TestAccAWSDBInstance_basic(t *testing.T) {
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
						"aws_db_instance.bar", "allocated_storage", "10"),
					resource.TestCheckResourceAttr(
						"aws_db_instance.bar", "engine", "mysql"),
					resource.TestCheckResourceAttr(
						"aws_db_instance.bar", "license_model", "general-public-license"),
					resource.TestCheckResourceAttr(
						"aws_db_instance.bar", "instance_class", "db.t1.micro"),
					resource.TestCheckResourceAttr(
						"aws_db_instance.bar", "name", "baz"),
					resource.TestCheckResourceAttr(
						"aws_db_instance.bar", "username", "foo"),
					resource.TestCheckResourceAttr(
						"aws_db_instance.bar", "parameter_group_name", "default.mysql5.6"),
				),
			},
		},
	})
}

func TestAccAWSDBInstance_optionGroup(t *testing.T) {
	var v rds.DBInstance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDBInstanceConfigWithOptionGroup,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBInstanceExists("aws_db_instance.bar", &v),
					testAccCheckAWSDBInstanceAttributes(&v),
					resource.TestCheckResourceAttr(
						"aws_db_instance.bar", "option_group_name", "option-group-test-terraform"),
				),
			},
		},
	})
}

func TestAccAWSDBInstanceReplica(t *testing.T) {
	var s, r rds.DBInstance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccReplicaInstanceConfig(rand.New(rand.NewSource(time.Now().UnixNano())).Int()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBInstanceExists("aws_db_instance.bar", &s),
					testAccCheckAWSDBInstanceExists("aws_db_instance.replica", &r),
					testAccCheckAWSDBInstanceReplicaAttributes(&s, &r),
				),
			},
		},
	})
}

func TestAccAWSDBInstanceSnapshot(t *testing.T) {
	var snap rds.DBInstance

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		// testAccCheckAWSDBInstanceSnapshot verifies a database snapshot is
		// created, and subequently deletes it
		CheckDestroy: testAccCheckAWSDBInstanceSnapshot,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccSnapshotInstanceConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBInstanceExists("aws_db_instance.snapshot", &snap),
				),
			},
		},
	})
}

func TestAccAWSDBInstanceNoSnapshot(t *testing.T) {
	var nosnap rds.DBInstance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBInstanceNoSnapshot,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNoSnapshotInstanceConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBInstanceExists("aws_db_instance.no_snapshot", &nosnap),
				),
			},
		},
	})
}

func TestAccAWSDBInstance_enhancedMonitoring(t *testing.T) {
	var dbInstance rds.DBInstance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBInstanceNoSnapshot,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccSnapshotInstanceConfig_enhancedMonitoring,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBInstanceExists("aws_db_instance.enhanced_monitoring", &dbInstance),
					resource.TestCheckResourceAttr(
						"aws_db_instance.enhanced_monitoring", "monitoring_interval", "5"),
				),
			},
		},
	})
}

func testAccCheckAWSDBInstanceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).rdsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_db_instance" {
			continue
		}

		// Try to find the Group
		var err error
		resp, err := conn.DescribeDBInstances(
			&rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: aws.String(rs.Primary.ID),
			})

		if ae, ok := err.(awserr.Error); ok && ae.Code() == "DBInstanceNotFound" {
			continue
		}

		if err == nil {
			if len(resp.DBInstances) != 0 &&
				*resp.DBInstances[0].DBInstanceIdentifier == rs.Primary.ID {
				return fmt.Errorf("DB Instance still exists")
			}
		}

		// Verify the error
		newerr, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if newerr.Code() != "DBInstanceNotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSDBInstanceAttributes(v *rds.DBInstance) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if *v.Engine != "mysql" {
			return fmt.Errorf("bad engine: %#v", *v.Engine)
		}

		if *v.EngineVersion == "" {
			return fmt.Errorf("bad engine_version: %#v", *v.EngineVersion)
		}

		if *v.BackupRetentionPeriod != 0 {
			return fmt.Errorf("bad backup_retention_period: %#v", *v.BackupRetentionPeriod)
		}

		return nil
	}
}

func testAccCheckAWSDBInstanceReplicaAttributes(source, replica *rds.DBInstance) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if replica.ReadReplicaSourceDBInstanceIdentifier != nil && *replica.ReadReplicaSourceDBInstanceIdentifier != *source.DBInstanceIdentifier {
			return fmt.Errorf("bad source identifier for replica, expected: '%s', got: '%s'", *source.DBInstanceIdentifier, *replica.ReadReplicaSourceDBInstanceIdentifier)
		}

		return nil
	}
}

func testAccCheckAWSDBInstanceSnapshot(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).rdsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_db_instance" {
			continue
		}

		var err error
		resp, err := conn.DescribeDBInstances(
			&rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: aws.String(rs.Primary.ID),
			})

		if err != nil {
			newerr, _ := err.(awserr.Error)
			if newerr.Code() != "DBInstanceNotFound" {
				return err
			}

		} else {
			if len(resp.DBInstances) != 0 &&
				*resp.DBInstances[0].DBInstanceIdentifier == rs.Primary.ID {
				return fmt.Errorf("DB Instance still exists")
			}
		}

		log.Printf("[INFO] Trying to locate the DBInstance Final Snapshot")
		snapshot_identifier := "foobarbaz-test-terraform-final-snapshot-1"
		_, snapErr := conn.DescribeDBSnapshots(
			&rds.DescribeDBSnapshotsInput{
				DBSnapshotIdentifier: aws.String(snapshot_identifier),
			})

		if snapErr != nil {
			newerr, _ := snapErr.(awserr.Error)
			if newerr.Code() == "DBSnapshotNotFound" {
				return fmt.Errorf("Snapshot %s not found", snapshot_identifier)
			}
		} else { // snapshot was found
			// verify we have the tags copied to the snapshot
			instanceARN, err := buildRDSARN(snapshot_identifier, testAccProvider.Meta())
			// tags have a different ARN, just swapping :db: for :snapshot:
			tagsARN := strings.Replace(instanceARN, ":db:", ":snapshot:", 1)
			if err != nil {
				return fmt.Errorf("Error building ARN for tags check with ARN (%s): %s", tagsARN, err)
			}
			resp, err := conn.ListTagsForResource(&rds.ListTagsForResourceInput{
				ResourceName: aws.String(tagsARN),
			})
			if err != nil {
				return fmt.Errorf("Error retrieving tags for ARN (%s): %s", tagsARN, err)
			}

			if resp.TagList == nil || len(resp.TagList) == 0 {
				return fmt.Errorf("Tag list is nil or zero: %s", resp.TagList)
			}

			var found bool
			for _, t := range resp.TagList {
				if *t.Key == "Name" && *t.Value == "tf-tags-db" {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("Expected to find tag Name (%s), but wasn't found. Tags: %s", "tf-tags-db", resp.TagList)
			}
			// end tag search

			log.Printf("[INFO] Deleting the Snapshot %s", snapshot_identifier)
			_, snapDeleteErr := conn.DeleteDBSnapshot(
				&rds.DeleteDBSnapshotInput{
					DBSnapshotIdentifier: aws.String(snapshot_identifier),
				})
			if snapDeleteErr != nil {
				return err
			}
		} // end snapshot was found
	}

	return nil
}

func testAccCheckAWSDBInstanceNoSnapshot(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).rdsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_db_instance" {
			continue
		}

		var err error
		resp, err := conn.DescribeDBInstances(
			&rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: aws.String(rs.Primary.ID),
			})

		if err != nil {
			newerr, _ := err.(awserr.Error)
			if newerr.Code() != "DBInstanceNotFound" {
				return err
			}

		} else {
			if len(resp.DBInstances) != 0 &&
				*resp.DBInstances[0].DBInstanceIdentifier == rs.Primary.ID {
				return fmt.Errorf("DB Instance still exists")
			}
		}

		snapshot_identifier := "foobarbaz-test-terraform-final-snapshot-2"
		_, snapErr := conn.DescribeDBSnapshots(
			&rds.DescribeDBSnapshotsInput{
				DBSnapshotIdentifier: aws.String(snapshot_identifier),
			})

		if snapErr != nil {
			newerr, _ := snapErr.(awserr.Error)
			if newerr.Code() != "DBSnapshotNotFound" {
				return fmt.Errorf("Snapshot %s found and it shouldn't have been", snapshot_identifier)
			}
		}
	}

	return nil
}

func testAccCheckAWSDBInstanceExists(n string, v *rds.DBInstance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DB Instance ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).rdsconn

		opts := rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(rs.Primary.ID),
		}

		resp, err := conn.DescribeDBInstances(&opts)

		if err != nil {
			return err
		}

		if len(resp.DBInstances) != 1 ||
			*resp.DBInstances[0].DBInstanceIdentifier != rs.Primary.ID {
			return fmt.Errorf("DB Instance not found")
		}

		*v = *resp.DBInstances[0]

		return nil
	}
}

// Database names cannot collide, and deletion takes so long, that making the
// name a bit random helps so able we can kill a test that's just waiting for a
// delete and not be blocked on kicking off another one.
var testAccAWSDBInstanceConfig = `
resource "aws_db_instance" "bar" {
	allocated_storage = 10
	engine = "MySQL"
	engine_version = "5.6.21"
	instance_class = "db.t1.micro"
	name = "baz"
	password = "barbarbarbar"
	username = "foo"


	# Maintenance Window is stored in lower case in the API, though not strictly 
	# documented. Terraform will downcase this to match (as opposed to throw a 
	# validation error).
	maintenance_window = "Fri:09:00-Fri:09:30"

	backup_retention_period = 0

	parameter_group_name = "default.mysql5.6"
}`

var testAccAWSDBInstanceConfigWithOptionGroup = fmt.Sprintf(`

resource "aws_db_option_group" "bar" {
	name = "option-group-test-terraform"
	option_group_description = "Test option group for terraform"
	engine_name = "mysql"
	major_engine_version = "5.6"
}

resource "aws_db_instance" "bar" {
	identifier = "foobarbaz-test-terraform-%d"

	allocated_storage = 10
	engine = "MySQL"
	instance_class = "db.m1.small"
	name = "baz"
	password = "barbarbarbar"
	username = "foo"

	backup_retention_period = 0

	parameter_group_name = "default.mysql5.6"
	option_group_name = "${aws_db_option_group.bar.name}"
}`, acctest.RandInt())

func testAccReplicaInstanceConfig(val int) string {
	return fmt.Sprintf(`
	resource "aws_db_instance" "bar" {
		identifier = "foobarbaz-test-terraform-%d"

		allocated_storage = 5
		engine = "mysql"
		engine_version = "5.6.21"
		instance_class = "db.t1.micro"
		name = "baz"
		password = "barbarbarbar"
		username = "foo"

		backup_retention_period = 1

		parameter_group_name = "default.mysql5.6"
	}
	
	resource "aws_db_instance" "replica" {
		identifier = "tf-replica-db-%d"
		backup_retention_period = 0
		replicate_source_db = "${aws_db_instance.bar.identifier}"
		allocated_storage = "${aws_db_instance.bar.allocated_storage}"
		engine = "${aws_db_instance.bar.engine}"
		engine_version = "${aws_db_instance.bar.engine_version}"
		instance_class = "${aws_db_instance.bar.instance_class}"
		password = "${aws_db_instance.bar.password}"
		username = "${aws_db_instance.bar.username}"
		tags {
			Name = "tf-replica-db"
		}
	}
	`, val, val)
}

func testAccSnapshotInstanceConfig() string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}
resource "aws_db_instance" "snapshot" {
	identifier = "tf-snapshot-%d"

	allocated_storage = 5
	engine = "mysql"
	engine_version = "5.6.21"
	instance_class = "db.t1.micro"
	name = "baz"
	password = "barbarbarbar"
	username = "foo"
	security_group_names = ["default"]
	backup_retention_period = 1

	parameter_group_name = "default.mysql5.6"

	skip_final_snapshot = false
	copy_tags_to_snapshot = true
	final_snapshot_identifier = "foobarbaz-test-terraform-final-snapshot-1"
	tags {
		Name = "tf-tags-db"
	}
}`, acctest.RandInt())
}

func testAccNoSnapshotInstanceConfig() string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}
resource "aws_db_instance" "no_snapshot" {
	identifier = "tf-test-%s"

	allocated_storage = 5
	engine = "mysql"
	engine_version = "5.6.21"
	instance_class = "db.t1.micro"
	name = "baz"
	password = "barbarbarbar"
	username = "foo"
    security_group_names = ["default"]
	backup_retention_period = 1

	parameter_group_name = "default.mysql5.6"

	skip_final_snapshot = true
	final_snapshot_identifier = "foobarbaz-test-terraform-final-snapshot-2"
}
`, acctest.RandString(5))
}

var testAccSnapshotInstanceConfig_enhancedMonitoring = `
resource "aws_iam_role" "enhanced_policy_role" {
    name = "enhanced-monitoring-role"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "monitoring.rds.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF

}

resource "aws_iam_policy_attachment" "test-attach" {
    name = "enhanced-monitoring-attachment"
    roles = [
        "${aws_iam_role.enhanced_policy_role.name}",
    ]

    policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonRDSEnhancedMonitoringRole"
}

resource "aws_db_instance" "enhanced_monitoring" {
	identifier = "foobarbaz-test-terraform-enhanced-monitoring"
	depends_on = ["aws_iam_policy_attachment.test-attach"]

	allocated_storage = 5
	engine = "mysql"
	engine_version = "5.6.21"
	instance_class = "db.m3.medium"
	name = "baz"
	password = "barbarbarbar"
	username = "foo"
	backup_retention_period = 1

	parameter_group_name = "default.mysql5.6"

	monitoring_role_arn = "${aws_iam_role.enhanced_policy_role.arn}"
	monitoring_interval = "5"

	skip_final_snapshot = true
}
`
