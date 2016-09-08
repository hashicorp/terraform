package aws

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// See http://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/billing-getting-started.html#step-2
var billingAccountId = "386209384616"

func dataSourceAwsBillingServiceAccount() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsBillingServiceAccountRead,

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsBillingServiceAccountRead(d *schema.ResourceData, meta interface{}) error {
	d.SetId(billingAccountId)

	d.Set("arn", "arn:aws:iam::"+billingAccountId+":root")

	return nil
}
