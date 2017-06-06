package aws

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func init() {
	resource.AddTestSweepers("aws_db_option_group", &resource.Sweeper{
		Name: "aws_db_option_group",
		F:    testSweepDbOptionGroups,
	})
}

func testSweepDbOptionGroups(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}

	conn := client.(*AWSClient).rdsconn

	opts := rds.DescribeOptionGroupsInput{}
	resp, err := conn.DescribeOptionGroups(&opts)
	if err != nil {
		return fmt.Errorf("error describing DB Option Groups in Sweeper: %s", err)
	}

	for _, og := range resp.OptionGroupsList {
		var testOptGroup bool
		for _, testName := range []string{"option-group-test-terraform-", "tf-test"} {
			if strings.HasPrefix(*og.OptionGroupName, testName) {
				testOptGroup = true
			}
		}

		if !testOptGroup {
			continue
		}

		deleteOpts := &rds.DeleteOptionGroupInput{
			OptionGroupName: og.OptionGroupName,
		}

		ret := resource.Retry(1*time.Minute, func() *resource.RetryError {
			_, err := conn.DeleteOptionGroup(deleteOpts)
			if err != nil {
				if awsErr, ok := err.(awserr.Error); ok {
					if awsErr.Code() == "InvalidOptionGroupStateFault" {
						log.Printf("[DEBUG] AWS believes the RDS Option Group is still in use, retrying")
						return resource.RetryableError(awsErr)
					}
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if ret != nil {
			return fmt.Errorf("Error Deleting DB Option Group (%s) in Sweeper: %s", *og.OptionGroupName, ret)
		}
	}

	return nil
}

func TestAccAWSDBOptionGroup_basic(t *testing.T) {
	var v rds.OptionGroup
	rName := fmt.Sprintf("option-group-test-terraform-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBOptionGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDBOptionGroupBasicConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBOptionGroupExists("aws_db_option_group.bar", &v),
					testAccCheckAWSDBOptionGroupAttributes(&v),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "name", rName),
				),
			},
		},
	})
}

func TestAccAWSDBOptionGroup_timeoutBlock(t *testing.T) {
	var v rds.OptionGroup
	rName := fmt.Sprintf("option-group-test-terraform-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBOptionGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDBOptionGroupBasicConfigTimeoutBlock(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBOptionGroupExists("aws_db_option_group.bar", &v),
					testAccCheckAWSDBOptionGroupAttributes(&v),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "name", rName),
				),
			},
		},
	})
}

func TestAccAWSDBOptionGroup_namePrefix(t *testing.T) {
	var v rds.OptionGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBOptionGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDBOptionGroup_namePrefix,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBOptionGroupExists("aws_db_option_group.test", &v),
					testAccCheckAWSDBOptionGroupAttributes(&v),
					resource.TestMatchResourceAttr(
						"aws_db_option_group.test", "name", regexp.MustCompile("^tf-test-")),
				),
			},
		},
	})
}

func TestAccAWSDBOptionGroup_generatedName(t *testing.T) {
	var v rds.OptionGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBOptionGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDBOptionGroup_generatedName,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBOptionGroupExists("aws_db_option_group.test", &v),
					testAccCheckAWSDBOptionGroupAttributes(&v),
				),
			},
		},
	})
}

func TestAccAWSDBOptionGroup_defaultDescription(t *testing.T) {
	var v rds.OptionGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBOptionGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDBOptionGroup_defaultDescription(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBOptionGroupExists("aws_db_option_group.test", &v),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.test", "option_group_description", "Managed by Terraform"),
				),
			},
		},
	})
}

func TestAccAWSDBOptionGroup_basicDestroyWithInstance(t *testing.T) {
	rName := fmt.Sprintf("option-group-test-terraform-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBOptionGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDBOptionGroupBasicDestroyConfig(rName),
			},
		},
	})
}

func TestAccAWSDBOptionGroup_OptionSettings(t *testing.T) {
	var v rds.OptionGroup
	rName := fmt.Sprintf("option-group-test-terraform-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBOptionGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDBOptionGroupOptionSettings(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBOptionGroupExists("aws_db_option_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "name", rName),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "option.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "option.961211605.option_settings.129825347.value", "UTC"),
				),
			},
			{
				Config: testAccAWSDBOptionGroupOptionSettings_update(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBOptionGroupExists("aws_db_option_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "name", rName),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "option.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "option.2422743510.option_settings.1350509764.value", "US/Pacific"),
				),
			},
		},
	})
}

func TestAccAWSDBOptionGroup_sqlServerOptionsUpdate(t *testing.T) {
	var v rds.OptionGroup
	rName := fmt.Sprintf("option-group-test-terraform-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBOptionGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDBOptionGroupSqlServerEEOptions(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBOptionGroupExists("aws_db_option_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "name", rName),
				),
			},

			{
				Config: testAccAWSDBOptionGroupSqlServerEEOptions_update(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBOptionGroupExists("aws_db_option_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "name", rName),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "option.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSDBOptionGroup_multipleOptions(t *testing.T) {
	var v rds.OptionGroup
	rName := fmt.Sprintf("option-group-test-terraform-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBOptionGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDBOptionGroupMultipleOptions(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBOptionGroupExists("aws_db_option_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_db_option_group.bar", "name", rName),
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

func testAccAWSDBOptionGroupBasicConfigTimeoutBlock(r string) string {
	return fmt.Sprintf(`
resource "aws_db_option_group" "bar" {
  name                     = "%s"
  option_group_description = "Test option group for terraform"
  engine_name              = "mysql"
  major_engine_version     = "5.6"

  timeouts {
  	delete = "10m"
  }
}
`, r)
}

func testAccAWSDBOptionGroupBasicConfig(r string) string {
	return fmt.Sprintf(`
resource "aws_db_option_group" "bar" {
  name                     = "%s"
  option_group_description = "Test option group for terraform"
  engine_name              = "mysql"
  major_engine_version     = "5.6"
}
`, r)
}

func testAccAWSDBOptionGroupBasicDestroyConfig(r string) string {
	return fmt.Sprintf(`
resource "aws_db_instance" "bar" {
	allocated_storage = 10
	engine = "MySQL"
	engine_version = "5.6.21"
	instance_class = "db.t2.micro"
	name = "baz"
	password = "barbarbarbar"
	username = "foo"


	# Maintenance Window is stored in lower case in the API, though not strictly
	# documented. Terraform will downcase this to match (as opposed to throw a
	# validation error).
	maintenance_window = "Fri:09:00-Fri:09:30"

	backup_retention_period = 0
	skip_final_snapshot = true

	option_group_name = "${aws_db_option_group.bar.name}"
}

resource "aws_db_option_group" "bar" {
  name                     = "%s"
  option_group_description = "Test option group for terraform"
  engine_name              = "mysql"
  major_engine_version     = "5.6"
}
`, r)
}

func testAccAWSDBOptionGroupOptionSettings(r string) string {
	return fmt.Sprintf(`
resource "aws_db_option_group" "bar" {
  name                     = "%s"
  option_group_description = "Test option group for terraform"
  engine_name              = "oracle-ee"
  major_engine_version     = "11.2"

  option {
    option_name = "Timezone"
    option_settings {
      name = "TIME_ZONE"
      value = "UTC"
    }
  }
}
`, r)
}

func testAccAWSDBOptionGroupOptionSettings_update(r string) string {
	return fmt.Sprintf(`
resource "aws_db_option_group" "bar" {
  name                     = "%s"
  option_group_description = "Test option group for terraform"
  engine_name              = "oracle-ee"
  major_engine_version     = "11.2"

  option {
    option_name = "Timezone"
    option_settings {
      name = "TIME_ZONE"
      value = "US/Pacific"
    }
  }
}
`, r)
}

func testAccAWSDBOptionGroupSqlServerEEOptions(r string) string {
	return fmt.Sprintf(`
resource "aws_db_option_group" "bar" {
  name                     = "%s"
  option_group_description = "Test option group for terraform"
  engine_name              = "sqlserver-ee"
  major_engine_version     = "11.00"
}
`, r)
}

func testAccAWSDBOptionGroupSqlServerEEOptions_update(r string) string {
	return fmt.Sprintf(`
resource "aws_db_option_group" "bar" {
  name                     = "%s"
  option_group_description = "Test option group for terraform"
  engine_name              = "sqlserver-ee"
  major_engine_version     = "11.00"

  option {
    option_name = "Mirroring"
  }
}
`, r)
}

func testAccAWSDBOptionGroupMultipleOptions(r string) string {
	return fmt.Sprintf(`
resource "aws_db_option_group" "bar" {
  name                     = "%s"
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
`, r)
}

const testAccAWSDBOptionGroup_namePrefix = `
resource "aws_db_option_group" "test" {
  name_prefix = "tf-test-"
  option_group_description = "Test option group for terraform"
  engine_name = "mysql"
  major_engine_version = "5.6"
}
`

const testAccAWSDBOptionGroup_generatedName = `
resource "aws_db_option_group" "test" {
  option_group_description = "Test option group for terraform"
  engine_name = "mysql"
  major_engine_version = "5.6"
}
`

func testAccAWSDBOptionGroup_defaultDescription(n int) string {
	return fmt.Sprintf(`
resource "aws_db_option_group" "test" {
  name = "tf-test-%d"
  engine_name = "mysql"
  major_engine_version = "5.6"
}
`, n)
}
