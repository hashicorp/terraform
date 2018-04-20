package aws

import (
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/hashicorp/terraform/helper/schema"
)

// See http://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/billing-getting-started.html#step-2
var billingAccountId = "386209384616"

func dataSourceAwsBillingServiceAccount() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsBillingServiceAccountRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsBillingServiceAccountRead(d *schema.ResourceData, meta interface{}) error {
	d.SetId(billingAccountId)
	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "iam",
		AccountID: billingAccountId,
		Resource:  "root",
	}.String()
	d.Set("arn", arn)

	return nil
}
