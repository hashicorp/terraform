package rackspace

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	osSecGroupRules "github.com/rackspace/gophercloud/openstack/networking/v2/extensions/security/rules"
	rsSecGroupRules "github.com/rackspace/gophercloud/rackspace/networking/v2/security/rules"
)

func resourceNetworkingSecGroupRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeSecGroupRuleCreate,
		Read:   resourceComputeSecGroupRuleRead,
		Delete: resourceComputeSecGroupRuleDelete,

		Schema: map[string]*schema.Schema{
			"direction": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ether_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"sec_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"port_range_max": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"port_range_min": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"remote_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"remote_ip_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceComputeSecGroupRuleCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace networking client: %s", err)
	}

	createOpts := osSecGroupRules.CreateOpts{
		Direction:      d.Get("direction").(string),
		EtherType:      d.Get("ether_type").(string),
		SecGroupID:     d.Get("sec_group_id").(string),
		PortRangeMax:   d.Get("port_range_max").(int),
		PortRangeMin:   d.Get("port_range_min").(int),
		Protocol:       d.Get("protocol").(string),
		RemoteGroupID:  d.Get("remote_group_id").(string),
		RemoteIPPrefix: d.Get("remote_ip_prefix").(string),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	sg, err := rsSecGroupRules.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating Rackspace security group rule: %s", err)
	}

	d.SetId(sg.ID)

	return resourceComputeSecGroupRuleRead(d, meta)
}

func resourceComputeSecGroupRuleRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace networking client: %s", err)
	}

	sgr, err := rsSecGroupRules.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "security group rule")
	}

	d.Set("direction", sgr.Direction)
	d.Set("ether_type", sgr.EtherType)
	d.Set("sec_group_id", sgr.SecGroupID)
	d.Set("port_range_max", sgr.PortRangeMax)
	d.Set("port_range_min", sgr.PortRangeMin)
	d.Set("protocol", sgr.Protocol)
	d.Set("remote_group_id", sgr.RemoteGroupID)
	d.Set("remote_ip_prefix", sgr.RemoteIPPrefix)

	return nil
}

func resourceComputeSecGroupRuleDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace networking client: %s", err)
	}

	err = rsSecGroupRules.Delete(networkingClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting Rackspace security group rule: %s", err)
	}
	d.SetId("")
	return nil
}
