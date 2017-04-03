package opc

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/oracle/terraform-provider-compute/sdk/compute"
	"log"
)

func resourceSecurityList() *schema.Resource {
	return &schema.Resource{
		Create: resourceSecurityListCreate,
		Read:   resourceSecurityListRead,
		Update: resourceSecurityListUpdate,
		Delete: resourceSecurityListDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"outbound_cidr_policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
		},
	}
}

func resourceSecurityListCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())

	name, policy, outboundCIDRPolicy := getSecurityListResourceData(d)

	log.Printf("[DEBUG] Creating security list with name %s, policy %s, outbound CIDR policy %s",
		name, policy, outboundCIDRPolicy)

	client := meta.(*OPCClient).SecurityLists()
	info, err := client.CreateSecurityList(name, policy, outboundCIDRPolicy)
	if err != nil {
		return fmt.Errorf("Error creating security list %s: %s", name, err)
	}

	d.SetId(info.Name)
	updateSecurityListResourceData(d, info)
	return nil
}

func updateSecurityListResourceData(d *schema.ResourceData, info *compute.SecurityListInfo) {
	d.Set("name", info.Name)
	d.Set("policy", info.Policy)
	d.Set("outbound_cidr_policy", info.OutboundCIDRPolicy)
}

func resourceSecurityListRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*OPCClient).SecurityLists()
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Reading state of security list %s", name)
	result, err := client.GetSecurityList(name)
	if err != nil {
		// Security List does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading security list %s: %s", name, err)
	}

	log.Printf("[DEBUG] Read state of ssh key %s: %#v", name, result)
	updateSecurityListResourceData(d, result)
	return nil
}

func getSecurityListResourceData(d *schema.ResourceData) (string, string, string) {
	return d.Get("name").(string),
		d.Get("policy").(string),
		d.Get("outbound_cidr_policy").(string)
}

func resourceSecurityListUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())

	client := meta.(*OPCClient).SecurityLists()
	name, policy, outboundCIDRPolicy := getSecurityListResourceData(d)

	log.Printf("[DEBUG] Updating security list %s with policy %s, outbound_cidr_policy %s",
		name, policy, outboundCIDRPolicy)

	info, err := client.UpdateSecurityList(name, policy, outboundCIDRPolicy)
	if err != nil {
		return fmt.Errorf("Error updating security list %s: %s", name, err)
	}

	updateSecurityListResourceData(d, info)
	return nil
}

func resourceSecurityListDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*OPCClient).SecurityLists()
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Deleting ssh key volume %s", name)
	if err := client.DeleteSecurityList(name); err != nil {
		return fmt.Errorf("Error deleting security list %s: %s", name, err)
	}
	return nil
}
