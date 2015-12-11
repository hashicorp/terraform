package aws

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDBParameterGroup_basic(t *testing.T) {
	var v rds.DBParameterGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBParameterGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDBParameterGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBParameterGroupExists("aws_db_parameter_group.bar", &v),
					testAccCheckAWSDBParameterGroupAttributes(&v),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "name", "parameter-group-test-terraform"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "family", "mysql5.6"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "description", "Test parameter group for terraform"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.1708034931.name", "character_set_results"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.1708034931.value", "utf8"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2421266705.name", "character_set_server"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2421266705.value", "utf8"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2478663599.name", "character_set_client"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2478663599.value", "utf8"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "tags.#", "1"),
				),
			},
			resource.TestStep{
				Config: testAccAWSDBParameterGroupAddParametersConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBParameterGroupExists("aws_db_parameter_group.bar", &v),
					testAccCheckAWSDBParameterGroupAttributes(&v),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "name", "parameter-group-test-terraform"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "family", "mysql5.6"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "description", "Test parameter group for terraform"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.1706463059.name", "collation_connection"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.1706463059.value", "utf8_unicode_ci"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.1708034931.name", "character_set_results"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.1708034931.value", "utf8"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2421266705.name", "character_set_server"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2421266705.value", "utf8"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2475805061.name", "collation_server"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2475805061.value", "utf8_unicode_ci"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2478663599.name", "character_set_client"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2478663599.value", "utf8"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "tags.#", "2"),
				),
			},
		},
	})
}

func TestAccAWSDBParameterGroupOnly(t *testing.T) {
	var v rds.DBParameterGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBParameterGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDBParameterGroupOnlyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBParameterGroupExists("aws_db_parameter_group.bar", &v),
					testAccCheckAWSDBParameterGroupAttributes(&v),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "name", "parameter-group-test-terraform"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "family", "mysql5.6"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "description", "Test parameter group for terraform"),
				),
			},
		},
	})
}

func TestResourceAWSDBParameterGroupName_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "tEsting123",
			ErrCount: 1,
		},
		{
			Value:    "testing123!",
			ErrCount: 1,
		},
		{
			Value:    "1testing123",
			ErrCount: 1,
		},
		{
			Value:    "testing--123",
			ErrCount: 1,
		},
		{
			Value:    "testing123-",
			ErrCount: 1,
		},
		{
			Value:    randomString(256),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateDbParamGroupName(tc.Value, "aws_db_parameter_group_name")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the DB Parameter Group Name to trigger a validation error")
		}
	}
}

func testAccCheckAWSDBParameterGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).rdsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_db_parameter_group" {
			continue
		}

		// Try to find the Group
		resp, err := conn.DescribeDBParameterGroups(
			&rds.DescribeDBParameterGroupsInput{
				DBParameterGroupName: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if len(resp.DBParameterGroups) != 0 &&
				*resp.DBParameterGroups[0].DBParameterGroupName == rs.Primary.ID {
				return fmt.Errorf("DB Parameter Group still exists")
			}
		}

		// Verify the error
		newerr, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if newerr.Code() != "DBParameterGroupNotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSDBParameterGroupAttributes(v *rds.DBParameterGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if *v.DBParameterGroupName != "parameter-group-test-terraform" {
			return fmt.Errorf("bad name: %#v", v.DBParameterGroupName)
		}

		if *v.DBParameterGroupFamily != "mysql5.6" {
			return fmt.Errorf("bad family: %#v", v.DBParameterGroupFamily)
		}

		if *v.Description != "Test parameter group for terraform" {
			return fmt.Errorf("bad description: %#v", v.Description)
		}

		return nil
	}
}

func testAccCheckAWSDBParameterGroupExists(n string, v *rds.DBParameterGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DB Parameter Group ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).rdsconn

		opts := rds.DescribeDBParameterGroupsInput{
			DBParameterGroupName: aws.String(rs.Primary.ID),
		}

		resp, err := conn.DescribeDBParameterGroups(&opts)

		if err != nil {
			return err
		}

		if len(resp.DBParameterGroups) != 1 ||
			*resp.DBParameterGroups[0].DBParameterGroupName != rs.Primary.ID {
			return fmt.Errorf("DB Parameter Group not found")
		}

		*v = *resp.DBParameterGroups[0]

		return nil
	}
}

func randomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

const testAccAWSDBParameterGroupConfig = `
resource "aws_db_parameter_group" "bar" {
	name = "parameter-group-test-terraform"
	family = "mysql5.6"
	description = "Test parameter group for terraform"
	parameter {
	  name = "character_set_server"
	  value = "utf8"
	}
	parameter {
	  name = "character_set_client"
	  value = "utf8"
	}
	parameter{
	  name = "character_set_results"
	  value = "utf8"
	}
	tags {
		foo = "bar"
	}
}
`

const testAccAWSDBParameterGroupAddParametersConfig = `
resource "aws_db_parameter_group" "bar" {
	name = "parameter-group-test-terraform"
	family = "mysql5.6"
	description = "Test parameter group for terraform"
	parameter {
	  name = "character_set_server"
	  value = "utf8"
	}
	parameter {
	  name = "character_set_client"
	  value = "utf8"
	}
	parameter{
	  name = "character_set_results"
	  value = "utf8"
	}
	parameter {
	  name = "collation_server"
	  value = "utf8_unicode_ci"
	}
	parameter {
	  name = "collation_connection"
	  value = "utf8_unicode_ci"
	}
	tags {
		foo = "bar"
		baz = "foo"
	}
}
`

const testAccAWSDBParameterGroupOnlyConfig = `
resource "aws_db_parameter_group" "bar" {
	name = "parameter-group-test-terraform"
	family = "mysql5.6"
	description = "Test parameter group for terraform"
}
`
