package dns

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataDnsTxtRecordSet_Basic(t *testing.T) {
	tests := []struct {
		DataSourceBlock string
		DataSourceName  string
		Expected        []string
		Host            string
	}{
		{
			`
			data "dns_txt_record_set" "foo" {
			  host = "hashicorp.com"
			}
			`,
			"foo",
			[]string{
				"google-site-verification=oqoe6Z7OB_726BNm33g4OdKK57KDtCfH266f8wAvLBo",
				"v=spf1 include:_spf.google.com include:spf.mail.intercom.io  include:stspg-customer.com include:mail.zendesk.com ~all",
				"status-page-domain-verification=dgtdvzlp8tfn",
			},
			"hashicorp.com",
		},
	}

	for _, test := range tests {
		recordName := fmt.Sprintf("data.dns_txt_record_set.%s", test.DataSourceName)
		resource.UnitTest(t, resource.TestCase{
			Providers: testAccProviders,
			Steps: []resource.TestStep{
				resource.TestStep{
					Config: test.DataSourceBlock,
					Check: resource.ComposeTestCheckFunc(
						testCheckAttrStringArray(recordName, "records", test.Expected),
					),
				},
				resource.TestStep{
					Config: test.DataSourceBlock,
					Check: resource.ComposeTestCheckFunc(
						testCheckAttrStringArrayMember(recordName, "record", test.Expected),
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
