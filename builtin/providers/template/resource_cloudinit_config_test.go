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
			`resource "template_cloudinit_config" "foo" {
				gzip = false
				base64_encode = false

				part {
					content_type = "text/x-shellscript"
					content = "baz"
				}
			}`,
			"Content-Type: multipart/mixed; boundary=\"MIMEBOUNDRY\"\n--MIMEBOUNDRY\r\nContent-Type: text/x-shellscript\r\n\r\nbaz\r\n--MIMEBOUNDRY--\r\n",
		},
		{
			`resource "template_cloudinit_config" "foo" {
				gzip = false
				base64_encode = false

				part {
					content_type = "text/x-shellscript"
					content = "baz"
					filename = "foobar.sh"
				}
			}`,
			"Content-Type: multipart/mixed; boundary=\"MIMEBOUNDRY\"\n--MIMEBOUNDRY\r\nContent-Type: text/x-shellscript\r\nContent-Disposition: attachment; filename=\"foobar.sh\"\r\n\r\nbaz\r\n--MIMEBOUNDRY--\r\n",
		},
		{
			`resource "template_cloudinit_config" "foo" {
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
			"Content-Type: multipart/mixed; boundary=\"MIMEBOUNDRY\"\n--MIMEBOUNDRY\r\nContent-Type: text/x-shellscript\r\n\r\nbaz\r\n--MIMEBOUNDRY\r\nContent-Type: text/x-shellscript\r\n\r\nffbaz\r\n--MIMEBOUNDRY--\r\n",
		},
		{
			`resource "template_cloudinit_config" "foo" {
				gzip = true
				base64_encode = false

				part {
					content_type = "text/x-shellscript"
					content = "baz"
					filename = "ah"
				}
				part {
					content_type = "text/x-shellscript"
					content = "ffbaz"
				}
			}`,
			"\x1f\x8b\b\x00\x00\tn\x88\x00\xff\x94\x8d\xbd\n\xc2@\x10\x84\xfb\x83{\x87\xe3\xfa%}B\x1a\x8d\x85E\x14D\v\xcbM\xb2!\v\xf7Gn\x03\x89O\xaf\x9d\x8a\x95\xe5\f3߷\x8fA(\b\\\xb7D\xa5\xf1\x8b\x13N8K\xe1y\xa5\xa12]\\\u0080\xf3V\xdb\xf6\xd8\x1ev\xe7۩\xb9ܭ\x02\xf8\x88Z}C\x84V)V\xc8\x139\x97\xfb\x99\x93\xbc\x17\r\xe7\x143\v\xc7P\x1a\x14\xc1~\xf2\xaf\xbe2#;\n詶8Y\xad\xb4\xea\xf0\xa1\xff\xf7h5\x8e\xbfO\x00\xad\x9e\x01\x00\x00\xff\xff\xecM\xd3\x1e\xe9\x00\x00\x00",
		},
	}

	for _, tt := range testCases {
		r.Test(t, r.TestCase{
			Providers: testProviders,
			Steps: []r.TestStep{
				r.TestStep{
					Config: tt.ResourceBlock,
					Check: r.ComposeTestCheckFunc(
						r.TestCheckResourceAttr("template_cloudinit_config.foo", "rendered", tt.Expected),
					),
				},
			},
		})
	}
}
