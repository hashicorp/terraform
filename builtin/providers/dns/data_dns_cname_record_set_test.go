package dns

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDnsCnameRecordSet_Basic(t *testing.T) {
	tests := []struct {
		DataSourceBlock string
		Expected        string
		Host            string
	}{
		{
			`
			data "dns_cname_record_set" "foo" {
			  host = "www.hashicorp.com"
			}
			`,
			"dualstack.s.shared.global.fastly.net.",
			"www.hashicorp.com",
		},
	}

	for _, test := range tests {
		resource.Test(t, resource.TestCase{
			Providers: testAccProviders,
			Steps: []resource.TestStep{
				resource.TestStep{
					Config: test.DataSourceBlock,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("data.dns_cname_record_set.foo", "cname", test.Expected),
					),
				},
				resource.TestStep{
					Config: test.DataSourceBlock,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("data.dns_cname_record_set.foo", "id", test.Host),
					),
				},
			},
		})
	}
}
