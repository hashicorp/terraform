package aws

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSKmsAlias_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKmsAliasDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSKmsSingleAlias,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsAliasExists("aws_kms_alias.single"),
				),
			},
			resource.TestStep{
				Config: testAccAWSKmsSingleAlias_modified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsAliasExists("aws_kms_alias.single"),
				),
			},
		},
	})
}

func TestAccAWSKmsAlias_name_prefix(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKmsAliasDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSKmsSingleAlias,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsAliasExists("aws_kms_alias.name_prefix"),
				),
			},
		},
	})
}

func TestAccAWSKmsAlias_no_name(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKmsAliasDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSKmsSingleAlias,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsAliasExists("aws_kms_alias.nothing"),
				),
			},
		},
	})
}

func TestAccAWSKmsAlias_multiple(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKmsAliasDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSKmsMultipleAliases,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsAliasExists("aws_kms_alias.one"),
					testAccCheckAWSKmsAliasExists("aws_kms_alias.two"),
				),
			},
		},
	})
}

func testAccCheckAWSKmsAliasDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).kmsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_kms_alias" {
			continue
		}

		entry, err := findKmsAliasByName(conn, rs.Primary.ID, nil)
		if err != nil {
			return err
		}
		if entry != nil {
			return fmt.Errorf("KMS alias still exists:\n%#v", entry)
		}

		return nil
	}

	return nil
}

func testAccCheckAWSKmsAliasExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		return nil
	}
}

var kmsAliasTimestamp = time.Now().Format(time.RFC1123)
var testAccAWSKmsSingleAlias = fmt.Sprintf(`
resource "aws_kms_key" "one" {
    description = "Terraform acc test One %s"
    deletion_window_in_days = 7
}
resource "aws_kms_key" "two" {
    description = "Terraform acc test Two %s"
    deletion_window_in_days = 7
}

resource "aws_kms_alias" "name_prefix" {
	name_prefix = "alias/tf-acc-key-alias"
	target_key_id = "${aws_kms_key.one.key_id}"
}

resource "aws_kms_alias" "nothing" {
	target_key_id = "${aws_kms_key.one.key_id}"
}

resource "aws_kms_alias" "single" {
    name = "alias/tf-acc-key-alias"
    target_key_id = "${aws_kms_key.one.key_id}"
}`, kmsAliasTimestamp, kmsAliasTimestamp)

var testAccAWSKmsSingleAlias_modified = fmt.Sprintf(`
resource "aws_kms_key" "one" {
    description = "Terraform acc test One %s"
    deletion_window_in_days = 7
}
resource "aws_kms_key" "two" {
    description = "Terraform acc test Two %s"
    deletion_window_in_days = 7
}

resource "aws_kms_alias" "single" {
    name = "alias/tf-acc-key-alias"
    target_key_id = "${aws_kms_key.two.key_id}"
}`, kmsAliasTimestamp, kmsAliasTimestamp)

var testAccAWSKmsMultipleAliases = fmt.Sprintf(`
resource "aws_kms_key" "single" {
    description = "Terraform acc test One %s"
    deletion_window_in_days = 7
}

resource "aws_kms_alias" "one" {
    name = "alias/tf-acc-key-alias-one"
    target_key_id = "${aws_kms_key.single.key_id}"
}
resource "aws_kms_alias" "two" {
    name = "alias/tf-acc-key-alias-two"
    target_key_id = "${aws_kms_key.single.key_id}"
}`, kmsAliasTimestamp)
