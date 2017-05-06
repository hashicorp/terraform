package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSElasticacheRedisCluster_vpc(t *testing.T) {
	var rg elasticache.ReplicationGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheReplicationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSElasticacheRedisClusterVPCConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheReplicationGroupExists("aws_elasticache_redis_cluster.bar", &rg),
					resource.TestCheckResourceAttr(
						"aws_elasticache_redis_cluster.bar", "number_cache_clusters", "4"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_redis_cluster.bar", "replicas_per_node_group", "1"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_redis_cluster.bar", "num_node_groups", "2"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_redis_cluster.bar", "port", "6379"),
				),
			},
		},
	})
}

var testAccAWSElasticacheRedisClusterVPCConfig = fmt.Sprintf(`
provider "aws" {
    region = "us-west-2"
}

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
        Name = "tf-test-%03d"
    }
}

resource "aws_subnet" "bar" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "192.168.16.0/20"
    availability_zone = "us-west-2b"
    tags {
        Name = "tf-test-%03d"
    }
}

resource "aws_elasticache_subnet_group" "bar" {
    name = "tf-test-cache-subnet-%03d"
    description = "tf-test-cache-subnet-group-descr"
    subnet_ids = [
        "${aws_subnet.foo.id}",
        "${aws_subnet.bar.id}"
    ]
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

resource "aws_elasticache_redis_cluster" "bar" {
    replication_group_id = "tf-%s"
    replication_group_description = "test description"
    node_type = "cache.t2.micro"
    port = 6379
    subnet_group_name = "${aws_elasticache_subnet_group.bar.name}"
    security_group_ids = ["${aws_security_group.bar.id}"]
    parameter_group_name = "default.redis3.2.cluster.on"
	replicas_per_node_group = 1
	num_node_groups = 2

}
`, acctest.RandInt(), acctest.RandInt(), acctest.RandInt(), acctest.RandInt(), acctest.RandString(10))
