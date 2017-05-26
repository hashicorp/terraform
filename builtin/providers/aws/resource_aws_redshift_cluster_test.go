package aws

import (
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestValidateRedshiftClusterDbName(t *testing.T) {
	validNames := []string{
		"testdbname",
		"test_dbname",
		"testdbname123",
		"testdbname$hashicorp",
		"_dbname",
	}
	for _, v := range validNames {
		_, errors := validateRedshiftClusterDbName(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Redshift DBName: %q", v, errors)
		}
	}

	invalidNames := []string{
		"!",
		"/",
		" ",
		":",
		";",
		"test name",
		"/slash-at-the-beginning",
		"slash-at-the-end/",
		"",
		randomString(100),
		"TestDBname",
	}
	for _, v := range invalidNames {
		_, errors := validateRedshiftClusterDbName(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid Redshift DBName", v)
		}
	}
}

func TestAccAWSRedshiftCluster_basic(t *testing.T) {
	var v redshift.Cluster

	ri := rand.New(rand.NewSource(time.Now().UnixNano())).Int()
	config := fmt.Sprintf(testAccAWSRedshiftClusterConfig_basic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "cluster_type", "single-node"),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "publicly_accessible", "true"),
				),
			},
		},
	})
}

func TestAccAWSRedshiftCluster_withFinalSnapshot(t *testing.T) {
	var v redshift.Cluster

	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftClusterSnapshot(rInt),
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRedshiftClusterConfigWithFinalSnapshot(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
				),
			},
		},
	})
}

func TestAccAWSRedshiftCluster_kmsKey(t *testing.T) {
	var v redshift.Cluster

	ri := rand.New(rand.NewSource(time.Now().UnixNano())).Int()
	config := fmt.Sprintf(testAccAWSRedshiftClusterConfig_kmsKey, ri, ri)
	keyRegex := regexp.MustCompile("^arn:aws:([a-zA-Z0-9\\-])+:([a-z]{2}-[a-z]+-\\d{1})?:(\\d{12})?:(.*)$")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "cluster_type", "single-node"),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "publicly_accessible", "true"),
					resource.TestMatchResourceAttr("aws_redshift_cluster.default", "kms_key_id", keyRegex),
				),
			},
		},
	})
}

func TestAccAWSRedshiftCluster_enhancedVpcRoutingEnabled(t *testing.T) {
	var v redshift.Cluster

	ri := rand.New(rand.NewSource(time.Now().UnixNano())).Int()
	preConfig := fmt.Sprintf(testAccAWSRedshiftClusterConfig_enhancedVpcRoutingEnabled, ri)
	postConfig := fmt.Sprintf(testAccAWSRedshiftClusterConfig_enhancedVpcRoutingDisabled, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "enhanced_vpc_routing", "true"),
				),
			},
			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "enhanced_vpc_routing", "false"),
				),
			},
		},
	})
}

func TestAccAWSRedshiftCluster_loggingEnabled(t *testing.T) {
	var v redshift.Cluster
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRedshiftClusterConfig_loggingEnabled(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "enable_logging", "true"),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "bucket_name", fmt.Sprintf("tf-redshift-logging-%d", rInt)),
				),
			},

			{
				Config: testAccAWSRedshiftClusterConfig_loggingDisabled(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "enable_logging", "false"),
				),
			},
		},
	})
}

func TestAccAWSRedshiftCluster_iamRoles(t *testing.T) {
	var v redshift.Cluster

	ri := rand.New(rand.NewSource(time.Now().UnixNano())).Int()
	preConfig := fmt.Sprintf(testAccAWSRedshiftClusterConfig_iamRoles, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAWSRedshiftClusterConfig_updateIamRoles, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "iam_roles.#", "2"),
				),
			},

			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "iam_roles.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSRedshiftCluster_publiclyAccessible(t *testing.T) {
	var v redshift.Cluster
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRedshiftClusterConfig_notPubliclyAccessible(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "publicly_accessible", "false"),
				),
			},

			{
				Config: testAccAWSRedshiftClusterConfig_updatePubliclyAccessible(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "publicly_accessible", "true"),
				),
			},
		},
	})
}

func TestAccAWSRedshiftCluster_updateNodeCount(t *testing.T) {
	var v redshift.Cluster

	ri := rand.New(rand.NewSource(time.Now().UnixNano())).Int()
	preConfig := fmt.Sprintf(testAccAWSRedshiftClusterConfig_basic, ri)
	postConfig := fmt.Sprintf(testAccAWSRedshiftClusterConfig_updateNodeCount, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "number_of_nodes", "1"),
				),
			},

			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "number_of_nodes", "2"),
				),
			},
		},
	})
}

func TestAccAWSRedshiftCluster_tags(t *testing.T) {
	var v redshift.Cluster

	ri := rand.New(rand.NewSource(time.Now().UnixNano())).Int()
	preConfig := fmt.Sprintf(testAccAWSRedshiftClusterConfig_tags, ri)
	postConfig := fmt.Sprintf(testAccAWSRedshiftClusterConfig_updatedTags, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "tags.%", "3"),
					resource.TestCheckResourceAttr("aws_redshift_cluster.default", "tags.environment", "Production"),
				),
			},

			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "tags.%", "1"),
					resource.TestCheckResourceAttr("aws_redshift_cluster.default", "tags.environment", "Production"),
				),
			},
		},
	})
}

func testAccCheckAWSRedshiftClusterDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_redshift_cluster" {
			continue
		}

		// Try to find the Group
		conn := testAccProvider.Meta().(*AWSClient).redshiftconn
		var err error
		resp, err := conn.DescribeClusters(
			&redshift.DescribeClustersInput{
				ClusterIdentifier: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if len(resp.Clusters) != 0 &&
				*resp.Clusters[0].ClusterIdentifier == rs.Primary.ID {
				return fmt.Errorf("Redshift Cluster %s still exists", rs.Primary.ID)
			}
		}

		// Return nil if the cluster is already destroyed
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ClusterNotFound" {
				return nil
			}
		}

		return err
	}

	return nil
}

func testAccCheckAWSRedshiftClusterSnapshot(rInt int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_redshift_cluster" {
				continue
			}

			var err error

			// Try and delete the snapshot before we check for the cluster not found
			conn := testAccProvider.Meta().(*AWSClient).redshiftconn

			snapshot_identifier := fmt.Sprintf("tf-acctest-snapshot-%d", rInt)
			arn, err := buildRedshiftARN(snapshot_identifier, testAccProvider.Meta().(*AWSClient).partition, testAccProvider.Meta().(*AWSClient).accountid, testAccProvider.Meta().(*AWSClient).region)
			tagsARN := strings.Replace(arn, ":cluster:", ":snapshot:", 1)
			if err != nil {
				return fmt.Errorf("Error building ARN for tags check with ARN (%s): %s", tagsARN, err)
			}

			log.Printf("[INFO] Deleting the Snapshot %s", snapshot_identifier)
			_, snapDeleteErr := conn.DeleteClusterSnapshot(
				&redshift.DeleteClusterSnapshotInput{
					SnapshotIdentifier: aws.String(snapshot_identifier),
				})
			if snapDeleteErr != nil {
				return err
			}

			//lastly check that the Cluster is missing
			resp, err := conn.DescribeClusters(
				&redshift.DescribeClustersInput{
					ClusterIdentifier: aws.String(rs.Primary.ID),
				})

			if err == nil {
				if len(resp.Clusters) != 0 &&
					*resp.Clusters[0].ClusterIdentifier == rs.Primary.ID {
					return fmt.Errorf("Redshift Cluster %s still exists", rs.Primary.ID)
				}
			}

			// Return nil if the cluster is already destroyed
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "ClusterNotFound" {
					return nil
				}

				return err
			}

		}

		return nil
	}
}

func testAccCheckAWSRedshiftClusterExists(n string, v *redshift.Cluster) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Redshift Cluster Instance ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).redshiftconn
		resp, err := conn.DescribeClusters(&redshift.DescribeClustersInput{
			ClusterIdentifier: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		for _, c := range resp.Clusters {
			if *c.ClusterIdentifier == rs.Primary.ID {
				*v = *c
				return nil
			}
		}

		return fmt.Errorf("Redshift Cluster (%s) not found", rs.Primary.ID)
	}
}

func TestResourceAWSRedshiftClusterIdentifierValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "tEsting",
			ErrCount: 1,
		},
		{
			Value:    "1testing",
			ErrCount: 1,
		},
		{
			Value:    "testing--123",
			ErrCount: 1,
		},
		{
			Value:    "testing!",
			ErrCount: 1,
		},
		{
			Value:    "testing-",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateRedshiftClusterIdentifier(tc.Value, "aws_redshift_cluster_identifier")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Redshift Cluster cluster_identifier to trigger a validation error")
		}
	}
}

func TestResourceAWSRedshiftClusterFinalSnapshotIdentifierValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "testing--123",
			ErrCount: 1,
		},
		{
			Value:    "testing-",
			ErrCount: 1,
		},
		{
			Value:    "Testingq123!",
			ErrCount: 1,
		},
		{
			Value:    randomString(256),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateRedshiftClusterFinalSnapshotIdentifier(tc.Value, "aws_redshift_cluster_final_snapshot_identifier")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Redshift Cluster final_snapshot_identifier to trigger a validation error")
		}
	}
}

func TestResourceAWSRedshiftClusterMasterUsernameValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "1Testing",
			ErrCount: 1,
		},
		{
			Value:    "Testing!!",
			ErrCount: 1,
		},
		{
			Value:    randomString(129),
			ErrCount: 1,
		},
		{
			Value:    "testing_testing123",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateRedshiftClusterMasterUsername(tc.Value, "aws_redshift_cluster_master_username")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Redshift Cluster master_username to trigger a validation error")
		}
	}
}

func TestResourceAWSRedshiftClusterMasterPasswordValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "1TESTING",
			ErrCount: 1,
		},
		{
			Value:    "1testing",
			ErrCount: 1,
		},
		{
			Value:    "TestTest",
			ErrCount: 1,
		},
		{
			Value:    "T3st",
			ErrCount: 1,
		},
		{
			Value:    "1Testing",
			ErrCount: 0,
		},
		{
			Value:    "1Testing@",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateRedshiftClusterMasterPassword(tc.Value, "aws_redshift_cluster_master_password")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Redshift Cluster master_password to trigger a validation error")
		}
	}
}

var testAccAWSRedshiftClusterConfig_updateNodeCount = `
resource "aws_redshift_cluster" "default" {
  cluster_identifier = "tf-redshift-cluster-%d"
  availability_zone = "us-west-2a"
  database_name = "mydb"
  master_username = "foo_test"
  master_password = "Mustbe8characters"
  node_type = "dc1.large"
  automated_snapshot_retention_period = 0
  allow_version_upgrade = false
  number_of_nodes = 2
  skip_final_snapshot = true
}
`

var testAccAWSRedshiftClusterConfig_basic = `
resource "aws_redshift_cluster" "default" {
  cluster_identifier = "tf-redshift-cluster-%d"
  availability_zone = "us-west-2a"
  database_name = "mydb"
  master_username = "foo_test"
  master_password = "Mustbe8characters"
  node_type = "dc1.large"
  automated_snapshot_retention_period = 0
  allow_version_upgrade = false
  skip_final_snapshot = true
}`

func testAccAWSRedshiftClusterConfigWithFinalSnapshot(rInt int) string {
	return fmt.Sprintf(`
resource "aws_redshift_cluster" "default" {
  cluster_identifier = "tf-redshift-cluster-%d"
  availability_zone = "us-west-2a"
  database_name = "mydb"
  master_username = "foo_test"
  master_password = "Mustbe8characters"
  node_type = "dc1.large"
  automated_snapshot_retention_period = 0
  allow_version_upgrade = false
  skip_final_snapshot = false
  final_snapshot_identifier = "tf-acctest-snapshot-%d"
}`, rInt, rInt)
}

var testAccAWSRedshiftClusterConfig_kmsKey = `
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

resource "aws_redshift_cluster" "default" {
  cluster_identifier = "tf-redshift-cluster-%d"
  availability_zone = "us-west-2a"
  database_name = "mydb"
  master_username = "foo_test"
  master_password = "Mustbe8characters"
  node_type = "dc1.large"
  automated_snapshot_retention_period = 0
  allow_version_upgrade = false
  kms_key_id = "${aws_kms_key.foo.arn}"
  encrypted = true
  skip_final_snapshot = true
}`

var testAccAWSRedshiftClusterConfig_enhancedVpcRoutingEnabled = `
resource "aws_redshift_cluster" "default" {
  cluster_identifier = "tf-redshift-cluster-%d"
  availability_zone = "us-west-2a"
  database_name = "mydb"
  master_username = "foo_test"
  master_password = "Mustbe8characters"
  node_type = "dc1.large"
  automated_snapshot_retention_period = 0
  allow_version_upgrade = false
  enhanced_vpc_routing = true
  skip_final_snapshot = true
}
`

var testAccAWSRedshiftClusterConfig_enhancedVpcRoutingDisabled = `
resource "aws_redshift_cluster" "default" {
  cluster_identifier = "tf-redshift-cluster-%d"
  availability_zone = "us-west-2a"
  database_name = "mydb"
  master_username = "foo_test"
  master_password = "Mustbe8characters"
  node_type = "dc1.large"
  automated_snapshot_retention_period = 0
  allow_version_upgrade = false
  enhanced_vpc_routing = false
  skip_final_snapshot = true
}
`

func testAccAWSRedshiftClusterConfig_loggingDisabled(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_redshift_cluster" "default" {
		cluster_identifier = "tf-redshift-cluster-%d"
		availability_zone = "us-west-2a"
		database_name = "mydb"
		master_username = "foo_test"
		master_password = "Mustbe8characters"
		node_type = "dc1.large"
		automated_snapshot_retention_period = 0
		allow_version_upgrade = false
		enable_logging = false
		skip_final_snapshot = true
	}`, rInt)
}

func testAccAWSRedshiftClusterConfig_loggingEnabled(rInt int) string {
	return fmt.Sprintf(`
 resource "aws_s3_bucket" "bucket" {
	 bucket = "tf-redshift-logging-%d"
	 force_destroy = true
	 policy = <<EOF
{
 "Version": "2008-10-17",
 "Statement": [
	 {
		 "Sid": "Stmt1376526643067",
		 "Effect": "Allow",
		 "Principal": {
			 "AWS": "arn:aws:iam::902366379725:user/logs"
		 },
		 "Action": "s3:PutObject",
		 "Resource": "arn:aws:s3:::tf-redshift-logging-%d/*"
	 },
	 {
		 "Sid": "Stmt137652664067",
		 "Effect": "Allow",
		 "Principal": {
			 "AWS": "arn:aws:iam::902366379725:user/logs"
		 },
		 "Action": "s3:GetBucketAcl",
		 "Resource": "arn:aws:s3:::tf-redshift-logging-%d"
	 }
 ]
}
EOF
 }


 resource "aws_redshift_cluster" "default" {
	 cluster_identifier = "tf-redshift-cluster-%d"
	 availability_zone = "us-west-2a"
	 database_name = "mydb"
	 master_username = "foo_test"
	 master_password = "Mustbe8characters"
	 node_type = "dc1.large"
	 automated_snapshot_retention_period = 0
	 allow_version_upgrade = false
	 enable_logging = true
	 bucket_name = "${aws_s3_bucket.bucket.bucket}"
	 skip_final_snapshot = true
 }`, rInt, rInt, rInt, rInt)
}

var testAccAWSRedshiftClusterConfig_tags = `
resource "aws_redshift_cluster" "default" {
  cluster_identifier = "tf-redshift-cluster-%d"
  availability_zone = "us-west-2a"
  database_name = "mydb"
  master_username = "foo"
  master_password = "Mustbe8characters"
  node_type = "dc1.large"
  automated_snapshot_retention_period = 7
  allow_version_upgrade = false
  skip_final_snapshot = true
  tags {
    environment = "Production"
    cluster = "reader"
    Type = "master"
  }
}`

var testAccAWSRedshiftClusterConfig_updatedTags = `
resource "aws_redshift_cluster" "default" {
  cluster_identifier = "tf-redshift-cluster-%d"
  availability_zone = "us-west-2a"
  database_name = "mydb"
  master_username = "foo"
  master_password = "Mustbe8characters"
  node_type = "dc1.large"
  automated_snapshot_retention_period = 7
  allow_version_upgrade = false
  skip_final_snapshot = true
  tags {
    environment = "Production"
  }
}`

func testAccAWSRedshiftClusterConfig_notPubliclyAccessible(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_vpc" "foo" {
		cidr_block = "10.1.0.0/16"
		tags {
			Name = "testAccAWSRedshiftClusterConfig_notPubliclyAccessible"
		}
	}
	resource "aws_internet_gateway" "foo" {
		vpc_id = "${aws_vpc.foo.id}"
		tags {
			foo = "bar"
		}
	}
	resource "aws_subnet" "foo" {
		cidr_block = "10.1.1.0/24"
		availability_zone = "us-west-2a"
		vpc_id = "${aws_vpc.foo.id}"
		tags {
			Name = "tf-dbsubnet-test-1"
		}
	}
	resource "aws_subnet" "bar" {
		cidr_block = "10.1.2.0/24"
		availability_zone = "us-west-2b"
		vpc_id = "${aws_vpc.foo.id}"
		tags {
			Name = "tf-dbsubnet-test-2"
		}
	}
	resource "aws_subnet" "foobar" {
		cidr_block = "10.1.3.0/24"
		availability_zone = "us-west-2c"
		vpc_id = "${aws_vpc.foo.id}"
		tags {
			Name = "tf-dbsubnet-test-3"
		}
	}
	resource "aws_redshift_subnet_group" "foo" {
		name = "foo-%d"
		description = "foo description"
		subnet_ids = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}", "${aws_subnet.foobar.id}"]
	}
	resource "aws_redshift_cluster" "default" {
		cluster_identifier = "tf-redshift-cluster-%d"
		availability_zone = "us-west-2a"
		database_name = "mydb"
		master_username = "foo"
		master_password = "Mustbe8characters"
		node_type = "dc1.large"
		automated_snapshot_retention_period = 0
		allow_version_upgrade = false
		cluster_subnet_group_name = "${aws_redshift_subnet_group.foo.name}"
		publicly_accessible = false
		skip_final_snapshot = true
	}`, rInt, rInt)
}

func testAccAWSRedshiftClusterConfig_updatePubliclyAccessible(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_vpc" "foo" {
		cidr_block = "10.1.0.0/16"
		tags {
			Name = "testAccAWSRedshiftClusterConfig_updatePubliclyAccessible"
		}
	}
	resource "aws_internet_gateway" "foo" {
		vpc_id = "${aws_vpc.foo.id}"
		tags {
			foo = "bar"
		}
	}
	resource "aws_subnet" "foo" {
		cidr_block = "10.1.1.0/24"
		availability_zone = "us-west-2a"
		vpc_id = "${aws_vpc.foo.id}"
		tags {
			Name = "tf-dbsubnet-test-1"
		}
	}
	resource "aws_subnet" "bar" {
		cidr_block = "10.1.2.0/24"
		availability_zone = "us-west-2b"
		vpc_id = "${aws_vpc.foo.id}"
		tags {
			Name = "tf-dbsubnet-test-2"
		}
	}
	resource "aws_subnet" "foobar" {
		cidr_block = "10.1.3.0/24"
		availability_zone = "us-west-2c"
		vpc_id = "${aws_vpc.foo.id}"
		tags {
			Name = "tf-dbsubnet-test-3"
		}
	}
	resource "aws_redshift_subnet_group" "foo" {
		name = "foo-%d"
		description = "foo description"
		subnet_ids = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}", "${aws_subnet.foobar.id}"]
	}
	resource "aws_redshift_cluster" "default" {
		cluster_identifier = "tf-redshift-cluster-%d"
		availability_zone = "us-west-2a"
		database_name = "mydb"
		master_username = "foo"
		master_password = "Mustbe8characters"
		node_type = "dc1.large"
		automated_snapshot_retention_period = 0
		allow_version_upgrade = false
		cluster_subnet_group_name = "${aws_redshift_subnet_group.foo.name}"
		publicly_accessible = true
		skip_final_snapshot = true
	}`, rInt, rInt)
}

var testAccAWSRedshiftClusterConfig_iamRoles = `
resource "aws_iam_role" "ec2-role" {
	name   = "test-role-ec2-%d"
	path = "/"
 	assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":[\"ec2.amazonaws.com\"]},\"Action\":[\"sts:AssumeRole\"]}]}"
}

resource "aws_iam_role" "lambda-role" {
 	name   = "test-role-lambda-%d"
 	path = "/"
 	assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":[\"lambda.amazonaws.com\"]},\"Action\":[\"sts:AssumeRole\"]}]}"
}

resource "aws_redshift_cluster" "default" {
   cluster_identifier = "tf-redshift-cluster-%d"
   availability_zone = "us-west-2a"
   database_name = "mydb"
   master_username = "foo_test"
   master_password = "Mustbe8characters"
   node_type = "dc1.large"
   automated_snapshot_retention_period = 0
   allow_version_upgrade = false
   iam_roles = ["${aws_iam_role.ec2-role.arn}", "${aws_iam_role.lambda-role.arn}"]
   skip_final_snapshot = true
}`

var testAccAWSRedshiftClusterConfig_updateIamRoles = `
resource "aws_iam_role" "ec2-role" {
 	name   = "test-role-ec2-%d"
 	path = "/"
 	assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":[\"ec2.amazonaws.com\"]},\"Action\":[\"sts:AssumeRole\"]}]}"
 }

 resource "aws_iam_role" "lambda-role" {
 	name   = "test-role-lambda-%d"
 	path = "/"
 	assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":[\"lambda.amazonaws.com\"]},\"Action\":[\"sts:AssumeRole\"]}]}"
 }

 resource "aws_redshift_cluster" "default" {
   cluster_identifier = "tf-redshift-cluster-%d"
   availability_zone = "us-west-2a"
   database_name = "mydb"
   master_username = "foo_test"
   master_password = "Mustbe8characters"
   node_type = "dc1.large"
   automated_snapshot_retention_period = 0
   allow_version_upgrade = false
   iam_roles = ["${aws_iam_role.ec2-role.arn}"]
   skip_final_snapshot = true
 }`
