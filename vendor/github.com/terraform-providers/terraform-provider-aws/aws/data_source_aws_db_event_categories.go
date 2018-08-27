package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsDbEventCategories() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsDbEventCategoriesRead,

		Schema: map[string]*schema.Schema{
			"source_type": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"event_categories": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func dataSourceAwsDbEventCategoriesRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	req := &rds.DescribeEventCategoriesInput{}

	if sourceType := d.Get("source_type").(string); sourceType != "" {
		req.SourceType = aws.String(sourceType)
	}

	log.Printf("[DEBUG] Describe Event Categories %s\n", req)
	resp, err := conn.DescribeEventCategories(req)
	if err != nil {
		return err
	}

	if resp == nil || len(resp.EventCategoriesMapList) == 0 {
		return fmt.Errorf("Event Categories not found")
	}

	eventCategories := make([]string, 0)

	for _, eventMap := range resp.EventCategoriesMapList {
		for _, v := range eventMap.EventCategories {
			eventCategories = append(eventCategories, aws.StringValue(v))
		}
	}

	d.SetId(resource.UniqueId())
	if err := d.Set("event_categories", eventCategories); err != nil {
		return fmt.Errorf("Error setting Event Categories: %s", err)
	}

	return nil

}
