package template

import (
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

type TestCloudInitData struct {
	ResourceBlock string
	Expected      string
	Base64        bool
	Gzip          bool
}

func TestRender(t *testing.T) {
	testCases := []TestCloudInitData{
		{
			`data "template_cloudinit_config" "foo" {
				gzip = false
				base64_encode = false

				part {
					content_type = "text/x-shellscript"
					content = "baz"
				}
			}`,
			"Content-Type: multipart/mixed; boundary=\"MIMEBOUNDARY\"\nMIME-Version: 1.0\r\n--MIMEBOUNDARY\r\nContent-Transfer-Encoding: 7bit\r\nContent-Type: text/x-shellscript\r\nMime-Version: 1.0\r\n\r\nbaz\r\n--MIMEBOUNDARY--\r\n",
			false,
			false,
		},
		{
			`data "template_cloudinit_config" "foo" {
				gzip = false
				base64_encode = false

				part {
					content_type = "text/x-shellscript"
					content = "baz"
					filename = "foobar.sh"
				}
			}`,
			"Content-Type: multipart/mixed; boundary=\"MIMEBOUNDARY\"\nMIME-Version: 1.0\r\n--MIMEBOUNDARY\r\nContent-Disposition: attachment; filename=\"foobar.sh\"\r\nContent-Transfer-Encoding: 7bit\r\nContent-Type: text/x-shellscript\r\nMime-Version: 1.0\r\n\r\nbaz\r\n--MIMEBOUNDARY--\r\n",
			false,
			false,
		},
		{
			`data "template_cloudinit_config" "foo" {
				gzip = false
				base64_encode = false

				part {
					content_type = "text/x-shellscript"
					content = "baz"
				}
				part {
					content_type = "text/x-shellscript"
					content = "ffbaz"
				}
			}`,
			"Content-Type: multipart/mixed; boundary=\"MIMEBOUNDARY\"\nMIME-Version: 1.0\r\n--MIMEBOUNDARY\r\nContent-Transfer-Encoding: 7bit\r\nContent-Type: text/x-shellscript\r\nMime-Version: 1.0\r\n\r\nbaz\r\n--MIMEBOUNDARY\r\nContent-Transfer-Encoding: 7bit\r\nContent-Type: text/x-shellscript\r\nMime-Version: 1.0\r\n\r\nffbaz\r\n--MIMEBOUNDARY--\r\n",
			false,
			false,
		},
		{
			`data "template_cloudinit_config" "foo" {
				single_part = true
				base64_encode = false
				gzip = false
				part {
					content      = "baz"
				}
			}`,
			"baz",
			false,
			false,
		},
		{
			`data "template_cloudinit_config" "foo" {
				single_part = true
				base64_encode = true
				gzip = false
				part {
					content      = "baz"
				}
			}`,
			"baz",
			true,
			false,
		},
		{
			`data "template_cloudinit_config" "foo" {
				single_part = true
				base64_encode = true
				gzip = true
				part {
					content      = "baz"
				}
			}`,
			"baz",
			true,
			true,
		},
		{
			`data "template_cloudinit_config" "foo" {
				single_part = true
				base64_encode = false
				gzip = true
				part {
					content      = "baz"
				}
			}`,
			"baz",
			false,
			true,
		},
	}

	for _, tt := range testCases {
		r.UnitTest(t, r.TestCase{
			Providers: testProviders,
			Steps: []r.TestStep{
				r.TestStep{
					Config: tt.ResourceBlock,
					Check:  testCheckResourceAttrEncoded("data.template_cloudinit_config.foo", "rendered", tt.Expected, tt.Base64, tt.Gzip),
				},
			},
		})
	}
}

func testCheckResourceAttrEncoded(name string, key string, value string, base64_decode bool, gzip_decode bool) r.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		is := rs.Primary
		if is == nil {
			return fmt.Errorf("No primary instance: %s", name)
		}

		var result string

		if base64_decode {
			temp, err := base64.StdEncoding.DecodeString(is.Attributes[key])
			if err != nil {
				return fmt.Errorf("Unable to decode base64 string")
			}
			result = string(temp)
		} else {
			result = is.Attributes[key]
		}

		if gzip_decode {
			gzipReader, err := gzip.NewReader(strings.NewReader(result))
			if err != nil {
				return fmt.Errorf("Unable to decompress gzip string")
			}
			gzipReader.Close()
			temp, err := ioutil.ReadAll(gzipReader)
			if err != nil {
				return fmt.Errorf("Unable read data from gzip reader")
			}
			result = string(temp)
		}

		if result != value {
			return fmt.Errorf(
				"%s: Attribute '%s' expected %#v, got %#v",
				name,
				key,
				value,
				result)
		}

		return nil
	}
}

var testCloudInitConfig_basic = `
data "template_cloudinit_config" "config" {
  part {
    content_type = "text/x-shellscript"
    content      = "baz"
  }
}`

var testCloudInitConfig_basic_expected = `Content-Type: multipart/mixed; boundary=\"MIMEBOUNDARY\"\nMIME-Version: 1.0\r\n--MIMEBOUNDARY\r\nContent-Transfer-Encoding: 7bit\r\nContent-Type: text/x-shellscript\r\nMime-Version: 1.0\r\n\r\nbaz\r\n--MIMEBOUNDARY--\r\n`
