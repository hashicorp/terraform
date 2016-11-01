package aws

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
)

func TestAccAWSRDSClusterInstance_basic(t *testing.T) {
	var v rds.DBInstance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSClusterInstanceConfig(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSClusterInstanceExists("aws_rds_cluster_instance.cluster_instances", &v),
					testAccCheckAWSDBClusterInstanceAttributes(&v),
				),
			},
		},
	})
}

func TestAccAWSRDSClusterInstance_kmsKey(t *testing.T) {
	var v rds.DBInstance
	keyRegex := regexp.MustCompile("^arn:aws:kms:")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSClusterInstanceConfigKmsKey(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSClusterInstanceExists("aws_rds_cluster_instance.cluster_instances", &v),
					resource.TestMatchResourceAttr(
						"aws_rds_cluster_instance.cluster_instances", "kms_key_id", keyRegex),
				),
			},
		},
	})
}

// https://github.com/hashicorp/terraform/issues/5350
func TestAccAWSRDSClusterInstance_disappears(t *testing.T) {
	var v rds.DBInstance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSClusterInstanceConfig(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSClusterInstanceExists("aws_rds_cluster_instance.cluster_instances", &v),
					testAccAWSClusterInstanceDisappears(&v),
				),
				// A non-empty plan is what we want. A crash is what we don't want. :)
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAWSClusterInstanceDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_rds_cluster" {
			continue
		}

		// Try to find the Group
		conn := testAccProvider.Meta().(*AWSClient).rdsconn
		var err error
		resp, err := conn.DescribeDBInstances(
			&rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if len(resp.DBInstances) != 0 &&
				*resp.DBInstances[0].DBInstanceIdentifier == rs.Primary.ID {
				return fmt.Errorf("DB Cluster Instance %s still exists", rs.Primary.ID)
			}
		}

		// Return nil if the Cluster Instance is already destroyed
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "DBInstanceNotFound" {
				return nil
			}
		}

		return err

	}

	return nil
}

func testAccCheckAWSDBClusterInstanceAttributes(v *rds.DBInstance) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if *v.Engine != "aurora" {
			return fmt.Errorf("bad engine, expected \"aurora\": %#v", *v.Engine)
		}

		if !strings.HasPrefix(*v.DBClusterIdentifier, "tf-aurora-cluster") {
			return fmt.Errorf("Bad Cluster Identifier prefix:\nexpected: %s\ngot: %s", "tf-aurora-cluster", *v.DBClusterIdentifier)
		}

		return nil
	}
}

func testAccAWSClusterInstanceDisappears(v *rds.DBInstance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).rdsconn
		opts := &rds.DeleteDBInstanceInput{
			DBInstanceIdentifier: v.DBInstanceIdentifier,
		}
		if _, err := conn.DeleteDBInstance(opts); err != nil {
			return err
		}
		return resource.Retry(40*time.Minute, func() *resource.RetryError {
			opts := &rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: v.DBInstanceIdentifier,
			}
			_, err := conn.DescribeDBInstances(opts)
			if err != nil {
				dbinstanceerr, ok := err.(awserr.Error)
				if ok && dbinstanceerr.Code() == "DBInstanceNotFound" {
					return nil
				}
				return resource.NonRetryableError(
					fmt.Errorf("Error retrieving DB Instances: %s", err))
			}
			return resource.RetryableError(fmt.Errorf(
				"Waiting for instance to be deleted: %v", v.DBInstanceIdentifier))
		})
	}
}

func testAccCheckAWSClusterInstanceExists(n string, v *rds.DBInstance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DB Instance ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).rdsconn
		resp, err := conn.DescribeDBInstances(&rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		for _, d := range resp.DBInstances {
			if *d.DBInstanceIdentifier == rs.Primary.ID {
				*v = *d
				return nil
			}
		}

		return fmt.Errorf("DB Cluster (%s) not found", rs.Primary.ID)
	}
}

func TestAccAWSRDSClusterInstance_withInstanceEnhancedMonitor(t *testing.T) {
	var v rds.DBInstance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSClusterInstanceEnhancedMonitor(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSClusterInstanceExists("aws_rds_cluster_instance.cluster_instances", &v),
					testAccCheckAWSDBClusterInstanceAttributes(&v),
				),
			},
		},
	})
}

// Add some random to the name, to avoid collision
func testAccAWSClusterInstanceConfig(n int) string {
	return fmt.Sprintf(`
resource "aws_rds_cluster" "default" {
  cluster_identifier = "tf-aurora-cluster-test-%d"
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]
  database_name      = "mydb"
  master_username    = "foo"
  master_password    = "mustbeeightcharaters"
}

resource "aws_rds_cluster_instance" "cluster_instances" {
  identifier              = "tf-cluster-instance-%d"
  cluster_identifier      = "${aws_rds_cluster.default.id}"
  instance_class          = "db.r3.large"
  db_parameter_group_name = "${aws_db_parameter_group.bar.name}"
  promotion_tier          = "3"
}

resource "aws_db_parameter_group" "bar" {
  name   = "tfcluster-test-group-%d"
  family = "aurora5.6"

  parameter {
    name         = "back_log"
    value        = "32767"
    apply_method = "pending-reboot"
  }

  tags {
    foo = "bar"
  }
}
`, n, n, n)
}

func testAccAWSClusterInstanceConfigKmsKey(n int) string {
	return fmt.Sprintf(`

resource "aws_kms_key" "foo" {
    description = "Terraform acc test %d"
    policy = <<POLICY
{
  "Version": "2012-10-17",
  "Id": "kms-tf-1",
  "Statement": [
    {
      "Sid": "Enable IAM User Permissions",
      "Effect": "Allow",
      "Principal": {
        "AWS": "*"
      },
      "Action": "kms:*",
      "Resource": "*"
    }
  ]
}
POLICY
}

resource "aws_rds_cluster" "default" {
  cluster_identifier = "tf-aurora-cluster-test-%d"
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]
  database_name      = "mydb"
  master_username    = "foo"
  master_password    = "mustbeeightcharaters"
  storage_encrypted = true
  kms_key_id = "${aws_kms_key.foo.arn}"
}

resource "aws_rds_cluster_instance" "cluster_instances" {
  identifier              = "tf-cluster-instance-%d"
  cluster_identifier      = "${aws_rds_cluster.default.id}"
  instance_class          = "db.r3.large"
  db_parameter_group_name = "${aws_db_parameter_group.bar.name}"
}

resource "aws_db_parameter_group" "bar" {
  name   = "tfcluster-test-group-%d"
  family = "aurora5.6"

  parameter {
    name         = "back_log"
    value        = "32767"
    apply_method = "pending-reboot"
  }

  tags {
    foo = "bar"
  }
}
`, n, n, n, n)
}

func testAccAWSClusterInstanceEnhancedMonitor(n int) string {
	return fmt.Sprintf(`
resource "aws_rds_cluster" "default" {
  cluster_identifier = "tf-aurora-cluster-test-%d"
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]
  database_name      = "mydb"
  master_username    = "foo"
  master_password    = "mustbeeightcharaters"
}

resource "aws_rds_cluster_instance" "cluster_instances" {
  identifier              = "tf-cluster-instance-%d"
  cluster_identifier      = "${aws_rds_cluster.default.id}"
  instance_class          = "db.r3.large"
  db_parameter_group_name = "${aws_db_parameter_group.bar.name}"
  monitoring_interval     = "60"
  monitoring_role_arn     = "${aws_iam_role.tf_enhanced_monitor_role.arn}"
}

resource "aws_iam_role" "tf_enhanced_monitor_role" {
    name = "tf_enhanced_monitor_role-%d"
    assume_role_policy = <<EOF
{
            "Version": "2012-10-17",
            "Statement": [
                {
                    "Action": "sts:AssumeRole",
                    "Principal": {
                        "Service": "monitoring.rds.amazonaws.com"
                    },
                    "Effect": "Allow",
                    "Sid": ""
                }
            ]
   }
EOF
}

resource "aws_iam_policy_attachment" "rds_m_attach" {
    name = "AmazonRDSEnhancedMonitoringRole"
    roles = ["${aws_iam_role.tf_enhanced_monitor_role.name}"]
    policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonRDSEnhancedMonitoringRole"
}

resource "aws_db_parameter_group" "bar" {
  name   = "tfcluster-test-group-%d"
  family = "aurora5.6"

  parameter {
    name         = "back_log"
    value        = "32767"
    apply_method = "pending-reboot"
  }

  tags {
    foo = "bar"
  }
}
`, n, n, n, n)
}
