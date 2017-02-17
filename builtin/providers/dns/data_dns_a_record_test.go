package dns

import (
	r "github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccDnsARecord_Basic(t *testing.T) {
	tests := []struct {
		DataSourceBlock string
		Expected        []string
	}{
		{
			`
			data "dns_a_record" "foo" {
			  host = "127.0.0.1.xip.io"
			}
			`,
			[]string{
				"127.0.0.1",
			},
		},
	}

	for _, test := range tests {
		r.UnitTest(t, r.TestCase{
			Providers: testAccProviders,
			Steps: []r.TestStep{
				r.TestStep{
					Config: test.DataSourceBlock,
					Check: r.ComposeTestCheckFunc(
						testCheckAttrStringArray("data.dns_a_record.foo", "addrs", test.Expected),
					),
				},
			},
		})
	}
}
