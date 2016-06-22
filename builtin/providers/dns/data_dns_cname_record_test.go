package dns

import (
	r "github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccDnsCnameRecord_Basic(t *testing.T) {
	tests := []struct {
		DataSourceBlock string
		Expected        string
	}{
		{
			`
			data "dns_cname_record" "foo" {
			  host = "www.hashicorp.com"
			}
			`,
			"prod.k.ssl.global.fastly.net.",
		},
	}

	for _, test := range tests {
		r.UnitTest(t, r.TestCase{
			Providers: testAccProviders,
			Steps: []r.TestStep{
				r.TestStep{
					Config: test.DataSourceBlock,
					Check: r.ComposeTestCheckFunc(
						r.TestCheckResourceAttr("data.dns_cname_record.foo", "cname", test.Expected),
					),
				},
			},
		})
	}
}
