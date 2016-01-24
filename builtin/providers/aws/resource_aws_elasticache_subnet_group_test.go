package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSElasticacheSubnetGroup_basic(t *testing.T) {
	var csg elasticache.CacheSubnetGroup
	config := fmt.Sprintf(testAccAWSElasticacheSubnetGroupConfig, genRandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheSubnetGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheSubnetGroupExists("aws_elasticache_subnet_group.bar", &csg),
				),
			},
		},
	})
}

func TestAccAWSElasticacheSubnetGroup_update(t *testing.T) {
	var csg elasticache.CacheSubnetGroup
	rn := "aws_elasticache_subnet_group.bar"
	ri := genRandInt()
	preConfig := fmt.Sprintf(testAccAWSElasticacheSubnetGroupUpdateConfigPre, ri)
	postConfig := fmt.Sprintf(testAccAWSElasticacheSubnetGroupUpdateConfigPost, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheSubnetGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheSubnetGroupExists(rn, &csg),
					testAccCheckAWSElastiCacheSubnetGroupAttrs(&csg, rn, 1),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheSubnetGroupExists(rn, &csg),
					testAccCheckAWSElastiCacheSubnetGroupAttrs(&csg, rn, 2),
				),
			},
		},
	})
}

func testAccCheckAWSElasticacheSubnetGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elasticacheconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elasticache_subnet_group" {
			continue
		}
		res, err := conn.DescribeCacheSubnetGroups(&elasticache.DescribeCacheSubnetGroupsInput{
			CacheSubnetGroupName: aws.String(rs.Primary.ID),
		})
		if err != nil {
			// Verify the error is what we want
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "CacheSubnetGroupNotFoundFault" {
				continue
			}
			return err
		}
		if len(res.CacheSubnetGroups) > 0 {
			return fmt.Errorf("still exist.")
		}
	}
	return nil
}

func testAccCheckAWSElasticacheSubnetGroupExists(n string, csg *elasticache.CacheSubnetGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No cache subnet group ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).elasticacheconn
		resp, err := conn.DescribeCacheSubnetGroups(&elasticache.DescribeCacheSubnetGroupsInput{
			CacheSubnetGroupName: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return fmt.Errorf("CacheSubnetGroup error: %v", err)
		}

		for _, c := range resp.CacheSubnetGroups {
			if rs.Primary.ID == *c.CacheSubnetGroupName {
				*csg = *c
			}
		}

		if csg == nil {
			return fmt.Errorf("cache subnet group not found")
		}
		return nil
	}
}

func testAccCheckAWSElastiCacheSubnetGroupAttrs(csg *elasticache.CacheSubnetGroup, n string, count int) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if len(csg.Subnets) != count {
			return fmt.Errorf("Bad cache subnet count, expected: %d, got: %d", count, len(csg.Subnets))
		}

		if rs.Primary.Attributes["description"] != *csg.CacheSubnetGroupDescription {
			return fmt.Errorf("Bad cache subnet description, expected: %s, got: %s", rs.Primary.Attributes["description"], *csg.CacheSubnetGroupDescription)
		}

		return nil
	}
}

var testAccAWSElasticacheSubnetGroupConfig = `
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
    // Including uppercase letters in this name to ensure
    // that we correctly handle the fact that the API
    // normalizes names to lowercase.
    name = "tf-TEST-cache-subnet-%03d"
    description = "tf-test-cache-subnet-group-descr"
    subnet_ids = ["${aws_subnet.foo.id}"]
}
`
var testAccAWSElasticacheSubnetGroupUpdateConfigPre = `
resource "aws_vpc" "foo" {
    cidr_block = "10.0.0.0/16"
    tags {
            Name = "tf-elc-sub-test"
    }
}

resource "aws_subnet" "foo" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "10.0.1.0/24"
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
`

var testAccAWSElasticacheSubnetGroupUpdateConfigPost = `
resource "aws_vpc" "foo" {
    cidr_block = "10.0.0.0/16"
    tags {
            Name = "tf-elc-sub-test"
    }
}

resource "aws_subnet" "foo" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "10.0.1.0/24"
    availability_zone = "us-west-2a"
    tags {
            Name = "tf-test"
    }
}

resource "aws_subnet" "bar" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "10.0.2.0/24"
    availability_zone = "us-west-2a"
    tags {
            Name = "tf-test-foo-update"
    }
}

resource "aws_elasticache_subnet_group" "bar" {
    name = "tf-test-cache-subnet-%03d"
    description = "tf-test-cache-subnet-group-descr-edited"
    subnet_ids = [
			"${aws_subnet.foo.id}",
			"${aws_subnet.bar.id}",
		]
}
`
