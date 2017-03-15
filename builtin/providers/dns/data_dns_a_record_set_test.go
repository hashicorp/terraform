package dns

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
)

func TestAccDnsARecordSet_Basic(t *testing.T) {
	tests := []struct {
		DataSourceBlock string
		DataSourceName  string
		Expected        []string
	}{
		{
			`
			data "dns_a_record_set" "foo" {
			  host = "127.0.0.1.nip.io"
			}
			`,
			"foo",
			[]string{
				"127.0.0.1",
			},
		},
		{
			`
			data "dns_a_record_set" "ntp" {
			  host = "time-c.nist.gov"
			}
			`,
			"ntp",
			[]string{
				"129.6.15.30",
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
						testCheckAttrStringArray(fmt.Sprintf("data.dns_a_record_set.%s", test.DataSourceName), "addrs", test.Expected),
					),
				},
			},
		})
	}
}
