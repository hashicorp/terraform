package aws

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
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
			"filter": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"values": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},
		},
	}
}

func dataSourceAwsAutoscalingGroupsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn

	log.Printf("[DEBUG] Reading Autoscaling Groups.")
	d.SetId(time.Now().UTC().String())

	var raw []string

	tf := d.Get("filter").(*schema.Set)
	if tf.Len() > 0 {
		out, err := conn.DescribeTags(&autoscaling.DescribeTagsInput{
			Filters: expandAsgTagFilters(tf.List()),
		})
		if err != nil {
			return err
		}

		raw = make([]string, len(out.Tags))
		for i, v := range out.Tags {
			raw[i] = *v.ResourceId
		}
	} else {

		resp, err := conn.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{})
		if err != nil {
			return fmt.Errorf("Error fetching Autoscaling Groups: %s", err)
		}

		raw = make([]string, len(resp.AutoScalingGroups))
		for i, v := range resp.AutoScalingGroups {
			raw[i] = *v.AutoScalingGroupName
		}
	}

	sort.Strings(raw)

	if err := d.Set("names", raw); err != nil {
		return fmt.Errorf("[WARN] Error setting Autoscaling Group Names: %s", err)
	}

	return nil

}

func expandAsgTagFilters(in []interface{}) []*autoscaling.Filter {
	out := make([]*autoscaling.Filter, len(in), len(in))
	for i, filter := range in {
		m := filter.(map[string]interface{})
		values := expandStringList(m["values"].(*schema.Set).List())

		out[i] = &autoscaling.Filter{
			Name:   aws.String(m["name"].(string)),
			Values: values,
		}
	}
	return out
}
