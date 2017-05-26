package dns

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataDnsARecordSet_Basic(t *testing.T) {
	tests := []struct {
		DataSourceBlock string
		DataSourceName  string
		Expected        []string
		Host            string
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
			"127.0.0.1.nip.io",
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
			"time-c.nist.gov",
		},
	}

	for _, test := range tests {
		recordName := fmt.Sprintf("data.dns_a_record_set.%s", test.DataSourceName)

		resource.Test(t, resource.TestCase{
			Providers: testAccProviders,
			Steps: []resource.TestStep{
				resource.TestStep{
					Config: test.DataSourceBlock,
					Check: resource.ComposeTestCheckFunc(
						testCheckAttrStringArray(recordName, "addrs", test.Expected),
					),
				},
				resource.TestStep{
					Config: test.DataSourceBlock,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(recordName, "id", test.Host),
					),
				},
			},
		})
	}

}
