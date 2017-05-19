package opc

import (
	"fmt"
	"log"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceOPCACL() *schema.Resource {
	return &schema.Resource{
		Create: resourceOPCACLCreate,
		Read:   resourceOPCACLRead,
		Update: resourceOPCACLUpdate,
		Delete: resourceOPCACLDelete,
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
			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"tags": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				ForceNew: true,
			},
			"uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceOPCACLCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())

	log.Print("[DEBUG] Creating acl")

	client := meta.(*compute.Client).ACLs()
	input := compute.CreateACLInput{
		Name:    d.Get("name").(string),
		Enabled: d.Get("enabled").(bool),
	}

	tags := getStringList(d, "tags")
	if len(tags) != 0 {
		input.Tags = tags
	}

	if description, ok := d.GetOk("description"); ok {
		input.Description = description.(string)
	}

	info, err := client.CreateACL(&input)
	if err != nil {
		return fmt.Errorf("Error creating ACL: %s", err)
	}

	d.SetId(info.Name)
	return resourceOPCACLRead(d, meta)
}

func resourceOPCACLRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*compute.Client).ACLs()

	log.Printf("[DEBUG] Reading state of ip reservation %s", d.Id())
	getInput := compute.GetACLInput{
		Name: d.Id(),
	}
	result, err := client.GetACL(&getInput)
	if err != nil {
		// ACL does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading acl %s: %s", d.Id(), err)
	}

	log.Printf("[DEBUG] Read state of acl %s: %#v", d.Id(), result)
	d.Set("name", result.Name)
	d.Set("enabled", result.Enabled)
	d.Set("description", result.Description)
	d.Set("uri", result.URI)
	if err := setStringList(d, "tags", result.Tags); err != nil {
		return err
	}
	return nil
}

func resourceOPCACLUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())

	log.Print("[DEBUG] Updating acl")

	client := meta.(*compute.Client).ACLs()
	input := compute.UpdateACLInput{
		Name:    d.Get("name").(string),
		Enabled: d.Get("enabled").(bool),
	}

	tags := getStringList(d, "tags")
	if len(tags) != 0 {
		input.Tags = tags
	}

	if description, ok := d.GetOk("description"); ok {
		input.Description = description.(string)
	}

	info, err := client.UpdateACL(&input)
	if err != nil {
		return fmt.Errorf("Error updating ACL: %s", err)
	}

	d.SetId(info.Name)
	return resourceOPCACLRead(d, meta)
}

func resourceOPCACLDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*compute.Client).ACLs()
	name := d.Id()

	log.Printf("[DEBUG] Deleting ACL: %v", name)

	input := compute.DeleteACLInput{
		Name: name,
	}
	if err := client.DeleteACL(&input); err != nil {
		return fmt.Errorf("Error deleting ACL")
	}
	return nil
}
