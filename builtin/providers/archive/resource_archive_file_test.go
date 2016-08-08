package archive

import (
	"fmt"
	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"os"
	"testing"
)

func TestAccArchiveFile_Basic(t *testing.T) {
	var fileSize string
	r.Test(t, r.TestCase{
		Providers: testProviders,
		CheckDestroy: r.ComposeTestCheckFunc(
			testAccArchiveFileMissing("zip_file_acc_test.zip"),
		),
		Steps: []r.TestStep{
			r.TestStep{
				Config: testAccArchiveFileContentConfig,
				Check: r.ComposeTestCheckFunc(
					testAccArchiveFileExists("zip_file_acc_test.zip", &fileSize),
					r.TestCheckResourceAttrPtr("archive_file.foo", "output_size", &fileSize),
				),
			},
			r.TestStep{
				Config: testAccArchiveFileFileConfig,
				Check: r.ComposeTestCheckFunc(
					testAccArchiveFileExists("zip_file_acc_test.zip", &fileSize),
					r.TestCheckResourceAttrPtr("archive_file.foo", "output_size", &fileSize),
				),
			},
			r.TestStep{
				Config: testAccArchiveFileDirConfig,
				Check: r.ComposeTestCheckFunc(
					testAccArchiveFileExists("zip_file_acc_test.zip", &fileSize),
					r.TestCheckResourceAttrPtr("archive_file.foo", "output_size", &fileSize),
				),
			},
		},
	})
}

func testAccArchiveFileExists(filename string, fileSize *string) r.TestCheckFunc {
	return func(s *terraform.State) error {
		*fileSize = ""
		fi, err := os.Stat(filename)
		if err != nil {
			return err
		}
		*fileSize = fmt.Sprintf("%d", fi.Size())
		return nil
	}
}

func testAccArchiveFileMissing(filename string) r.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := os.Stat(filename)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		return fmt.Errorf("found file expected to be deleted: %s", filename)
	}
}

var testAccArchiveFileContentConfig = `
resource "archive_file" "foo" {
  type                    = "zip"
  source_content          = "This is some content"
  source_content_filename = "content.txt"
  output_path             = "zip_file_acc_test.zip"
}
`

var testAccArchiveFileFileConfig = `
resource "archive_file" "foo" {
  type        = "zip"
  source_file = "test-fixtures/test-file.txt"
  output_path = "zip_file_acc_test.zip"
}
`

var testAccArchiveFileDirConfig = `
resource "archive_file" "foo" {
  type        = "zip"
  source_dir  = "test-fixtures/test-dir"
  output_path = "zip_file_acc_test.zip"
}
`
