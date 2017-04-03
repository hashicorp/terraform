package opc

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/oracle/terraform-provider-compute/sdk/compute"
	"log"
)

func resourceSecurityIPList() *schema.Resource {
	return &schema.Resource{
		Create: resourceSecurityIPListCreate,
		Read:   resourceSecurityIPListRead,
		Update: resourceSecurityIPListUpdate,
		Delete: resourceSecurityIPListDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ip_entries": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceSecurityIPListCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())

	name, ipEntries := getSecurityIPListResourceData(d)

	log.Printf("[DEBUG] Creating security IP list with name %s, entries %s",
		name, ipEntries)

	client := meta.(*OPCClient).SecurityIPLists()
	info, err := client.CreateSecurityIPList(name, ipEntries)
	if err != nil {
		return fmt.Errorf("Error creating security IP list %s: %s", name, err)
	}

	d.SetId(info.Name)
	updateSecurityIPListResourceData(d, info)
	return nil
}

func updateSecurityIPListResourceData(d *schema.ResourceData, info *compute.SecurityIPListInfo) {
	d.Set("name", info.Name)
	d.Set("entries", info.SecIPEntries)
}

func resourceSecurityIPListRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*OPCClient).SecurityIPLists()
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Reading state of security IP list %s", name)
	result, err := client.GetSecurityIPList(name)
	if err != nil {
		// Security IP List does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading security IP list %s: %s", name, err)
	}

	log.Printf("[DEBUG] Read state of security IP list %s: %#v", name, result)
	updateSecurityIPListResourceData(d, result)
	return nil
}

func getSecurityIPListResourceData(d *schema.ResourceData) (string, []string) {
	name := d.Get("name").(string)
	ipEntries := d.Get("ip_entries").([]interface{})
	ipEntryStrings := []string{}
	for _, entry := range ipEntries {
		ipEntryStrings = append(ipEntryStrings, entry.(string))
	}
	return name, ipEntryStrings
}

func resourceSecurityIPListUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())

	client := meta.(*OPCClient).SecurityIPLists()
	name, entries := getSecurityIPListResourceData(d)

	log.Printf("[DEBUG] Updating security IP list %s with ip entries %s",
		name, entries)

	info, err := client.UpdateSecurityIPList(name, entries)
	if err != nil {
		return fmt.Errorf("Error updating security IP list %s: %s", name, err)
	}

	updateSecurityIPListResourceData(d, info)
	return nil
}

func resourceSecurityIPListDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*OPCClient).SecurityIPLists()
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Deleting security IP list %s", name)
	if err := client.DeleteSecurityIPList(name); err != nil {
		return fmt.Errorf("Error deleting security IP list %s: %s", name, err)
	}
	return nil
}
