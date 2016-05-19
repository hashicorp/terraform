package aws

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSRedshiftCluster_basic(t *testing.T) {
	var v redshift.Cluster

	ri := rand.New(rand.NewSource(time.Now().UnixNano())).Int()
	config := fmt.Sprintf(testAccAWSRedshiftClusterConfig_basic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
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

func TestAccAWSRedshiftCluster_publiclyAccessible(t *testing.T) {
	var v redshift.Cluster

	ri := rand.New(rand.NewSource(time.Now().UnixNano())).Int()
	preConfig := fmt.Sprintf(testAccAWSRedshiftClusterConfig_notPubliclyAccessible, ri)
	postConfig := fmt.Sprintf(testAccAWSRedshiftClusterConfig_updatePubliclyAccessible, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "publicly_accessible", "false"),
				),
			},

			resource.TestStep{
				Config: postConfig,
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
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftClusterExists("aws_redshift_cluster.default", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_cluster.default", "number_of_nodes", "1"),
				),
			},

			resource.TestStep{
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

func TestResourceAWSRedshiftClusterDbNameValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "tEsting",
			ErrCount: 1,
		},
		{
			Value:    "testing1",
			ErrCount: 1,
		},
		{
			Value:    "testing-",
			ErrCount: 1,
		},
		{
			Value:    "",
			ErrCount: 2,
		},
		{
			Value:    randomString(65),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateRedshiftClusterDbName(tc.Value, "aws_redshift_cluster_database_name")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Redshift Cluster database_name to trigger a validation error")
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

var testAccAWSRedshiftClusterConfig_updateNodeCount = `
provider "aws" {
	region = "us-west-2"
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
  number_of_nodes = 2
}
`

var testAccAWSRedshiftClusterConfig_basic = `
provider "aws" {
	region = "us-west-2"
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
}`

var testAccAWSRedshiftClusterConfig_notPubliclyAccessible = `
provider "aws" {
	region = "us-west-2"
}

resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
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
	name = "foo"
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
}`

var testAccAWSRedshiftClusterConfig_updatePubliclyAccessible = `
provider "aws" {
	region = "us-west-2"
}

resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
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
	name = "foo"
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
}`
