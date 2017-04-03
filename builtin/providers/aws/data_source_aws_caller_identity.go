package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/service/sts"
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

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"user_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsCallerIdentityRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).stsconn

	res, err := client.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("Error getting Caller Identity: %v", err)
	}

	log.Printf("[DEBUG] Reading Caller Identity.")
	d.SetId(time.Now().UTC().String())

	if *res.Account == "" {
		log.Println("[DEBUG] No Account ID available, failing")
		return fmt.Errorf("No AWS Account ID is available to the provider. Please ensure that\n" +
			"skip_requesting_account_id is not set on the AWS provider.")
	}

	log.Printf("[DEBUG] Setting AWS Account ID to %s.", *res.Account)
	d.Set("account_id", res.Account)
	d.Set("arn", res.Arn)
	d.Set("user_id", res.UserId)
	return nil
}
