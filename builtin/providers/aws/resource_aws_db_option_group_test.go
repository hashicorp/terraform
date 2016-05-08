package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDBOptionGroup_basic(t *testing.T) {
	var v rds.OptionGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBOptionGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDBOptionGroupBasicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBOptionGroupExists("aws_db_option_group.bar", &v),
					testAccCheckAWSDBOptionGroupAttributes(&v),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "name", "option-group-test-terraform"),
				),
			},
		},
	})
}

func TestAccAWSDBOptionGroup_sqlServerOptionsUpdate(t *testing.T) {
	var v rds.OptionGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBOptionGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDBOptionGroupSqlServerEEOptions,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBOptionGroupExists("aws_db_option_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "name", "option-group-test-terraform"),
				),
			},

			resource.TestStep{
				Config: testAccAWSDBOptionGroupSqlServerEEOptions_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBOptionGroupExists("aws_db_option_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "name", "option-group-test-terraform"),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "option.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSDBOptionGroup_multipleOptions(t *testing.T) {
	var v rds.OptionGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBOptionGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDBOptionGroupMultipleOptions,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBOptionGroupExists("aws_db_option_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "name", "option-group-test-terraform"),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "option.#", "2"),
				),
			},
		},
	})
}

func testAccCheckAWSDBOptionGroupAttributes(v *rds.OptionGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if *v.EngineName != "mysql" {
			return fmt.Errorf("bad engine_name: %#v", *v.EngineName)
		}

		if *v.MajorEngineVersion != "5.6" {
			return fmt.Errorf("bad major_engine_version: %#v", *v.MajorEngineVersion)
		}

		if *v.OptionGroupDescription != "Test option group for terraform" {
			return fmt.Errorf("bad option_group_description: %#v", *v.OptionGroupDescription)
		}

		return nil
	}
}

func TestResourceAWSDBOptionGroupName_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
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
		_, errors := validateDbOptionGroupName(tc.Value, "aws_db_option_group_name")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the DB Option Group Name to trigger a validation error")
		}
	}
}

func testAccCheckAWSDBOptionGroupExists(n string, v *rds.OptionGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DB Option Group Name is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).rdsconn

		opts := rds.DescribeOptionGroupsInput{
			OptionGroupName: aws.String(rs.Primary.ID),
		}

		resp, err := conn.DescribeOptionGroups(&opts)

		if err != nil {
			return err
		}

		if len(resp.OptionGroupsList) != 1 ||
			*resp.OptionGroupsList[0].OptionGroupName != rs.Primary.ID {
			return fmt.Errorf("DB Option Group not found")
		}

		*v = *resp.OptionGroupsList[0]

		return nil
	}
}

func testAccCheckAWSDBOptionGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).rdsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_db_option_group" {
			continue
		}

		resp, err := conn.DescribeOptionGroups(
			&rds.DescribeOptionGroupsInput{
				OptionGroupName: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if len(resp.OptionGroupsList) != 0 &&
				*resp.OptionGroupsList[0].OptionGroupName == rs.Primary.ID {
				return fmt.Errorf("DB Option Group still exists")
			}
		}

		// Verify the error
		newerr, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if newerr.Code() != "OptionGroupNotFoundFault" {
			return err
		}
	}

	return nil
}

const testAccAWSDBOptionGroupBasicConfig = `
resource "aws_db_option_group" "bar" {
  name                     = "option-group-test-terraform"
  option_group_description = "Test option group for terraform"
  engine_name              = "mysql"
  major_engine_version     = "5.6"
}
`

const testAccAWSDBOptionGroupSqlServerEEOptions = `
resource "aws_db_option_group" "bar" {
  name                     = "option-group-test-terraform"
  option_group_description = "Test option group for terraform"
  engine_name              = "sqlserver-ee"
  major_engine_version     = "11.00"
}
`

const testAccAWSDBOptionGroupSqlServerEEOptions_update = `
resource "aws_db_option_group" "bar" {
  name                     = "option-group-test-terraform"
  option_group_description = "Test option group for terraform"
  engine_name              = "sqlserver-ee"
  major_engine_version     = "11.00"

  option {
    option_name = "Mirroring"
  }
}
`

const testAccAWSDBOptionGroupMultipleOptions = `
resource "aws_db_option_group" "bar" {
  name                     = "option-group-test-terraform"
  option_group_description = "Test option group for terraform"
  engine_name              = "oracle-se"
  major_engine_version     = "11.2"

  option {
    option_name = "STATSPACK"
  }

  option {
    option_name = "XMLDB"
  }
}
`
