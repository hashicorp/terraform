package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsCanonicalUserId() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsCanonicalUserIdRead,

		Schema: map[string]*schema.Schema{
			"display_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsCanonicalUserIdRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).s3conn

	log.Printf("[DEBUG] Listing S3 buckets.")

	req := &s3.ListBucketsInput{}
	resp, err := conn.ListBuckets(req)
	if err != nil {
		return err
	}
	if resp == nil || resp.Owner == nil {
		return fmt.Errorf("no canonical user ID found")
	}

	d.SetId(aws.StringValue(resp.Owner.ID))
	d.Set("display_name", resp.Owner.DisplayName)

	return nil
}
