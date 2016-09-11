package template

import (
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
)

func TestRender(t *testing.T) {
	testCases := []struct {
		ResourceBlock string
		Expected      string
	}{
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
		},
	}

	for _, tt := range testCases {
		r.UnitTest(t, r.TestCase{
			Providers: testProviders,
			Steps: []r.TestStep{
				r.TestStep{
					Config: tt.ResourceBlock,
					Check: r.ComposeTestCheckFunc(
						r.TestCheckResourceAttr("data.template_cloudinit_config.foo", "rendered", tt.Expected),
					),
				},
			},
		})
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
