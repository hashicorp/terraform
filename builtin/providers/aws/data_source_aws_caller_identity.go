package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsCallerIdentity() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsCallerIdentityRead,

		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsCallerIdentityRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient)

	log.Printf("[DEBUG] Reading Caller Identity.")
	d.SetId(time.Now().UTC().String())

	if client.accountid == "" {
		log.Println("[DEBUG] No Account ID available, failing")
		return fmt.Errorf("No AWS Account ID is available to the provider. Please ensure that\n" +
			"skip_requesting_account_id is not set on the AWS provider.")
	}

	log.Printf("[DEBUG] Setting AWS Account ID to %s.", client.accountid)
	d.Set("account_id", meta.(*AWSClient).accountid)

	return nil
}
