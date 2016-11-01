package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSElasticacheReplicationGroup_basic(t *testing.T) {
	var rg elasticache.ReplicationGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheReplicationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSElasticacheReplicationGroupConfig(acctest.RandString(10)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheReplicationGroupExists("aws_elasticache_replication_group.bar", &rg),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "number_cache_clusters", "2"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "auto_minor_version_upgrade", "false"),
				),
			},
		},
	})
}

func TestAccAWSElasticacheReplicationGroup_updateDescription(t *testing.T) {
	var rg elasticache.ReplicationGroup
	rName := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheReplicationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSElasticacheReplicationGroupConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheReplicationGroupExists("aws_elasticache_replication_group.bar", &rg),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "number_cache_clusters", "2"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "replication_group_description", "test description"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "auto_minor_version_upgrade", "false"),
				),
			},

			resource.TestStep{
				Config: testAccAWSElasticacheReplicationGroupConfigUpdatedDescription(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheReplicationGroupExists("aws_elasticache_replication_group.bar", &rg),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "number_cache_clusters", "2"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "replication_group_description", "updated description"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "auto_minor_version_upgrade", "true"),
				),
			},
		},
	})
}

func TestAccAWSElasticacheReplicationGroup_updateNodeSize(t *testing.T) {
	var rg elasticache.ReplicationGroup
	rName := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheReplicationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSElasticacheReplicationGroupConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheReplicationGroupExists("aws_elasticache_replication_group.bar", &rg),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "number_cache_clusters", "2"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "node_type", "cache.m1.small"),
				),
			},

			resource.TestStep{
				Config: testAccAWSElasticacheReplicationGroupConfigUpdatedNodeSize(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheReplicationGroupExists("aws_elasticache_replication_group.bar", &rg),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "number_cache_clusters", "2"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "node_type", "cache.m1.medium"),
				),
			},
		},
	})
}

//This is a test to prove that we panic we get in https://github.com/hashicorp/terraform/issues/9097
func TestAccAWSElasticacheReplicationGroup_updateParameterGroup(t *testing.T) {
	var rg elasticache.ReplicationGroup
	rName := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheReplicationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSElasticacheReplicationGroupConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheReplicationGroupExists("aws_elasticache_replication_group.bar", &rg),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "parameter_group_name", "default.redis3.2"),
				),
			},

			resource.TestStep{
				Config: testAccAWSElasticacheReplicationGroupConfigUpdatedParameterGroup(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheReplicationGroupExists("aws_elasticache_replication_group.bar", &rg),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "parameter_group_name", "allkeys-lru"),
				),
			},
		},
	})
}

func TestAccAWSElasticacheReplicationGroup_vpc(t *testing.T) {
	var rg elasticache.ReplicationGroup
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheReplicationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSElasticacheReplicationGroupInVPCConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheReplicationGroupExists("aws_elasticache_replication_group.bar", &rg),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "number_cache_clusters", "1"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "auto_minor_version_upgrade", "false"),
				),
			},
		},
	})
}

func TestAccAWSElasticacheReplicationGroup_multiAzInVpc(t *testing.T) {
	var rg elasticache.ReplicationGroup
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheReplicationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSElasticacheReplicationGroupMultiAZInVPCConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheReplicationGroupExists("aws_elasticache_replication_group.bar", &rg),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "number_cache_clusters", "2"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "automatic_failover_enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "snapshot_window", "02:00-03:00"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "snapshot_retention_limit", "7"),
					resource.TestCheckResourceAttrSet(
						"aws_elasticache_replication_group.bar", "primary_endpoint_address"),
				),
			},
		},
	})
}

func TestAccAWSElasticacheReplicationGroup_redisClusterInVpc2(t *testing.T) {
	var rg elasticache.ReplicationGroup
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheReplicationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSElasticacheReplicationGroupRedisClusterInVPCConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheReplicationGroupExists("aws_elasticache_replication_group.bar", &rg),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "number_cache_clusters", "2"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "automatic_failover_enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "snapshot_window", "02:00-03:00"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_replication_group.bar", "snapshot_retention_limit", "7"),
					resource.TestCheckResourceAttrSet(
						"aws_elasticache_replication_group.bar", "configuration_endpoint_address"),
				),
			},
		},
	})
}

func TestResourceAWSElastiCacheReplicationGroupIdValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "tEsting",
			ErrCount: 0,
		},
		{
			Value:    "t.sting",
			ErrCount: 1,
		},
		{
			Value:    "t--sting",
			ErrCount: 1,
		},
		{
			Value:    "1testing",
			ErrCount: 1,
		},
		{
			Value:    "testing-",
			ErrCount: 1,
		},
		{
			Value:    randomString(17),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateAwsElastiCacheReplicationGroupId(tc.Value, "aws_elasticache_replication_group_replication_group_id")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the ElastiCache Replication Group Id to trigger a validation error")
		}
	}
}

func TestResourceAWSElastiCacheReplicationGroupEngineValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Redis",
			ErrCount: 0,
		},
		{
			Value:    "REDIS",
			ErrCount: 0,
		},
		{
			Value:    "memcached",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateAwsElastiCacheReplicationGroupEngine(tc.Value, "aws_elasticache_replication_group_engine")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the ElastiCache Replication Group Engine to trigger a validation error")
		}
	}
}

func testAccCheckAWSElasticacheReplicationGroupExists(n string, v *elasticache.ReplicationGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No replication group ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).elasticacheconn
		res, err := conn.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{
			ReplicationGroupId: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return fmt.Errorf("Elasticache error: %v", err)
		}

		for _, rg := range res.ReplicationGroups {
			if *rg.ReplicationGroupId == rs.Primary.ID {
				*v = *rg
			}
		}

		return nil
	}
}

func testAccCheckAWSElasticacheReplicationDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elasticacheconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elasticache_replication_group" {
			continue
		}
		res, err := conn.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{
			ReplicationGroupId: aws.String(rs.Primary.ID),
		})
		if err != nil {
			// Verify the error is what we want
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ReplicationGroupNotFoundFault" {
				continue
			}
			return err
		}
		if len(res.ReplicationGroups) > 0 {
			return fmt.Errorf("still exist.")
		}
	}
	return nil
}

func testAccAWSElasticacheReplicationGroupConfig(rName string) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}
resource "aws_security_group" "bar" {
    name = "tf-test-security-group-%s"
    description = "tf-test-security-group-descr"
    ingress {
        from_port = -1
        to_port = -1
        protocol = "icmp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_elasticache_security_group" "bar" {
    name = "tf-test-security-group-%s"
    description = "tf-test-security-group-descr"
    security_group_names = ["${aws_security_group.bar.name}"]
}

resource "aws_elasticache_replication_group" "bar" {
    replication_group_id = "tf-%s"
    replication_group_description = "test description"
    node_type = "cache.m1.small"
    number_cache_clusters = 2
    port = 6379
    parameter_group_name = "default.redis3.2"
    security_group_names = ["${aws_elasticache_security_group.bar.name}"]
    apply_immediately = true
    auto_minor_version_upgrade = false
}`, rName, rName, rName)
}

func testAccAWSElasticacheReplicationGroupConfigUpdatedParameterGroup(rName string) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}
resource "aws_security_group" "bar" {
    name = "tf-test-security-group-%s"
    description = "tf-test-security-group-descr"
    ingress {
        from_port = -1
        to_port = -1
        protocol = "icmp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_elasticache_security_group" "bar" {
    name = "tf-test-security-group-%s"
    description = "tf-test-security-group-descr"
    security_group_names = ["${aws_security_group.bar.name}"]
}

resource "aws_elasticache_parameter_group" "bar" {
    name = "allkeys-lru"
    family = "redis3.2"

    parameter {
        name = "maxmemory-policy"
        value = "allkeys-lru"
    }
}

resource "aws_elasticache_replication_group" "bar" {
    replication_group_id = "tf-%s"
    replication_group_description = "test description"
    node_type = "cache.m1.small"
    number_cache_clusters = 2
    port = 6379
    parameter_group_name = "${aws_elasticache_parameter_group.bar.name}"
    security_group_names = ["${aws_elasticache_security_group.bar.name}"]
    apply_immediately = true
}`, rName, rName, rName)
}

func testAccAWSElasticacheReplicationGroupConfigUpdatedDescription(rName string) string {
	return fmt.Sprintf(`
provider "aws" {
	region = "us-east-1"
}
resource "aws_security_group" "bar" {
    name = "tf-test-security-group-%s"
    description = "tf-test-security-group-descr"
    ingress {
        from_port = -1
        to_port = -1
        protocol = "icmp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_elasticache_security_group" "bar" {
    name = "tf-test-security-group-%s"
    description = "tf-test-security-group-descr"
    security_group_names = ["${aws_security_group.bar.name}"]
}

resource "aws_elasticache_replication_group" "bar" {
    replication_group_id = "tf-%s"
    replication_group_description = "updated description"
    node_type = "cache.m1.small"
    number_cache_clusters = 2
    port = 6379
    parameter_group_name = "default.redis3.2"
    security_group_names = ["${aws_elasticache_security_group.bar.name}"]
    apply_immediately = true
    auto_minor_version_upgrade = true
}`, rName, rName, rName)
}

func testAccAWSElasticacheReplicationGroupConfigUpdatedNodeSize(rName string) string {
	return fmt.Sprintf(`
provider "aws" {
	region = "us-east-1"
}
resource "aws_security_group" "bar" {
    name = "tf-test-security-group-%s"
    description = "tf-test-security-group-descr"
    ingress {
        from_port = -1
        to_port = -1
        protocol = "icmp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_elasticache_security_group" "bar" {
    name = "tf-test-security-group-%s"
    description = "tf-test-security-group-descr"
    security_group_names = ["${aws_security_group.bar.name}"]
}

resource "aws_elasticache_replication_group" "bar" {
    replication_group_id = "tf-%s"
    replication_group_description = "updated description"
    node_type = "cache.m1.medium"
    number_cache_clusters = 2
    port = 6379
    parameter_group_name = "default.redis3.2"
    security_group_names = ["${aws_elasticache_security_group.bar.name}"]
    apply_immediately = true
}`, rName, rName, rName)
}

var testAccAWSElasticacheReplicationGroupInVPCConfig = fmt.Sprintf(`
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

resource "aws_elasticache_replication_group" "bar" {
    replication_group_id = "tf-%s"
    replication_group_description = "test description"
    node_type = "cache.m1.small"
    number_cache_clusters = 1
    port = 6379
    subnet_group_name = "${aws_elasticache_subnet_group.bar.name}"
    security_group_ids = ["${aws_security_group.bar.id}"]
    parameter_group_name = "default.redis3.2"
    availability_zones = ["us-west-2a"]
    auto_minor_version_upgrade = false
}

`, acctest.RandInt(), acctest.RandInt(), acctest.RandString(10))

var testAccAWSElasticacheReplicationGroupMultiAZInVPCConfig = fmt.Sprintf(`
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

resource "aws_elasticache_replication_group" "bar" {
    replication_group_id = "tf-%s"
    replication_group_description = "test description"
    node_type = "cache.m1.small"
    number_cache_clusters = 2
    port = 6379
    subnet_group_name = "${aws_elasticache_subnet_group.bar.name}"
    security_group_ids = ["${aws_security_group.bar.id}"]
    parameter_group_name = "default.redis3.2"
    availability_zones = ["us-west-2a","us-west-2b"]
    automatic_failover_enabled = true
    snapshot_window = "02:00-03:00"
    snapshot_retention_limit = 7
}
`, acctest.RandInt(), acctest.RandInt(), acctest.RandInt(), acctest.RandInt(), acctest.RandString(10))

var testAccAWSElasticacheReplicationGroupRedisClusterInVPCConfig = fmt.Sprintf(`
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

resource "aws_elasticache_replication_group" "bar" {
    replication_group_id = "tf-%s"
    replication_group_description = "test description"
    node_type = "cache.t2.micro"
    number_cache_clusters = "2"
    port = 6379
    subnet_group_name = "${aws_elasticache_subnet_group.bar.name}"
    security_group_ids = ["${aws_security_group.bar.id}"]
    parameter_group_name = "default.redis3.2.cluster.on"
    availability_zones = ["us-west-2a","us-west-2b"]
    automatic_failover_enabled = true
    snapshot_window = "02:00-03:00"
    snapshot_retention_limit = 7
    engine_version = "3.2.4"
    maintenance_window = "thu:03:00-thu:04:00"
}
`, acctest.RandInt(), acctest.RandInt(), acctest.RandInt(), acctest.RandInt(), acctest.RandString(10))
