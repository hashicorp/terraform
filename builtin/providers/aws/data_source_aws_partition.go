package aws

import (
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsPartition() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsPartitionRead,

		Schema: map[string]*schema.Schema{
			"partition": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsPartitionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient)

	log.Printf("[DEBUG] Reading Partition.")
	d.SetId(time.Now().UTC().String())

	log.Printf("[DEBUG] Setting AWS Partition to %s.", client.partition)
	d.Set("partition", meta.(*AWSClient).partition)

	return nil
}
