package aws

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsAutoscalingGroups() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsAutoscalingGroupsRead,

		Schema: map[string]*schema.Schema{
			"names": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceAwsAutoscalingGroupsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn

	log.Printf("[DEBUG] Reading Autoscaling Groups.")
	d.SetId(time.Now().UTC().String())

	resp, err := conn.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		return fmt.Errorf("Error fetching Autoscaling Groups: %s", err)
	}

	raw := make([]string, len(resp.AutoScalingGroups))
	for i, v := range resp.AutoScalingGroups {
		raw[i] = *v.AutoScalingGroupName
	}

	sort.Strings(raw)

	if err := d.Set("names", raw); err != nil {
		return fmt.Errorf("[WARN] Error setting Autoscaling Group Names: %s", err)
	}

	return nil

}
