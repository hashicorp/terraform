package google

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	//"google.golang.org/api/storage/v1"
)

var tfObjectAcl, errObjectAcl = ioutil.TempFile("", "tf-gce-test")
var testAclObjectName = fmt.Sprintf("%s-%d", "tf-test-acl-object",
	rand.New(rand.NewSource(time.Now().UnixNano())).Int())

func TestAccGoogleStorageObjectAcl_basic(t *testing.T) {
	objectData := []byte("data data data")
	ioutil.WriteFile(tfObjectAcl.Name(), objectData, 0644)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			if errObjectAcl != nil {
				panic(errObjectAcl)
			}
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageObjectAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageObjectsAclBasic1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageObjectAcl(testAclBucketName,
						testAclObjectName, roleEntityBasic1),
					testAccCheckGoogleStorageObjectAcl(testAclBucketName,
						testAclObjectName, roleEntityBasic2),
				),
			},
		},
	})
}

func TestAccGoogleStorageObjectAcl_upgrade(t *testing.T) {
	objectData := []byte("data data data")
	ioutil.WriteFile(tfObjectAcl.Name(), objectData, 0644)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			if errObjectAcl != nil {
				panic(errObjectAcl)
			}
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageObjectAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageObjectsAclBasic1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageObjectAcl(testAclBucketName,
						testAclObjectName, roleEntityBasic1),
					testAccCheckGoogleStorageObjectAcl(testAclBucketName,
						testAclObjectName, roleEntityBasic2),
				),
			},

			resource.TestStep{
				Config: testGoogleStorageObjectsAclBasic2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageObjectAcl(testAclBucketName,
						testAclObjectName, roleEntityBasic2),
					testAccCheckGoogleStorageObjectAcl(testAclBucketName,
						testAclObjectName, roleEntityBasic3_owner),
				),
			},

			resource.TestStep{
				Config: testGoogleStorageObjectsAclBasicDelete,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageObjectAclDelete(testAclBucketName,
						testAclObjectName, roleEntityBasic1),
					testAccCheckGoogleStorageObjectAclDelete(testAclBucketName,
						testAclObjectName, roleEntityBasic2),
					testAccCheckGoogleStorageObjectAclDelete(testAclBucketName,
						testAclObjectName, roleEntityBasic3_reader),
				),
			},
		},
	})
}

func TestAccGoogleStorageObjectAcl_downgrade(t *testing.T) {
	objectData := []byte("data data data")
	ioutil.WriteFile(tfObjectAcl.Name(), objectData, 0644)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			if errObjectAcl != nil {
				panic(errObjectAcl)
			}
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageObjectAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageObjectsAclBasic2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageObjectAcl(testAclBucketName,
						testAclObjectName, roleEntityBasic2),
					testAccCheckGoogleStorageObjectAcl(testAclBucketName,
						testAclObjectName, roleEntityBasic3_owner),
				),
			},

			resource.TestStep{
				Config: testGoogleStorageObjectsAclBasic3,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageObjectAcl(testAclBucketName,
						testAclObjectName, roleEntityBasic2),
					testAccCheckGoogleStorageObjectAcl(testAclBucketName,
						testAclObjectName, roleEntityBasic3_reader),
				),
			},

			resource.TestStep{
				Config: testGoogleStorageObjectsAclBasicDelete,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleStorageObjectAclDelete(testAclBucketName,
						testAclObjectName, roleEntityBasic1),
					testAccCheckGoogleStorageObjectAclDelete(testAclBucketName,
						testAclObjectName, roleEntityBasic2),
					testAccCheckGoogleStorageObjectAclDelete(testAclBucketName,
						testAclObjectName, roleEntityBasic3_reader),
				),
			},
		},
	})
}

func TestAccGoogleStorageObjectAcl_predefined(t *testing.T) {
	objectData := []byte("data data data")
	ioutil.WriteFile(tfObjectAcl.Name(), objectData, 0644)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			if errObjectAcl != nil {
				panic(errObjectAcl)
			}
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleStorageObjectAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleStorageObjectsAclPredefined,
			},
		},
	})
}

func testAccCheckGoogleStorageObjectAcl(bucket, object, roleEntityS string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		roleEntity, _ := getRoleEntityPair(roleEntityS)
		config := testAccProvider.Meta().(*Config)

		res, err := config.clientStorage.ObjectAccessControls.Get(bucket,
			object, roleEntity.Entity).Do()

		if err != nil {
			return fmt.Errorf("Error retrieving contents of acl for bucket %s: %s", bucket, err)
		}

		if res.Role != roleEntity.Role {
			return fmt.Errorf("Error, Role mismatch %s != %s", res.Role, roleEntity.Role)
		}

		return nil
	}
}

func testAccCheckGoogleStorageObjectAclDelete(bucket, object, roleEntityS string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		roleEntity, _ := getRoleEntityPair(roleEntityS)
		config := testAccProvider.Meta().(*Config)

		_, err := config.clientStorage.ObjectAccessControls.Get(bucket,
			object, roleEntity.Entity).Do()

		if err != nil {
			return nil
		}

		return fmt.Errorf("Error, Entity still exists %s", roleEntity.Entity)
	}
}

func testAccGoogleStorageObjectAclDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_storage_bucket_acl" {
			continue
		}

		bucket := rs.Primary.Attributes["bucket"]
		object := rs.Primary.Attributes["object"]

		_, err := config.clientStorage.ObjectAccessControls.List(bucket, object).Do()

		if err == nil {
			return fmt.Errorf("Acl for bucket %s still exists", bucket)
		}
	}

	return nil
}

var testGoogleStorageObjectsAclBasicDelete = fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_object" "object" {
	name = "%s"
	bucket = "${google_storage_bucket.bucket.name}"
	source = "%s"
}

resource "google_storage_object_acl" "acl" {
	object = "${google_storage_bucket_object.object.name}"
	bucket = "${google_storage_bucket.bucket.name}"
	role_entity = []
}
`, testAclBucketName, testAclObjectName, tfObjectAcl.Name())

var testGoogleStorageObjectsAclBasic1 = fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_object" "object" {
	name = "%s"
	bucket = "${google_storage_bucket.bucket.name}"
	source = "%s"
}

resource "google_storage_object_acl" "acl" {
	object = "${google_storage_bucket_object.object.name}"
	bucket = "${google_storage_bucket.bucket.name}"
	role_entity = ["%s", "%s"]
}
`, testAclBucketName, testAclObjectName, tfObjectAcl.Name(),
	roleEntityBasic1, roleEntityBasic2)

var testGoogleStorageObjectsAclBasic2 = fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_object" "object" {
	name = "%s"
	bucket = "${google_storage_bucket.bucket.name}"
	source = "%s"
}

resource "google_storage_object_acl" "acl" {
	object = "${google_storage_bucket_object.object.name}"
	bucket = "${google_storage_bucket.bucket.name}"
	role_entity = ["%s", "%s"]
}
`, testAclBucketName, testAclObjectName, tfObjectAcl.Name(),
	roleEntityBasic2, roleEntityBasic3_owner)

var testGoogleStorageObjectsAclBasic3 = fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_object" "object" {
	name = "%s"
	bucket = "${google_storage_bucket.bucket.name}"
	source = "%s"
}

resource "google_storage_object_acl" "acl" {
	object = "${google_storage_bucket_object.object.name}"
	bucket = "${google_storage_bucket.bucket.name}"
	role_entity = ["%s", "%s"]
}
`, testAclBucketName, testAclObjectName, tfObjectAcl.Name(),
	roleEntityBasic2, roleEntityBasic3_reader)

var testGoogleStorageObjectsAclPredefined = fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_object" "object" {
	name = "%s"
	bucket = "${google_storage_bucket.bucket.name}"
	source = "%s"
}

resource "google_storage_object_acl" "acl" {
	object = "${google_storage_bucket_object.object.name}"
	bucket = "${google_storage_bucket.bucket.name}"
	predefined_acl = "projectPrivate"
}
`, testAclBucketName, testAclObjectName, tfObjectAcl.Name())
