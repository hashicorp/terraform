package pagerduty

import (
	"fmt"
	"log"

	pagerduty "github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourcePagerDutyEscalationPolicy() *schema.Resource {
	return &schema.Resource{
		Read: dataSourcePagerDutyEscalationPolicyRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourcePagerDutyEscalationPolicyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty escalation policy")

	searchName := d.Get("name").(string)

	o := &pagerduty.ListEscalationPoliciesOptions{
		Query: searchName,
	}

	resp, err := client.ListEscalationPolicies(*o)
	if err != nil {
		return err
	}

	var found *pagerduty.EscalationPolicy

	for _, policy := range resp.EscalationPolicies {
		if policy.Name == searchName {
			found = &policy
			break
		}
	}

	if found == nil {
		return fmt.Errorf("Unable to locate any escalation policy with the name: %s", searchName)
	}

	d.SetId(found.ID)
	d.Set("name", found.Name)

	return nil
}
