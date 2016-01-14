package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
)

func TestAccAWSDBSubnetGroup_basic(t *testing.T) {
	var v rds.DBSubnetGroup

	testCheck := func(*terraform.State) error {
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDBSubnetGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDBSubnetGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDBSubnetGroupExists(
						"aws_db_subnet_group.foo", &v),
					testCheck,
				),
			},
		},
	})
}

// Regression test for https://github.com/hashicorp/terraform/issues/2603 and
// https://github.com/hashicorp/terraform/issues/2664
func TestAccAWSDBSubnetGroup_withUndocumentedCharacters(t *testing.T) {
	var v rds.DBSubnetGroup

	testCheck := func(*terraform.State) error {
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDBSubnetGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDBSubnetGroupConfig_withUnderscoresAndPeriodsAndSpaces,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDBSubnetGroupExists(
						"aws_db_subnet_group.underscores", &v),
					testAccCheckDBSubnetGroupExists(
						"aws_db_subnet_group.periods", &v),
					testAccCheckDBSubnetGroupExists(
						"aws_db_subnet_group.spaces", &v),
					testCheck,
				),
			},
		},
	})
}

func TestResourceAWSDBSubnetGroupNameValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "tEsting",
			ErrCount: 1,
		},
		{
			Value:    "testing?",
			ErrCount: 1,
		},
		{
			Value:    "default",
			ErrCount: 1,
		},
		{
			Value:    randomString(300),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateSubnetGroupName(tc.Value, "aws_db_subnet_group")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the DB Subnet Group name to trigger a validation error")
		}
	}
}

func testAccCheckDBSubnetGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).rdsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_db_subnet_group" {
			continue
		}

		// Try to find the resource
		resp, err := conn.DescribeDBSubnetGroups(
			&rds.DescribeDBSubnetGroupsInput{DBSubnetGroupName: aws.String(rs.Primary.ID)})
		if err == nil {
			if len(resp.DBSubnetGroups) > 0 {
				return fmt.Errorf("still exist.")
			}

			return nil
		}

		// Verify the error is what we want
		rdserr, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if rdserr.Code() != "DBSubnetGroupNotFoundFault" {
			return err
		}
	}

	return nil
}

func testAccCheckDBSubnetGroupExists(n string, v *rds.DBSubnetGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).rdsconn
		resp, err := conn.DescribeDBSubnetGroups(
			&rds.DescribeDBSubnetGroupsInput{DBSubnetGroupName: aws.String(rs.Primary.ID)})
		if err != nil {
			return err
		}
		if len(resp.DBSubnetGroups) == 0 {
			return fmt.Errorf("DbSubnetGroup not found")
		}

		*v = *resp.DBSubnetGroups[0]

		return nil
	}
}

const testAccDBSubnetGroupConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
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

resource "aws_db_subnet_group" "foo" {
	name = "foo"
	description = "foo description"
	subnet_ids = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}"]
	tags {
		Name = "tf-dbsubnet-group-test"
	}
}
`

const testAccDBSubnetGroupConfig_withUnderscoresAndPeriodsAndSpaces = `
resource "aws_vpc" "main" {
    cidr_block = "192.168.0.0/16"
}

resource "aws_subnet" "frontend" {
    vpc_id = "${aws_vpc.main.id}"
    availability_zone = "us-west-2b"
    cidr_block = "192.168.1.0/24"
}

resource "aws_subnet" "backend" {
    vpc_id = "${aws_vpc.main.id}"
    availability_zone = "us-west-2c"
    cidr_block = "192.168.2.0/24"
}

resource "aws_db_subnet_group" "underscores" {
    name = "with_underscores"
    description = "Our main group of subnets"
    subnet_ids = ["${aws_subnet.frontend.id}", "${aws_subnet.backend.id}"]
}

resource "aws_db_subnet_group" "periods" {
    name = "with.periods"
    description = "Our main group of subnets"
    subnet_ids = ["${aws_subnet.frontend.id}", "${aws_subnet.backend.id}"]
}

resource "aws_db_subnet_group" "spaces" {
    name = "with spaces"
    description = "Our main group of subnets"
    subnet_ids = ["${aws_subnet.frontend.id}", "${aws_subnet.backend.id}"]
}
`
