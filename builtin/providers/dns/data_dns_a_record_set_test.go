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

	var steps []resource.TestStep

	for _, test := range tests {
		ts := resource.TestStep{
			Config: test.DataSourceBlock,
			Check: resource.ComposeTestCheckFunc(
				testCheckAttrStringArray(fmt.Sprintf("data.dns_a_record_set.%s", test.DataSourceName), "addrs", test.Expected),
			),
		}
		steps = append(steps, ts)
	}

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps:     steps,
	})
}
