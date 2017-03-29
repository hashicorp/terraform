package opc

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/oracle/terraform-provider-compute/sdk/compute"
	"log"
)

func resourceSecurityRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceSecurityRuleCreate,
		Read:   resourceSecurityRuleRead,
		Update: resourceSecurityRuleUpdate,
		Delete: resourceSecurityRuleDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"source_list": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"destination_list": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"application": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"action": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"disabled": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
				ForceNew: false,
			},
		},
	}
}

func resourceSecurityRuleCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())

	name, sourceList, destinationList, application, action, disabled := getSecurityRuleResourceData(d)

	log.Printf("[DEBUG] Creating security list with name %s, sourceList %s, destinationList %s, application %s, action %s, disabled %s",
		name, sourceList, destinationList, application, action, disabled)

	client := meta.(*OPCClient).SecurityRules()
	info, err := client.CreateSecurityRule(name, sourceList, destinationList, application, action, disabled)
	if err != nil {
		return fmt.Errorf("Error creating security rule %s: %s", name, err)
	}

	d.SetId(info.Name)
	updateSecurityRuleResourceData(d, info)
	return nil
}

func updateSecurityRuleResourceData(d *schema.ResourceData, info *compute.SecurityRuleInfo) {
	d.Set("name", info.Name)
	d.Set("source_list", info.SourceList)
	d.Set("destination_list", info.DestinationList)
	d.Set("application", info.Application)
	d.Set("action", info.Action)
	d.Set("disabled", info.Disabled)
}

func resourceSecurityRuleRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*OPCClient).SecurityRules()
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Reading state of security rule %s", name)
	result, err := client.GetSecurityRule(name)
	if err != nil {
		// Security Rule does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading security list %s: %s", name, err)
	}

	log.Printf("[DEBUG] Read state of ssh key %s: %#v", name, result)
	updateSecurityRuleResourceData(d, result)
	return nil
}

func getSecurityRuleResourceData(d *schema.ResourceData) (string, string, string, string, string, bool) {
	return d.Get("name").(string),
		d.Get("source_list").(string),
		d.Get("destination_list").(string),
		d.Get("application").(string),
		d.Get("action").(string),
		d.Get("disabled").(bool)
}

func resourceSecurityRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())

	client := meta.(*OPCClient).SecurityRules()
	name, sourceList, destinationList, application, action, disabled := getSecurityRuleResourceData(d)

	log.Printf("[DEBUG] Updating security list %s with sourceList %s, destinationList %s, application %s, action %s, disabled %s",
		name, sourceList, destinationList, application, action, disabled)

	info, err := client.UpdateSecurityRule(name, sourceList, destinationList, application, action, disabled)
	if err != nil {
		return fmt.Errorf("Error updating security rule %s: %s", name, err)
	}

	updateSecurityRuleResourceData(d, info)
	return nil
}

func resourceSecurityRuleDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*OPCClient).SecurityRules()
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Deleting ssh key volume %s", name)
	if err := client.DeleteSecurityRule(name); err != nil {
		return fmt.Errorf("Error deleting security rule %s: %s", name, err)
	}
	return nil
}
