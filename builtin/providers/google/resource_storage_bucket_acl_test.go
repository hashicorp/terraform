package google

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	//"google.golang.org/api/storage/v1"
)

var roleEntityBasic1 = "OWNER:user-omeemail@gmail.com"

var roleEntityBasic2 = "READER:user-anotheremail@gmail.com"

var roleEntityBasic3_owner = "OWNER:user-yetanotheremail@gmail.com"

var roleEntityBasic3_reader = "READER:user-yetanotheremail@gmail.com"

var testAclBucketName = fmt.Sprintf("%s-%d", "tf-test-acl-bucket", rand.New(rand.NewSource(time.Now().UnixNano())).Int())

func TestAccGoogleStorageBucketAcl_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageBucketAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageBucketsAclBasic1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageBucketAcl(testAclBucketName, roleEntityBasic1),
					testAccCheckGoogleStorageBucketAcl(testAclBucketName, roleEntityBasic2),
				),
			},
		},
	})
}

func TestAccGoogleStorageBucketAcl_upgrade(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageBucketAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageBucketsAclBasic1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageBucketAcl(testAclBucketName, roleEntityBasic1),
					testAccCheckGoogleStorageBucketAcl(testAclBucketName, roleEntityBasic2),
				),
			},

			resource.TestStep{
				Config: testGoogleStorageBucketsAclBasic2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageBucketAcl(testAclBucketName, roleEntityBasic2),
					testAccCheckGoogleStorageBucketAcl(testAclBucketName, roleEntityBasic3_owner),
				),
			},

			resource.TestStep{
				Config: testGoogleStorageBucketsAclBasicDelete,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageBucketAclDelete(testAclBucketName, roleEntityBasic1),
					testAccCheckGoogleStorageBucketAclDelete(testAclBucketName, roleEntityBasic2),
					testAccCheckGoogleStorageBucketAclDelete(testAclBucketName, roleEntityBasic3_owner),
				),
			},
		},
	})
}

func TestAccGoogleStorageBucketAcl_downgrade(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageBucketAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageBucketsAclBasic2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageBucketAcl(testAclBucketName, roleEntityBasic2),
					testAccCheckGoogleStorageBucketAcl(testAclBucketName, roleEntityBasic3_owner),
				),
			},

			resource.TestStep{
				Config: testGoogleStorageBucketsAclBasic3,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageBucketAcl(testAclBucketName, roleEntityBasic2),
					testAccCheckGoogleStorageBucketAcl(testAclBucketName, roleEntityBasic3_reader),
				),
			},

			resource.TestStep{
				Config: testGoogleStorageBucketsAclBasicDelete,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageBucketAclDelete(testAclBucketName, roleEntityBasic1),
					testAccCheckGoogleStorageBucketAclDelete(testAclBucketName, roleEntityBasic2),
					testAccCheckGoogleStorageBucketAclDelete(testAclBucketName, roleEntityBasic3_owner),
				),
			},
		},
	})
}

func TestAccGoogleStorageBucketAcl_predefined(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageBucketAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageBucketsAclPredefined,
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

var testGoogleStorageBucketsAclBasic1 = fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_acl" "acl" {
	bucket = "${google_storage_bucket.bucket.name}"
	role_entity = ["%s", "%s"]
}
`, testAclBucketName, roleEntityBasic1, roleEntityBasic2)

var testGoogleStorageBucketsAclBasic2 = fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_acl" "acl" {
	bucket = "${google_storage_bucket.bucket.name}"
	role_entity = ["%s", "%s"]
}
`, testAclBucketName, roleEntityBasic2, roleEntityBasic3_owner)

var testGoogleStorageBucketsAclBasicDelete = fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_acl" "acl" {
	bucket = "${google_storage_bucket.bucket.name}"
	role_entity = []
}
`, testAclBucketName)

var testGoogleStorageBucketsAclBasic3 = fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_acl" "acl" {
	bucket = "${google_storage_bucket.bucket.name}"
	role_entity = ["%s", "%s"]
}
`, testAclBucketName, roleEntityBasic2, roleEntityBasic3_reader)

var testGoogleStorageBucketsAclPredefined = fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_acl" "acl" {
	bucket = "${google_storage_bucket.bucket.name}"
	predefined_acl = "projectPrivate"
	default_acl = "projectPrivate"
}
`, testAclBucketName)
