package pagerduty

import (
	"log"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePagerDutyEscalationPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourcePagerDutyEscalationPolicyCreate,
		Read:   resourcePagerDutyEscalationPolicyRead,
		Update: resourcePagerDutyEscalationPolicyUpdate,
		Delete: resourcePagerDutyEscalationPolicyDelete,
		Importer: &schema.ResourceImporter{
			State: resourcePagerDutyEscalationPolicyImport,
		},
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},
			"num_loops": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"escalation_rule": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"escalation_delay_in_minutes": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"target": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
										Default:  "user_reference",
									},
									"id": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildEscalationPolicyStruct(d *schema.ResourceData) *pagerduty.EscalationPolicy {
	escalationRules := d.Get("escalation_rule").([]interface{})

	policy := pagerduty.EscalationPolicy{
		Name:            d.Get("name").(string),
		EscalationRules: expandRules(escalationRules),
	}

	if attr, ok := d.GetOk("description"); ok {
		policy.Description = attr.(string)
	}

	if attr, ok := d.GetOk("num_loops"); ok {
		policy.NumLoops = uint(attr.(int))
	}

	return &policy
}

func resourcePagerDutyEscalationPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	e := buildEscalationPolicyStruct(d)

	log.Printf("[INFO] Creating PagerDuty escalation policy: %s", e.Name)

	e, err := client.CreateEscalationPolicy(*e)

	if err != nil {
		return err
	}

	d.SetId(e.ID)

	return resourcePagerDutyEscalationPolicyRead(d, meta)
}

func resourcePagerDutyEscalationPolicyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty escalation policy: %s", d.Id())

	e, err := client.GetEscalationPolicy(d.Id(), &pagerduty.GetEscalationPolicyOptions{})

	if err != nil {
		return err
	}

	d.Set("name", e.Name)
	d.Set("description", e.Description)
	d.Set("num_loops", e.NumLoops)
	d.Set("escalation_rule", flattenRules(e.EscalationRules))

	return nil
}

func resourcePagerDutyEscalationPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	e := buildEscalationPolicyStruct(d)

	log.Printf("[INFO] Updating PagerDuty escalation policy: %s", d.Id())

	e, err := client.UpdateEscalationPolicy(d.Id(), e)

	if err != nil {
		return err
	}

	return nil
}

func resourcePagerDutyEscalationPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Deleting PagerDuty escalation policy: %s", d.Id())

	err := client.DeleteEscalationPolicy(d.Id())

	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func resourcePagerDutyEscalationPolicyImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := resourcePagerDutyEscalationPolicyRead(d, meta); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
