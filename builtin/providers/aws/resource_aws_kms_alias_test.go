package aws

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSKmsAlias_basic(t *testing.T) {
	rInt := acctest.RandInt()
	kmsAliasTimestamp := time.Now().Format(time.RFC1123)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKmsAliasDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSKmsSingleAlias(rInt, kmsAliasTimestamp),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsAliasExists("aws_kms_alias.single"),
				),
			},
			{
				Config: testAccAWSKmsSingleAlias_modified(rInt, kmsAliasTimestamp),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsAliasExists("aws_kms_alias.single"),
				),
			},
		},
	})
}

func TestAccAWSKmsAlias_name_prefix(t *testing.T) {
	rInt := acctest.RandInt()
	kmsAliasTimestamp := time.Now().Format(time.RFC1123)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKmsAliasDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSKmsSingleAlias(rInt, kmsAliasTimestamp),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsAliasExists("aws_kms_alias.name_prefix"),
				),
			},
		},
	})
}

func TestAccAWSKmsAlias_no_name(t *testing.T) {
	rInt := acctest.RandInt()
	kmsAliasTimestamp := time.Now().Format(time.RFC1123)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKmsAliasDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSKmsSingleAlias(rInt, kmsAliasTimestamp),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKmsAliasExists("aws_kms_alias.nothing"),
				),
			},
		},
	})
}

func TestAccAWSKmsAlias_multiple(t *testing.T) {
	rInt := acctest.RandInt()
	kmsAliasTimestamp := time.Now().Format(time.RFC1123)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKmsAliasDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSKmsMultipleAliases(rInt, kmsAliasTimestamp),
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

func testAccAWSKmsSingleAlias(rInt int, timestamp string) string {
	return fmt.Sprintf(`
resource "aws_kms_key" "one" {
    description = "Terraform acc test One %s"
    deletion_window_in_days = 7
}
resource "aws_kms_key" "two" {
    description = "Terraform acc test Two %s"
    deletion_window_in_days = 7
}

resource "aws_kms_alias" "name_prefix" {
	name_prefix = "alias/tf-acc-key-alias-%d"
	target_key_id = "${aws_kms_key.one.key_id}"
}

resource "aws_kms_alias" "nothing" {
	target_key_id = "${aws_kms_key.one.key_id}"
}

resource "aws_kms_alias" "single" {
    name = "alias/tf-acc-key-alias-%d"
    target_key_id = "${aws_kms_key.one.key_id}"
}`, timestamp, timestamp, rInt, rInt)
}

func testAccAWSKmsSingleAlias_modified(rInt int, timestamp string) string {
	return fmt.Sprintf(`
resource "aws_kms_key" "one" {
    description = "Terraform acc test One %s"
    deletion_window_in_days = 7
}
resource "aws_kms_key" "two" {
    description = "Terraform acc test Two %s"
    deletion_window_in_days = 7
}

resource "aws_kms_alias" "single" {
    name = "alias/tf-acc-key-alias-%d"
    target_key_id = "${aws_kms_key.two.key_id}"
}`, timestamp, timestamp, rInt)
}

func testAccAWSKmsMultipleAliases(rInt int, timestamp string) string {
	return fmt.Sprintf(`
resource "aws_kms_key" "single" {
    description = "Terraform acc test One %s"
    deletion_window_in_days = 7
}

resource "aws_kms_alias" "one" {
    name = "alias/tf-acc-alias-one-%d"
    target_key_id = "${aws_kms_key.single.key_id}"
}
resource "aws_kms_alias" "two" {
    name = "alias/tf-acc-alias-two-%d"
    target_key_id = "${aws_kms_key.single.key_id}"
}`, timestamp, rInt, rInt)
}
