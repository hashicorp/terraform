package opc

import (
	"fmt"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceOPCSecRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceOPCSecRuleCreate,
		Read:   resourceOPCSecRuleRead,
		Update: resourceOPCSecRuleUpdate,
		Delete: resourceOPCSecRuleDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"source_list": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"destination_list": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"application": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"action": {
				Type:     schema.TypeString,
				Required: true,
			},
			"disabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceOPCSecRuleCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).SecRules()

	name := d.Get("name").(string)
	sourceList := d.Get("source_list").(string)
	destinationList := d.Get("destination_list").(string)
	application := d.Get("application").(string)
	action := d.Get("action").(string)
	disabled := d.Get("disabled").(bool)

	input := compute.CreateSecRuleInput{
		Name:            name,
		Action:          action,
		SourceList:      sourceList,
		DestinationList: destinationList,
		Disabled:        disabled,
		Application:     application,
	}
	desc, descOk := d.GetOk("description")
	if descOk {
		input.Description = desc.(string)
	}

	info, err := client.CreateSecRule(&input)
	if err != nil {
		return fmt.Errorf("Error creating sec rule %s: %s", name, err)
	}

	d.SetId(info.Name)

	return resourceOPCSecRuleRead(d, meta)
}

func resourceOPCSecRuleRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).SecRules()

	name := d.Id()

	input := compute.GetSecRuleInput{
		Name: name,
	}
	result, err := client.GetSecRule(&input)
	if err != nil {
		// Sec Rule does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading sec list %s: %s", name, err)
	}

	d.Set("name", result.Name)
	d.Set("description", result.Description)
	d.Set("source_list", result.SourceList)
	d.Set("destination_list", result.DestinationList)
	d.Set("application", result.Application)
	d.Set("action", result.Action)
	d.Set("disabled", result.Disabled)

	return nil
}

func resourceOPCSecRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).SecRules()

	name := d.Get("name").(string)
	sourceList := d.Get("source_list").(string)
	destinationList := d.Get("destination_list").(string)
	application := d.Get("application").(string)
	action := d.Get("action").(string)
	disabled := d.Get("disabled").(bool)

	input := compute.UpdateSecRuleInput{
		Action:          action,
		Application:     application,
		DestinationList: destinationList,
		Disabled:        disabled,
		Name:            name,
		SourceList:      sourceList,
	}
	desc, descOk := d.GetOk("description")
	if descOk {
		input.Description = desc.(string)
	}

	_, err := client.UpdateSecRule(&input)
	if err != nil {
		return fmt.Errorf("Error updating sec rule %s: %s", name, err)
	}

	return resourceOPCSecRuleRead(d, meta)
}

func resourceOPCSecRuleDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).SecRules()
	name := d.Id()

	input := compute.DeleteSecRuleInput{
		Name: name,
	}
	if err := client.DeleteSecRule(&input); err != nil {
		return fmt.Errorf("Error deleting sec rule %s: %s", name, err)
	}

	return nil
}
