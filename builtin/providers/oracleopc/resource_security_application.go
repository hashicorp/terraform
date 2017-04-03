package opc

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/oracle/terraform-provider-compute/sdk/compute"
	"log"
)

func resourceSecurityApplication() *schema.Resource {
	return &schema.Resource{
		Create: resourceSecurityApplicationCreate,
		Read:   resourceSecurityApplicationRead,
		Delete: resourceSecurityApplicationDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"dport": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"icmptype": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"icmpcode": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceSecurityApplicationCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())

	name, protocol, dport, icmptype, icmpcode, description := getSecurityApplicationResourceData(d)

	log.Printf("[DEBUG] Creating security application %s", name)

	client := meta.(*OPCClient).SecurityApplications()
	info, err := client.CreateSecurityApplication(name, protocol, dport, icmptype, icmpcode, description)
	if err != nil {
		return fmt.Errorf("Error creating security application %s: %s", name, err)
	}

	d.SetId(info.Name)
	updateSecurityApplicationResourceData(d, info)
	return nil
}

func updateSecurityApplicationResourceData(d *schema.ResourceData, info *compute.SecurityApplicationInfo) {
	d.Set("name", info.Name)
	d.Set("protocol", info.Protocol)
	d.Set("dport", info.DPort)
	d.Set("icmptype", info.ICMPType)
	d.Set("icmpcode", info.ICMPCode)
	d.Set("description", info.Description)
}

func resourceSecurityApplicationRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*OPCClient).SecurityApplications()
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Reading state of security application %s", name)
	result, err := client.GetSecurityApplication(name)
	if err != nil {
		// Security Application does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading security application %s: %s", name, err)
	}

	log.Printf("[DEBUG] Read state of security application %s: %#v", name, result)
	updateSecurityApplicationResourceData(d, result)
	return nil
}

func getSecurityApplicationResourceData(d *schema.ResourceData) (string, string, string, string, string, string) {
	return d.Get("name").(string),
		d.Get("protocol").(string),
		d.Get("dport").(string),
		d.Get("icmptype").(string),
		d.Get("icmpcode").(string),
		d.Get("description").(string)
}

func resourceSecurityApplicationDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*OPCClient).SecurityApplications()
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Deleting security application %s", name)

	if err := client.DeleteSecurityApplication(name); err != nil {
		return fmt.Errorf("Error deleting security application %s: %s", name, err)
	}
	return nil
}
