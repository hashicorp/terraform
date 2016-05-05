package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	//"google.golang.org/api/storage/v1"
)

var roleEntityBasic1 = "OWNER:user-omeemail@gmail.com"

var roleEntityBasic2 = "READER:user-anotheremail@gmail.com"

var roleEntityBasic3_owner = "OWNER:user-yetanotheremail@gmail.com"

var roleEntityBasic3_reader = "READER:user-yetanotheremail@gmail.com"

func testBucketName() string {
	return fmt.Sprintf("%s-%d", "tf-test-acl-bucket", acctest.RandInt())
}

func TestAccGoogleStorageBucketAcl_basic(t *testing.T) {
	bucketName := testBucketName()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageBucketAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageBucketsAclBasic1(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageBucketAcl(bucketName, roleEntityBasic1),
					testAccCheckGoogleStorageBucketAcl(bucketName, roleEntityBasic2),
				),
			},
		},
	})
}

func TestAccGoogleStorageBucketAcl_upgrade(t *testing.T) {
	bucketName := testBucketName()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageBucketAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageBucketsAclBasic1(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageBucketAcl(bucketName, roleEntityBasic1),
					testAccCheckGoogleStorageBucketAcl(bucketName, roleEntityBasic2),
				),
			},

			resource.TestStep{
				Config: testGoogleStorageBucketsAclBasic2(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageBucketAcl(bucketName, roleEntityBasic2),
					testAccCheckGoogleStorageBucketAcl(bucketName, roleEntityBasic3_owner),
				),
			},

			resource.TestStep{
				Config: testGoogleStorageBucketsAclBasicDelete(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageBucketAclDelete(bucketName, roleEntityBasic1),
					testAccCheckGoogleStorageBucketAclDelete(bucketName, roleEntityBasic2),
					testAccCheckGoogleStorageBucketAclDelete(bucketName, roleEntityBasic3_owner),
				),
			},
		},
	})
}

func TestAccGoogleStorageBucketAcl_downgrade(t *testing.T) {
	bucketName := testBucketName()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageBucketAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageBucketsAclBasic2(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageBucketAcl(bucketName, roleEntityBasic2),
					testAccCheckGoogleStorageBucketAcl(bucketName, roleEntityBasic3_owner),
				),
			},

			resource.TestStep{
				Config: testGoogleStorageBucketsAclBasic3(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageBucketAcl(bucketName, roleEntityBasic2),
					testAccCheckGoogleStorageBucketAcl(bucketName, roleEntityBasic3_reader),
				),
			},

			resource.TestStep{
				Config: testGoogleStorageBucketsAclBasicDelete(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageBucketAclDelete(bucketName, roleEntityBasic1),
					testAccCheckGoogleStorageBucketAclDelete(bucketName, roleEntityBasic2),
					testAccCheckGoogleStorageBucketAclDelete(bucketName, roleEntityBasic3_owner),
				),
			},
		},
	})
}

func TestAccGoogleStorageBucketAcl_predefined(t *testing.T) {
	bucketName := testBucketName()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageBucketAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageBucketsAclPredefined(bucketName),
			},
		},
	})
}

func testAccCheckGoogleStorageBucketAclDelete(bucket, roleEntityS string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		roleEntity, _ := getRoleEntityPair(roleEntityS)
		config := testAccProvider.Meta().(*Config)

		_, err := config.clientStorage.BucketAccessControls.Get(bucket, roleEntity.Entity).Do()

		if err != nil {
			return nil
		}

		return fmt.Errorf("Error, entity %s still exists", roleEntity.Entity)
	}
}

func testAccCheckGoogleStorageBucketAcl(bucket, roleEntityS string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		roleEntity, _ := getRoleEntityPair(roleEntityS)
		config := testAccProvider.Meta().(*Config)

		res, err := config.clientStorage.BucketAccessControls.Get(bucket, roleEntity.Entity).Do()

		if err != nil {
			return fmt.Errorf("Error retrieving contents of acl for bucket %s: %s", bucket, err)
		}

		if res.Role != roleEntity.Role {
			return fmt.Errorf("Error, Role mismatch %s != %s", res.Role, roleEntity.Role)
		}

		return nil
	}
}

func testAccGoogleStorageBucketAclDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_storage_bucket_acl" {
			continue
		}

		bucket := rs.Primary.Attributes["bucket"]

		_, err := config.clientStorage.BucketAccessControls.List(bucket).Do()

		if err == nil {
			return fmt.Errorf("Acl for bucket %s still exists", bucket)
		}
	}

	return nil
}

func testGoogleStorageBucketsAclBasic1(bucketName string) string {
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_acl" "acl" {
	bucket = "${google_storage_bucket.bucket.name}"
	role_entity = ["%s", "%s"]
}
`, bucketName, roleEntityBasic1, roleEntityBasic2)
}

func testGoogleStorageBucketsAclBasic2(bucketName string) string {
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_acl" "acl" {
	bucket = "${google_storage_bucket.bucket.name}"
	role_entity = ["%s", "%s"]
}
`, bucketName, roleEntityBasic2, roleEntityBasic3_owner)
}

func testGoogleStorageBucketsAclBasicDelete(bucketName string) string {
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_acl" "acl" {
	bucket = "${google_storage_bucket.bucket.name}"
	role_entity = []
}
`, bucketName)
}

func testGoogleStorageBucketsAclBasic3(bucketName string) string {
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_acl" "acl" {
	bucket = "${google_storage_bucket.bucket.name}"
	role_entity = ["%s", "%s"]
}
`, bucketName, roleEntityBasic2, roleEntityBasic3_reader)
}

func testGoogleStorageBucketsAclPredefined(bucketName string) string {
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_acl" "acl" {
	bucket = "${google_storage_bucket.bucket.name}"
	predefined_acl = "projectPrivate"
	default_acl = "projectPrivate"
}
`, bucketName)
}
