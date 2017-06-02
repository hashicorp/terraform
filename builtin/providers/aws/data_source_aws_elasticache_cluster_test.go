package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSDataElasticacheCluster_basic(t *testing.T) {
	rInt := acctest.RandInt()
	rString := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSElastiCacheClusterConfigWithDataSource(rString, rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_elasticache_cluster.bar", "engine", "memcached"),
					resource.TestCheckResourceAttr("data.aws_elasticache_cluster.bar", "node_type", "cache.m1.small"),
					resource.TestCheckResourceAttr("data.aws_elasticache_cluster.bar", "port", "11211"),
					resource.TestCheckResourceAttr("data.aws_elasticache_cluster.bar", "num_cache_nodes", "1"),
					resource.TestCheckResourceAttrSet("data.aws_elasticache_cluster.bar", "configuration_endpoint"),
					resource.TestCheckResourceAttrSet("data.aws_elasticache_cluster.bar", "cluster_address"),
					resource.TestCheckResourceAttrSet("data.aws_elasticache_cluster.bar", "availability_zone"),
				),
			},
		},
	})
}

func testAccAWSElastiCacheClusterConfigWithDataSource(rString string, rInt int) string {
	return fmt.Sprintf(`
provider "aws" {
	region = "us-east-1"
}

resource "aws_security_group" "bar" {
    name = "tf-test-security-group-%d"
    description = "tf-test-security-group-descr"
    ingress {
        from_port = -1
        to_port = -1
        protocol = "icmp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_elasticache_security_group" "bar" {
    name = "tf-test-security-group-%d"
    description = "tf-test-security-group-descr"
    security_group_names = ["${aws_security_group.bar.name}"]
}

resource "aws_elasticache_cluster" "bar" {
    cluster_id = "tf-%s"
    engine = "memcached"
    node_type = "cache.m1.small"
    num_cache_nodes = 1
    port = 11211
    parameter_group_name = "default.memcached1.4"
    security_group_names = ["${aws_elasticache_security_group.bar.name}"]
}

data "aws_elasticache_cluster" "bar" {
	cluster_id = "${aws_elasticache_cluster.bar.cluster_id}"
}

`, rInt, rInt, rString)
}
