package dns

import (
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
)

func TestAccDnsCnameRecordSet_Basic(t *testing.T) {
	tests := []struct {
		DataSourceBlock string
		Expected        string
	}{
		{
			`
			data "dns_cname_record_set" "foo" {
			  host = "www.hashicorp.com"
			}
			`,
			"dualstack.s.shared.global.fastly.net.",
		},
	}

	for _, test := range tests {
		r.UnitTest(t, r.TestCase{
			Providers: testAccProviders,
			Steps: []r.TestStep{
				r.TestStep{
					Config: test.DataSourceBlock,
					Check: r.ComposeTestCheckFunc(
						r.TestCheckResourceAttr("data.dns_cname_record_set.foo", "cname", test.Expected),
					),
				},
			},
		})
	}
}
