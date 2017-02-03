package gpg

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

var testProviders = map[string]terraform.ResourceProvider{
	"gpg": Provider(),
}

func TestGPGRendering(t *testing.T) {
	var cases = []struct {
		ciphertext string
		keypath    string
		plaintext  string
	}{
		{`-----BEGIN PGP MESSAGE-----
Version: GnuPG v2

hQEMA576X39jwPWtAQf/ewn4dIw2APLjRBLU0f7BrR3jyK+zrXdxpbc5ZtVvheFC
Los3EfRmypAk7cfxp4yXvyegLdckVAlEj4EUBQxHmeP6iaPt9VW58OEnCYsTCPcc
P7prgM9Ms3u40ZNlp0TQzTAaHSC7HNHGZw0G5ra4mm5EaKH+YP/WPQu/bEP4U8X0
6I8HVqkQKHBBODDqtdhTeOQvARsWhOBRHHB2pzG533MkE9Ck4nnb/tA1LhgGFIHO
pPXbEFByuwCu/fjdBSCkVERO/g/l6Ji2anjxckTmjTeQfB+QFGJO6c12SqnJ7zGf
yuwJ3+OF2DFF9I+ri7FvUkdiQsyNNG/+Xcd7oqO2vdJGAXYPK1kxUoFUtphHSh2I
oCYSxZq4V6mAn93nfyhFpI/qb5eeUlZBLYxIYVqWHrQIEKU4HOSsQkV5aS5BzF6U
aKII3VUjpQ==
=porM
-----END PGP MESSAGE-----`,
			"test-fixtures",
			"supersecret"},
	}

	for _, tt := range cases {
		r.UnitTest(t, r.TestCase{
			Providers: testProviders,
			Steps: []r.TestStep{
				r.TestStep{
					Config: testTemplateConfig(tt.ciphertext, tt.keypath),
					Check: func(s *terraform.State) error {
						got := s.RootModule().Outputs["rendered"]
						if tt.plaintext != got.Value {
							return fmt.Errorf("template:\n%s\nvars:\n%s\ngot:\n%s\nwant:\n%s\n", tt.ciphertext, tt.keypath, got, tt.plaintext)
						}
						return nil
					},
				},
			},
		})
	}
}

func testTemplateConfig(ciphertext, keyDir string) string {
	return fmt.Sprintf(`
		data "gpg_message" "t0" {
			encrypted_data = <<EOF
%s
EOF
			key_directory = "%s"
		}
		output "rendered" {
				value = "${data.gpg_message.t0.decrypted_data}"
		}`, ciphertext, keyDir)
}
