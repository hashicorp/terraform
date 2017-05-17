package opc

import (
	"fmt"
	"log"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceOPCSecurityIPList() *schema.Resource {
	return &schema.Resource{
		Create: resourceOPCSecurityIPListCreate,
		Read:   resourceOPCSecurityIPListRead,
		Update: resourceOPCSecurityIPListUpdate,
		Delete: resourceOPCSecurityIPListDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ip_entries": {
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceOPCSecurityIPListCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*compute.Client).SecurityIPLists()

	ipEntries := d.Get("ip_entries").([]interface{})
	ipEntryStrings := []string{}
	for _, entry := range ipEntries {
		ipEntryStrings = append(ipEntryStrings, entry.(string))
	}

	input := compute.CreateSecurityIPListInput{
		Name:         d.Get("name").(string),
		SecIPEntries: ipEntryStrings,
	}
	if description, ok := d.GetOk("description"); ok {
		input.Description = description.(string)
	}

	log.Printf("[DEBUG] Creating security IP list with %+v", input)
	info, err := client.CreateSecurityIPList(&input)
	if err != nil {
		return fmt.Errorf("Error creating security IP list %s: %s", input.Name, err)
	}

	d.SetId(info.Name)
	return resourceOPCSecurityIPListRead(d, meta)
}

func resourceOPCSecurityIPListRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*compute.Client).SecurityIPLists()
	name := d.Id()

	log.Printf("[DEBUG] Reading state of security IP list %s", name)
	input := compute.GetSecurityIPListInput{
		Name: name,
	}
	result, err := client.GetSecurityIPList(&input)
	if err != nil {
		// Security IP List does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading security IP list %s: %s", name, err)
	}

	log.Printf("[DEBUG] Read state of security IP list %s: %#v", name, result)
	d.Set("name", result.Name)
	d.Set("ip_entries", result.SecIPEntries)
	d.Set("description", result.Description)
	return nil
}

func resourceOPCSecurityIPListUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())

	client := meta.(*compute.Client).SecurityIPLists()

	ipEntries := d.Get("ip_entries").([]interface{})
	ipEntryStrings := []string{}
	for _, entry := range ipEntries {
		ipEntryStrings = append(ipEntryStrings, entry.(string))
	}
	input := compute.UpdateSecurityIPListInput{
		Name:         d.Get("name").(string),
		SecIPEntries: ipEntryStrings,
	}
	if description, ok := d.GetOk("description"); ok {
		input.Description = description.(string)
	}

	log.Printf("[DEBUG] Updating security IP list with %+v", input)
	info, err := client.UpdateSecurityIPList(&input)
	if err != nil {
		return fmt.Errorf("Error updating security IP list %s: %s", input.Name, err)
	}
	d.SetId(info.Name)

	return resourceOPCSecurityIPListRead(d, meta)
}

func resourceOPCSecurityIPListDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*compute.Client).SecurityIPLists()
	name := d.Id()

	log.Printf("[DEBUG] Deleting security IP list %s", name)
	input := compute.DeleteSecurityIPListInput{
		Name: name,
	}
	if err := client.DeleteSecurityIPList(&input); err != nil {
		return fmt.Errorf("Error deleting security IP list %s: %s", name, err)
	}
	return nil
}
