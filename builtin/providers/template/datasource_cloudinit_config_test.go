package template

import (
	"regexp"
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
			"Content-Type: multipart/mixed; boundary=\"MIMEBOUNDARY\"\nMIME-Version: 1.0\r\n\r\n--MIMEBOUNDARY\r\nContent-Transfer-Encoding: 7bit\r\nContent-Type: text/x-shellscript\r\nMime-Version: 1.0\r\n\r\nbaz\r\n--MIMEBOUNDARY--\r\n",
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
			"Content-Type: multipart/mixed; boundary=\"MIMEBOUNDARY\"\nMIME-Version: 1.0\r\n\r\n--MIMEBOUNDARY\r\nContent-Disposition: attachment; filename=\"foobar.sh\"\r\nContent-Transfer-Encoding: 7bit\r\nContent-Type: text/x-shellscript\r\nMime-Version: 1.0\r\n\r\nbaz\r\n--MIMEBOUNDARY--\r\n",
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
			"Content-Type: multipart/mixed; boundary=\"MIMEBOUNDARY\"\nMIME-Version: 1.0\r\n\r\n--MIMEBOUNDARY\r\nContent-Transfer-Encoding: 7bit\r\nContent-Type: text/x-shellscript\r\nMime-Version: 1.0\r\n\r\nbaz\r\n--MIMEBOUNDARY\r\nContent-Transfer-Encoding: 7bit\r\nContent-Type: text/x-shellscript\r\nMime-Version: 1.0\r\n\r\nffbaz\r\n--MIMEBOUNDARY--\r\n",
		},
	}

	for _, tt := range testCases {
		r.UnitTest(t, r.TestCase{
			Providers: testProviders,
			Steps: []r.TestStep{
				{
					Config: tt.ResourceBlock,
					Check: r.ComposeTestCheckFunc(
						r.TestCheckResourceAttr("data.template_cloudinit_config.foo", "rendered", tt.Expected),
					),
				},
			},
		})
	}
}

// From GH-13572, Correctly handle panic on a misconfigured cloudinit part
func TestRender_handlePanic(t *testing.T) {
	r.UnitTest(t, r.TestCase{
		Providers: testProviders,
		Steps: []r.TestStep{
			{
				Config:      testCloudInitConfig_misconfiguredParts,
				ExpectError: regexp.MustCompile("Unable to parse parts in cloudinit resource declaration"),
			},
		},
	})
}

var testCloudInitConfig_misconfiguredParts = `
data "template_cloudinit_config" "foo" {
  part {
    content = ""
  }
}
`
