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
			"Content-Type: multipart/mixed; boundary=\"MIMEBOUNDRY\"\nMIME-Version: 1.0\r\n--MIMEBOUNDRY\r\nContent-Transfer-Encoding: 7bit\r\nContent-Type: text/x-shellscript\r\nMime-Version: 1.0\r\n\r\nbaz\r\n--MIMEBOUNDRY--\r\n",
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
			"Content-Type: multipart/mixed; boundary=\"MIMEBOUNDRY\"\nMIME-Version: 1.0\r\n--MIMEBOUNDRY\r\nContent-Disposition: attachment; filename=\"foobar.sh\"\r\nContent-Transfer-Encoding: 7bit\r\nContent-Type: text/x-shellscript\r\nMime-Version: 1.0\r\n\r\nbaz\r\n--MIMEBOUNDRY--\r\n",
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
			"Content-Type: multipart/mixed; boundary=\"MIMEBOUNDRY\"\nMIME-Version: 1.0\r\n--MIMEBOUNDRY\r\nContent-Transfer-Encoding: 7bit\r\nContent-Type: text/x-shellscript\r\nMime-Version: 1.0\r\n\r\nbaz\r\n--MIMEBOUNDRY\r\nContent-Transfer-Encoding: 7bit\r\nContent-Type: text/x-shellscript\r\nMime-Version: 1.0\r\n\r\nffbaz\r\n--MIMEBOUNDRY--\r\n",
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
			"\x1f\x8b\b\x00\x00\tn\x88\x00\xff\xac\xce\xc1J\x031\x10\xc6\xf1{`\xdf!\xe4>VO\u0096^\xb4=xX\x05\xa9\x82\xc7\xd9݉;\x90LB2\x85\xadOo-\x88\x8b\xe2\xadǄ\x1f\xf3\xfd\xef\x93(\x89\xc2\xfe\x98\xa9\xb5\xf1\x10\x943\x16]E\x9ei\\\xdb>\x1dd\xc4rܸ\xee\xa1\xdb\xdd=\xbd<n\x9fߜ\xf9z\xc0+\x95\xcaIZ{su\xdd\x18\x80\x85h\xcc\xf7\xdd-ל*\xeb\x19\xa2*\x0eS<\xfd\xaf\xad\xe7@\x82\x916\x0e'\xf7\xe3\xf7\x05\xa5z*\xb0\x93!\x8d,ﭽ\xedY\x17\xe0\x1c\xaa4\xebj\x86:Q\bu(\x9cO\xa2\xe3H\xbf\xa2\x1a\xd3\xe3ǿm\x97\xde\xf2\xfe\xef\x1a@c>\x03\x00\x00\xff\xffmB\x8c\xeed\x01\x00\x00",
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
