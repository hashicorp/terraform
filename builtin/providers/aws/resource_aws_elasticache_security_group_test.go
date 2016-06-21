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

func TestAccAWSElasticacheSecurityGroup_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSElasticacheSecurityGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheSecurityGroupExists("aws_elasticache_security_group.bar"),
					resource.TestCheckResourceAttr(
						"aws_elasticache_security_group.bar", "description", "Managed by Terraform"),
				),
			},
		},
	})
}

func testAccCheckAWSElasticacheSecurityGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elasticacheconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elasticache_security_group" {
			continue
		}
		res, err := conn.DescribeCacheSecurityGroups(&elasticache.DescribeCacheSecurityGroupsInput{
			CacheSecurityGroupName: aws.String(rs.Primary.ID),
		})
		if awserr, ok := err.(awserr.Error); ok && awserr.Code() == "CacheSecurityGroupNotFound" {
			continue
		}

		if len(res.CacheSecurityGroups) > 0 {
			return fmt.Errorf("cache security group still exists")
		}
		return err
	}
	return nil
}

func testAccCheckAWSElasticacheSecurityGroupExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No cache security group ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).elasticacheconn
		_, err := conn.DescribeCacheSecurityGroups(&elasticache.DescribeCacheSecurityGroupsInput{
			CacheSecurityGroupName: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return fmt.Errorf("CacheSecurityGroup error: %v", err)
		}
		return nil
	}
}

var testAccAWSElasticacheSecurityGroupConfig = fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}

resource "aws_security_group" "bar" {
  name = "tf-test-security-group-%03d"

  ingress {
    from_port   = -1
    to_port     = -1
    protocol    = "icmp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_elasticache_security_group" "bar" {
  name                 = "tf-test-security-group-%03d"
  security_group_names = ["${aws_security_group.bar.name}"]
}
`, acctest.RandInt(), acctest.RandInt())
