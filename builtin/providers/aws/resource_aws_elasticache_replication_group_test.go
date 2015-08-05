package aws

import (
	"fmt"
	"testing"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEcacheReplicationGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcacheReplicationGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEcacheReplicationGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcacheReplicationGroupExists("aws_elasticache_replication_group.bar"),
				),
			},
		},
	})
}

func testAccCheckAWSEcacheReplicationGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elasticacheconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elasticache_replication_group" {
			continue
		}
		res, err := conn.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{
			ReplicationGroupID: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return err
		}
		if len(res.ReplicationGroups) > 0 {
			return fmt.Errorf("still exist.")
		}
	}
	return nil
}

func testAccCheckAWSEcacheReplicationGroupExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).elasticacheconn
		res, err := conn.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{
			ReplicationGroupID: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return fmt.Errorf("CacheReplicationGroup error: %v", err)
		}

		if len(res.ReplicationGroups) != 1 ||
			*res.ReplicationGroups[0].ReplicationGroupID != rs.Primary.ID {
			return fmt.Errorf("Replication group not found")
		}
		log.Printf("[DEBUG] Rep group found")
		return nil
	}
}

var testAccAWSEcacheReplicationGroupConfig = fmt.Sprintf(`
resource "aws_elasticache_replication_group" "bar" {
    replication_group_id = "tf-repgrp-%03d"
    cache_node_type = "cache.m1.small"
    num_cache_clusters = 2
    description = "tf-test-replication-group-descr"
}
`, genRandInt())
