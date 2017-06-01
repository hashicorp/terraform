package aws

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
)

func TestAccAWSRDSCluster_basic(t *testing.T) {
	var v rds.DBCluster

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSClusterConfig(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSClusterExists("aws_rds_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_rds_cluster.default", "storage_encrypted", "false"),
					resource.TestCheckResourceAttr(
						"aws_rds_cluster.default", "db_cluster_parameter_group_name", "default.aurora5.6"),
					resource.TestCheckResourceAttrSet(
						"aws_rds_cluster.default", "reader_endpoint"),
					resource.TestCheckResourceAttrSet(
						"aws_rds_cluster.default", "cluster_resource_id"),
				),
			},
		},
	})
}

func TestAccAWSRDSCluster_namePrefix(t *testing.T) {
	var v rds.DBCluster

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSClusterConfig_namePrefix(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSClusterExists("aws_rds_cluster.test", &v),
					resource.TestMatchResourceAttr(
						"aws_rds_cluster.test", "cluster_identifier", regexp.MustCompile("^tf-test-")),
				),
			},
		},
	})
}

func TestAccAWSRDSCluster_generatedName(t *testing.T) {
	var v rds.DBCluster

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSClusterConfig_generatedName(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSClusterExists("aws_rds_cluster.test", &v),
					resource.TestMatchResourceAttr(
						"aws_rds_cluster.test", "cluster_identifier", regexp.MustCompile("^tf-")),
				),
			},
		},
	})
}

func TestAccAWSRDSCluster_takeFinalSnapshot(t *testing.T) {
	var v rds.DBCluster
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSClusterSnapshot(rInt),
		Steps: []resource.TestStep{
			{
				Config: testAccAWSClusterConfigWithFinalSnapshot(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSClusterExists("aws_rds_cluster.default", &v),
				),
			},
		},
	})
}

/// This is a regression test to make sure that we always cover the scenario as hightlighted in
/// https://github.com/hashicorp/terraform/issues/11568
func TestAccAWSRDSCluster_missingUserNameCausesError(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAWSClusterConfigWithoutUserNameAndPassword(acctest.RandInt()),
				ExpectError: regexp.MustCompile(`required field is not set`),
			},
		},
	})
}

func TestAccAWSRDSCluster_updateTags(t *testing.T) {
	var v rds.DBCluster
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSClusterConfig(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSClusterExists("aws_rds_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_rds_cluster.default", "tags.%", "1"),
				),
			},
			{
				Config: testAccAWSClusterConfigUpdatedTags(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSClusterExists("aws_rds_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_rds_cluster.default", "tags.%", "2"),
				),
			},
		},
	})
}

func TestAccAWSRDSCluster_kmsKey(t *testing.T) {
	var v rds.DBCluster
	keyRegex := regexp.MustCompile("^arn:aws:kms:")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSClusterConfig_kmsKey(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSClusterExists("aws_rds_cluster.default", &v),
					resource.TestMatchResourceAttr(
						"aws_rds_cluster.default", "kms_key_id", keyRegex),
				),
			},
		},
	})
}

func TestAccAWSRDSCluster_encrypted(t *testing.T) {
	var v rds.DBCluster

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSClusterConfig_encrypted(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSClusterExists("aws_rds_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_rds_cluster.default", "storage_encrypted", "true"),
					resource.TestCheckResourceAttr(
						"aws_rds_cluster.default", "db_cluster_parameter_group_name", "default.aurora5.6"),
				),
			},
		},
	})
}

func TestAccAWSRDSCluster_backupsUpdate(t *testing.T) {
	var v rds.DBCluster

	ri := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSClusterConfig_backups(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSClusterExists("aws_rds_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_rds_cluster.default", "preferred_backup_window", "07:00-09:00"),
					resource.TestCheckResourceAttr(
						"aws_rds_cluster.default", "backup_retention_period", "5"),
					resource.TestCheckResourceAttr(
						"aws_rds_cluster.default", "preferred_maintenance_window", "tue:04:00-tue:04:30"),
				),
			},

			resource.TestStep{
				Config: testAccAWSClusterConfig_backupsUpdate(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSClusterExists("aws_rds_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_rds_cluster.default", "preferred_backup_window", "03:00-09:00"),
					resource.TestCheckResourceAttr(
						"aws_rds_cluster.default", "backup_retention_period", "10"),
					resource.TestCheckResourceAttr(
						"aws_rds_cluster.default", "preferred_maintenance_window", "wed:01:00-wed:01:30"),
				),
			},
		},
	})
}

func TestAccAWSRDSCluster_iamAuth(t *testing.T) {
	var v rds.DBCluster

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSClusterConfig_iamAuth(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSClusterExists("aws_rds_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_rds_cluster.default", "iam_database_authentication_enabled", "true"),
				),
			},
		},
	})
}

func testAccCheckAWSClusterDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_rds_cluster" {
			continue
		}

		// Try to find the Group
		conn := testAccProvider.Meta().(*AWSClient).rdsconn
		var err error
		resp, err := conn.DescribeDBClusters(
			&rds.DescribeDBClustersInput{
				DBClusterIdentifier: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if len(resp.DBClusters) != 0 &&
				*resp.DBClusters[0].DBClusterIdentifier == rs.Primary.ID {
				return fmt.Errorf("DB Cluster %s still exists", rs.Primary.ID)
			}
		}

		// Return nil if the cluster is already destroyed
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "DBClusterNotFoundFault" {
				return nil
			}
		}

		return err
	}

	return nil
}

func testAccCheckAWSClusterSnapshot(rInt int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_rds_cluster" {
				continue
			}

			// Try and delete the snapshot before we check for the cluster not found
			snapshot_identifier := fmt.Sprintf("tf-acctest-rdscluster-snapshot-%d", rInt)

			awsClient := testAccProvider.Meta().(*AWSClient)
			conn := awsClient.rdsconn

			arn, arnErr := buildRDSClusterARN(snapshot_identifier, awsClient.partition, awsClient.accountid, awsClient.region)
			tagsARN := strings.Replace(arn, ":cluster:", ":snapshot:", 1)
			if arnErr != nil {
				return fmt.Errorf("Error building ARN for tags check with ARN (%s): %s", tagsARN, arnErr)
			}

			log.Printf("[INFO] Deleting the Snapshot %s", snapshot_identifier)
			_, snapDeleteErr := conn.DeleteDBClusterSnapshot(
				&rds.DeleteDBClusterSnapshotInput{
					DBClusterSnapshotIdentifier: aws.String(snapshot_identifier),
				})
			if snapDeleteErr != nil {
				return snapDeleteErr
			}

			// Try to find the Group
			var err error
			resp, err := conn.DescribeDBClusters(
				&rds.DescribeDBClustersInput{
					DBClusterIdentifier: aws.String(rs.Primary.ID),
				})

			if err == nil {
				if len(resp.DBClusters) != 0 &&
					*resp.DBClusters[0].DBClusterIdentifier == rs.Primary.ID {
					return fmt.Errorf("DB Cluster %s still exists", rs.Primary.ID)
				}
			}

			// Return nil if the cluster is already destroyed
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "DBClusterNotFoundFault" {
					return nil
				}
			}

			return err
		}

		return nil
	}
}

func testAccCheckAWSClusterExists(n string, v *rds.DBCluster) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DB Instance ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).rdsconn
		resp, err := conn.DescribeDBClusters(&rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		for _, c := range resp.DBClusters {
			if *c.DBClusterIdentifier == rs.Primary.ID {
				*v = *c
				return nil
			}
		}

		return fmt.Errorf("DB Cluster (%s) not found", rs.Primary.ID)
	}
}

func testAccAWSClusterConfig(n int) string {
	return fmt.Sprintf(`
resource "aws_rds_cluster" "default" {
  cluster_identifier = "tf-aurora-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  database_name = "mydb"
  master_username = "foo"
  master_password = "mustbeeightcharaters"
  db_cluster_parameter_group_name = "default.aurora5.6"
  skip_final_snapshot = true
  tags {
    Environment = "production"
  }
}`, n)
}

func testAccAWSClusterConfig_namePrefix(n int) string {
	return fmt.Sprintf(`
resource "aws_rds_cluster" "test" {
  cluster_identifier_prefix = "tf-test-"
  master_username = "root"
  master_password = "password"
  db_subnet_group_name = "${aws_db_subnet_group.test.name}"
  skip_final_snapshot = true
}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"
	tags {
		Name = "testAccAWSClusterConfig_namePrefix"
	}
}

resource "aws_subnet" "a" {
  vpc_id = "${aws_vpc.test.id}"
  cidr_block = "10.0.0.0/24"
  availability_zone = "us-west-2a"
}

resource "aws_subnet" "b" {
  vpc_id = "${aws_vpc.test.id}"
  cidr_block = "10.0.1.0/24"
  availability_zone = "us-west-2b"
}

resource "aws_db_subnet_group" "test" {
  name = "tf-test-%d"
  subnet_ids = ["${aws_subnet.a.id}", "${aws_subnet.b.id}"]
}
`, n)
}

func testAccAWSClusterConfig_generatedName(n int) string {
	return fmt.Sprintf(`
resource "aws_rds_cluster" "test" {
  master_username = "root"
  master_password = "password"
  db_subnet_group_name = "${aws_db_subnet_group.test.name}"
  skip_final_snapshot = true
}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"
	tags {
		Name = "testAccAWSClusterConfig_generatedName"
	}
}

resource "aws_subnet" "a" {
  vpc_id = "${aws_vpc.test.id}"
  cidr_block = "10.0.0.0/24"
  availability_zone = "us-west-2a"
}

resource "aws_subnet" "b" {
  vpc_id = "${aws_vpc.test.id}"
  cidr_block = "10.0.1.0/24"
  availability_zone = "us-west-2b"
}

resource "aws_db_subnet_group" "test" {
  name = "tf-test-%d"
  subnet_ids = ["${aws_subnet.a.id}", "${aws_subnet.b.id}"]
}
`, n)
}

func testAccAWSClusterConfigWithFinalSnapshot(n int) string {
	return fmt.Sprintf(`
resource "aws_rds_cluster" "default" {
  cluster_identifier = "tf-aurora-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  database_name = "mydb"
  master_username = "foo"
  master_password = "mustbeeightcharaters"
  db_cluster_parameter_group_name = "default.aurora5.6"
  final_snapshot_identifier = "tf-acctest-rdscluster-snapshot-%d"
  tags {
    Environment = "production"
  }
}`, n, n)
}

func testAccAWSClusterConfigWithoutUserNameAndPassword(n int) string {
	return fmt.Sprintf(`
resource "aws_rds_cluster" "default" {
  cluster_identifier = "tf-aurora-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  database_name = "mydb"
  skip_final_snapshot = true
}`, n)
}

func testAccAWSClusterConfigUpdatedTags(n int) string {
	return fmt.Sprintf(`
resource "aws_rds_cluster" "default" {
  cluster_identifier = "tf-aurora-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  database_name = "mydb"
  master_username = "foo"
  master_password = "mustbeeightcharaters"
  db_cluster_parameter_group_name = "default.aurora5.6"
  skip_final_snapshot = true
  tags {
    Environment = "production"
    AnotherTag = "test"
  }
}`, n)
}

func testAccAWSClusterConfig_kmsKey(n int) string {
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
   cluster_identifier = "tf-aurora-cluster-%d"
   availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
   database_name = "mydb"
   master_username = "foo"
   master_password = "mustbeeightcharaters"
   db_cluster_parameter_group_name = "default.aurora5.6"
   storage_encrypted = true
   kms_key_id = "${aws_kms_key.foo.arn}"
   skip_final_snapshot = true
 }`, n, n)
}

func testAccAWSClusterConfig_encrypted(n int) string {
	return fmt.Sprintf(`
resource "aws_rds_cluster" "default" {
  cluster_identifier = "tf-aurora-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  database_name = "mydb"
  master_username = "foo"
  master_password = "mustbeeightcharaters"
  storage_encrypted = true
  skip_final_snapshot = true
}`, n)
}

func testAccAWSClusterConfig_backups(n int) string {
	return fmt.Sprintf(`
resource "aws_rds_cluster" "default" {
  cluster_identifier = "tf-aurora-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  database_name = "mydb"
  master_username = "foo"
  master_password = "mustbeeightcharaters"
  backup_retention_period = 5
  preferred_backup_window = "07:00-09:00"
  preferred_maintenance_window = "tue:04:00-tue:04:30"
  skip_final_snapshot = true
}`, n)
}

func testAccAWSClusterConfig_backupsUpdate(n int) string {
	return fmt.Sprintf(`
resource "aws_rds_cluster" "default" {
  cluster_identifier = "tf-aurora-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  database_name = "mydb"
  master_username = "foo"
  master_password = "mustbeeightcharaters"
  backup_retention_period = 10
  preferred_backup_window = "03:00-09:00"
  preferred_maintenance_window = "wed:01:00-wed:01:30"
  apply_immediately = true
  skip_final_snapshot = true
}`, n)
}

func testAccAWSClusterConfig_iamAuth(n int) string {
	return fmt.Sprintf(`
resource "aws_rds_cluster" "default" {
  cluster_identifier = "tf-aurora-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  database_name = "mydb"
  master_username = "foo"
  master_password = "mustbeeightcharaters"
  iam_database_authentication_enabled = true
  skip_final_snapshot = true
}`, n)
}
