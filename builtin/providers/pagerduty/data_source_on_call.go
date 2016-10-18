package pagerduty

import (
	"encoding/json"
	"log"
	"strconv"

	pagerduty "github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourcePagerDutyOnCall() *schema.Resource {
	return &schema.Resource{
		Read: dataSourcePagerDutyOnCallRead,

		Schema: map[string]*schema.Schema{
			"time_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"include": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"user_ids": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"escalation_policy_ids": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"schedule_ids": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"since": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"until": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"earliest": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"oncalls": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"escalation_level": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"start": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"end": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"user": &schema.Schema{
							Type:     schema.TypeMap,
							Computed: true,
						},
						"schedule": &schema.Schema{
							Type:     schema.TypeMap,
							Computed: true,
						},
						"escalation_policy": &schema.Schema{
							Type:     schema.TypeMap,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourcePagerDutyOnCallRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	o := &pagerduty.ListOnCallOptions{}

	if attr, ok := d.GetOk("time_zone"); ok {
		o.TimeZone = attr.(string)
	}

	if attr, ok := d.GetOk("include"); ok {
		includes := make([]string, 0, len(attr.([]interface{})))

		for _, include := range attr.([]interface{}) {
			includes = append(includes, include.(string))
		}

		o.Includes = includes
	}

	if attr, ok := d.GetOk("user_ids"); ok {
		userIDs := make([]string, 0, len(attr.([]interface{})))

		for _, user := range attr.([]interface{}) {
			userIDs = append(userIDs, user.(string))
		}

		o.UserIDs = userIDs
	}

	if attr, ok := d.GetOk("escalation_policy_ids"); ok {
		escalationPolicyIDs := make([]string, 0, len(attr.([]interface{})))

		for _, escalationPolicy := range attr.([]interface{}) {
			escalationPolicyIDs = append(escalationPolicyIDs, escalationPolicy.(string))
		}

		o.EscalationPolicyIDs = escalationPolicyIDs
	}

	if attr, ok := d.GetOk("since"); ok {
		o.Since = attr.(string)
	}

	if attr, ok := d.GetOk("until"); ok {
		o.Until = attr.(string)
	}

	if attr, ok := d.GetOk("earliest"); ok {
		o.Earliest = attr.(bool)
	}

	log.Printf("[INFO] Reading On Calls with options: %v", *o)

	resp, err := client.ListOnCalls(*o)

	if err != nil {
		return err
	}

	data := flattenOnCalls(resp.OnCalls)
	id, err := json.Marshal(data)

	if err != nil {
		return err
	}

	d.SetId(strconv.Itoa(hashcode.String(string(id))))

	if err := d.Set("oncalls", data); err != nil {
		return err
	}

	return nil

}
