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
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},
			"num_loops": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"teams": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"rule": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"escalation_delay_in_minutes": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"target": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Optional: true,
										Default:  "user_reference",
									},
									"id": {
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
	escalationRules := d.Get("rule").([]interface{})

	escalationPolicy := pagerduty.EscalationPolicy{
		Name:            d.Get("name").(string),
		EscalationRules: expandEscalationRules(escalationRules),
	}

	if attr, ok := d.GetOk("description"); ok {
		escalationPolicy.Description = attr.(string)
	}

	if attr, ok := d.GetOk("num_loops"); ok {
		escalationPolicy.NumLoops = uint(attr.(int))
	}

	if attr, ok := d.GetOk("teams"); ok {
		escalationPolicy.Teams = expandTeams(attr.([]interface{}))
	}

	return &escalationPolicy
}

func resourcePagerDutyEscalationPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	escalationPolicy := buildEscalationPolicyStruct(d)

	log.Printf("[INFO] Creating PagerDuty escalation policy: %s", escalationPolicy.Name)

	escalationPolicy, err := client.CreateEscalationPolicy(*escalationPolicy)

	if err != nil {
		return err
	}

	d.SetId(escalationPolicy.ID)

	return resourcePagerDutyEscalationPolicyRead(d, meta)
}

func resourcePagerDutyEscalationPolicyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty escalation policy: %s", d.Id())

	o := &pagerduty.GetEscalationPolicyOptions{}

	escalationPolicy, err := client.GetEscalationPolicy(d.Id(), o)

	if err != nil {
		return err
	}

	d.Set("name", escalationPolicy.Name)
	d.Set("teams", escalationPolicy.Teams)
	d.Set("description", escalationPolicy.Description)
	d.Set("num_loops", escalationPolicy.NumLoops)

	if err := d.Set("rule", flattenEscalationRules(escalationPolicy.EscalationRules)); err != nil {
		return err
	}

	return nil
}

func resourcePagerDutyEscalationPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	escalationPolicy := buildEscalationPolicyStruct(d)

	log.Printf("[INFO] Updating PagerDuty escalation policy: %s", d.Id())

	if _, err := client.UpdateEscalationPolicy(d.Id(), escalationPolicy); err != nil {
		return err
	}

	return nil
}

func resourcePagerDutyEscalationPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Deleting PagerDuty escalation policy: %s", d.Id())

	if err := client.DeleteEscalationPolicy(d.Id()); err != nil {
		return err
	}

	d.SetId("")

	return nil
}
