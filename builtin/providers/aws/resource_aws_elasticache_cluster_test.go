package aws

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSElasticacheCluster_basic(t *testing.T) {
	var ec elasticache.CacheCluster
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSElasticacheClusterConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheSecurityGroupExists("aws_elasticache_security_group.bar"),
					testAccCheckAWSElasticacheClusterExists("aws_elasticache_cluster.bar", &ec),
					resource.TestCheckResourceAttr(
						"aws_elasticache_cluster.bar", "cache_nodes.0.id", "0001"),
				),
			},
		},
	})
}

func TestAccAWSElasticacheCluster_snapshotsWithUpdates(t *testing.T) {
	var ec elasticache.CacheCluster

	ri := genRandInt()
	preConfig := fmt.Sprintf(testAccAWSElasticacheClusterConfig_snapshots, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAWSElasticacheClusterConfig_snapshotsUpdated, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheSecurityGroupExists("aws_elasticache_security_group.bar"),
					testAccCheckAWSElasticacheClusterExists("aws_elasticache_cluster.bar", &ec),
					resource.TestCheckResourceAttr(
						"aws_elasticache_cluster.bar", "snapshot_window", "05:00-09:00"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_cluster.bar", "snapshot_retention_limit", "3"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheSecurityGroupExists("aws_elasticache_security_group.bar"),
					testAccCheckAWSElasticacheClusterExists("aws_elasticache_cluster.bar", &ec),
					resource.TestCheckResourceAttr(
						"aws_elasticache_cluster.bar", "snapshot_window", "07:00-09:00"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_cluster.bar", "snapshot_retention_limit", "7"),
				),
			},
		},
	})
}

func TestAccAWSElasticacheCluster_decreasingCacheNodes(t *testing.T) {
	var ec elasticache.CacheCluster

	ri := genRandInt()
	preConfig := fmt.Sprintf(testAccAWSElasticacheClusterConfigDecreasingNodes, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAWSElasticacheClusterConfigDecreasingNodes_update, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheSecurityGroupExists("aws_elasticache_security_group.bar"),
					testAccCheckAWSElasticacheClusterExists("aws_elasticache_cluster.bar", &ec),
					resource.TestCheckResourceAttr(
						"aws_elasticache_cluster.bar", "num_cache_nodes", "3"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheSecurityGroupExists("aws_elasticache_security_group.bar"),
					testAccCheckAWSElasticacheClusterExists("aws_elasticache_cluster.bar", &ec),
					resource.TestCheckResourceAttr(
						"aws_elasticache_cluster.bar", "num_cache_nodes", "1"),
				),
			},
		},
	})
}

func TestAccAWSElasticacheCluster_vpc(t *testing.T) {
	var csg elasticache.CacheSubnetGroup
	var ec elasticache.CacheCluster
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSElasticacheClusterInVPCConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheSubnetGroupExists("aws_elasticache_subnet_group.bar", &csg),
					testAccCheckAWSElasticacheClusterExists("aws_elasticache_cluster.bar", &ec),
					testAccCheckAWSElasticacheClusterAttributes(&ec),
				),
			},
		},
	})
}

func testAccCheckAWSElasticacheClusterAttributes(v *elasticache.CacheCluster) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if v.NotificationConfiguration == nil {
			return fmt.Errorf("Expected NotificationConfiguration for ElastiCache Cluster (%s)", *v.CacheClusterId)
		}

		if strings.ToLower(*v.NotificationConfiguration.TopicStatus) != "active" {
			return fmt.Errorf("Expected NotificationConfiguration status to be 'active', got (%s)", *v.NotificationConfiguration.TopicStatus)
		}

		return nil
	}
}

func testAccCheckAWSElasticacheClusterDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elasticacheconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elasticache_cluster" {
			continue
		}
		res, err := conn.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{
			CacheClusterId: aws.String(rs.Primary.ID),
		})
		if err != nil {
			// Verify the error is what we want
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "CacheClusterNotFound" {
				continue
			}
			return err
		}
		if len(res.CacheClusters) > 0 {
			return fmt.Errorf("still exist.")
		}
	}
	return nil
}

func testAccCheckAWSElasticacheClusterExists(n string, v *elasticache.CacheCluster) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No cache cluster ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).elasticacheconn
		resp, err := conn.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{
			CacheClusterId: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return fmt.Errorf("Elasticache error: %v", err)
		}

		for _, c := range resp.CacheClusters {
			if *c.CacheClusterId == rs.Primary.ID {
				*v = *c
			}
		}

		return nil
	}
}

func genRandInt() int {
	return rand.New(rand.NewSource(time.Now().UnixNano())).Int() % 1000
}

var testAccAWSElasticacheClusterConfig = fmt.Sprintf(`
provider "aws" {
	region = "us-east-1"
}
resource "aws_security_group" "bar" {
    name = "tf-test-security-group-%03d"
    description = "tf-test-security-group-descr"
    ingress {
        from_port = -1
        to_port = -1
        protocol = "icmp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_elasticache_security_group" "bar" {
    name = "tf-test-security-group-%03d"
    description = "tf-test-security-group-descr"
    security_group_names = ["${aws_security_group.bar.name}"]
}

resource "aws_elasticache_cluster" "bar" {
    cluster_id = "tf-test-%03d"
    engine = "memcached"
    node_type = "cache.m1.small"
    num_cache_nodes = 1
    port = 11211
    parameter_group_name = "default.memcached1.4"
    security_group_names = ["${aws_elasticache_security_group.bar.name}"]
}
`, genRandInt(), genRandInt(), genRandInt())

var testAccAWSElasticacheClusterConfig_snapshots = `
provider "aws" {
	region = "us-east-1"
}
resource "aws_security_group" "bar" {
    name = "tf-test-security-group-%03d"
    description = "tf-test-security-group-descr"
    ingress {
        from_port = -1
        to_port = -1
        protocol = "icmp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_elasticache_security_group" "bar" {
    name = "tf-test-security-group-%03d"
    description = "tf-test-security-group-descr"
    security_group_names = ["${aws_security_group.bar.name}"]
}

resource "aws_elasticache_cluster" "bar" {
    cluster_id = "tf-test-%03d"
    engine = "redis"
    node_type = "cache.m1.small"
    num_cache_nodes = 1
    port = 6379
  	parameter_group_name = "default.redis2.8"
    security_group_names = ["${aws_elasticache_security_group.bar.name}"]
    snapshot_window = "05:00-09:00"
    snapshot_retention_limit = 3
}
`

var testAccAWSElasticacheClusterConfig_snapshotsUpdated = `
provider "aws" {
	region = "us-east-1"
}
resource "aws_security_group" "bar" {
    name = "tf-test-security-group-%03d"
    description = "tf-test-security-group-descr"
    ingress {
        from_port = -1
        to_port = -1
        protocol = "icmp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_elasticache_security_group" "bar" {
    name = "tf-test-security-group-%03d"
    description = "tf-test-security-group-descr"
    security_group_names = ["${aws_security_group.bar.name}"]
}

resource "aws_elasticache_cluster" "bar" {
    cluster_id = "tf-test-%03d"
    engine = "redis"
    node_type = "cache.m1.small"
    num_cache_nodes = 1
    port = 6379
  	parameter_group_name = "default.redis2.8"
    security_group_names = ["${aws_elasticache_security_group.bar.name}"]
    snapshot_window = "07:00-09:00"
    snapshot_retention_limit = 7
    apply_immediately = true
}
`

var testAccAWSElasticacheClusterConfigDecreasingNodes = `
provider "aws" {
	region = "us-east-1"
}
resource "aws_security_group" "bar" {
    name = "tf-test-security-group-%03d"
    description = "tf-test-security-group-descr"
    ingress {
        from_port = -1
        to_port = -1
        protocol = "icmp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_elasticache_security_group" "bar" {
    name = "tf-test-security-group-%03d"
    description = "tf-test-security-group-descr"
    security_group_names = ["${aws_security_group.bar.name}"]
}

resource "aws_elasticache_cluster" "bar" {
    cluster_id = "tf-test-%03d"
    engine = "memcached"
    node_type = "cache.m1.small"
    num_cache_nodes = 3
    port = 11211
    parameter_group_name = "default.memcached1.4"
    security_group_names = ["${aws_elasticache_security_group.bar.name}"]
}
`

var testAccAWSElasticacheClusterConfigDecreasingNodes_update = `
provider "aws" {
	region = "us-east-1"
}
resource "aws_security_group" "bar" {
    name = "tf-test-security-group-%03d"
    description = "tf-test-security-group-descr"
    ingress {
        from_port = -1
        to_port = -1
        protocol = "icmp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_elasticache_security_group" "bar" {
    name = "tf-test-security-group-%03d"
    description = "tf-test-security-group-descr"
    security_group_names = ["${aws_security_group.bar.name}"]
}

resource "aws_elasticache_cluster" "bar" {
    cluster_id = "tf-test-%03d"
    engine = "memcached"
    node_type = "cache.m1.small"
    num_cache_nodes = 1
    port = 11211
    parameter_group_name = "default.memcached1.4"
    security_group_names = ["${aws_elasticache_security_group.bar.name}"]
    apply_immediately = true
}
`

var testAccAWSElasticacheClusterInVPCConfig = fmt.Sprintf(`
resource "aws_vpc" "foo" {
    cidr_block = "192.168.0.0/16"
    tags {
            Name = "tf-test"
    }
}

resource "aws_subnet" "foo" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "192.168.0.0/20"
    availability_zone = "us-west-2a"
    tags {
            Name = "tf-test"
    }
}

resource "aws_elasticache_subnet_group" "bar" {
    name = "tf-test-cache-subnet-%03d"
    description = "tf-test-cache-subnet-group-descr"
    subnet_ids = ["${aws_subnet.foo.id}"]
}

resource "aws_security_group" "bar" {
    name = "tf-test-security-group-%03d"
    description = "tf-test-security-group-descr"
    vpc_id = "${aws_vpc.foo.id}"
    ingress {
        from_port = -1
        to_port = -1
        protocol = "icmp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_elasticache_cluster" "bar" {
    // Including uppercase letters in this name to ensure
    // that we correctly handle the fact that the API
    // normalizes names to lowercase.
    cluster_id = "tf-TEST-%03d"
    node_type = "cache.m1.small"
    num_cache_nodes = 1
    engine = "redis"
    engine_version = "2.8.19"
    port = 6379
    subnet_group_name = "${aws_elasticache_subnet_group.bar.name}"
    security_group_ids = ["${aws_security_group.bar.id}"]
    parameter_group_name = "default.redis2.8"
    notification_topic_arn      = "${aws_sns_topic.topic_example.arn}"
}

resource "aws_sns_topic" "topic_example" {
  name = "tf-ecache-cluster-test"
}
`, genRandInt(), genRandInt(), genRandInt())
