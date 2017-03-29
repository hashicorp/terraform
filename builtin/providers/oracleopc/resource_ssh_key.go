package opc

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/oracle/terraform-provider-compute/sdk/compute"
	"log"
)

func resourceSSHKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceSSHKeyCreate,
		Read:   resourceSSHKeyRead,
		Update: resourceSSHKeyUpdate,
		Delete: resourceSSHKeyDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
				ForceNew: false,
			},
		},
	}
}

func resourceSSHKeyCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource data: %#v", d)

	client := meta.(*OPCClient).SSHKeys()
	name := d.Get("name").(string)
	key := d.Get("key").(string)
	enabled := d.Get("enabled").(bool)

	log.Printf("[DEBUG] Creating ssh key with name %s, key %s, enabled %s",
		name, key, enabled)

	info, err := client.CreateSSHKey(name, key, enabled)
	if err != nil {
		return fmt.Errorf("Error creating ssh key %s: %s", name, err)
	}

	d.SetId(info.Name)
	updateSSHKeyResourceData(d, info)
	return nil
}

func updateSSHKeyResourceData(d *schema.ResourceData, info *compute.SSHKeyInfo) {
	d.Set("name", info.Name)
	d.Set("key", info.Key)
	d.Set("enabled", info.Enabled)
}

func resourceSSHKeyRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource data: %#v", d)
	client := meta.(*OPCClient).SSHKeys()
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Reading state of ssh key %s", name)
	result, err := client.GetSSHKey(name)
	if err != nil {
		// SSH Key does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading ssh key %s: %s", name, err)
	}

	log.Printf("[DEBUG] Read state of ssh key %s: %#v", name, result)
	updateSSHKeyResourceData(d, result)
	return nil
}

func resourceSSHKeyUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource data: %#v", d)

	client := meta.(*OPCClient).SSHKeys()
	name := d.Get("name").(string)
	key := d.Get("key").(string)
	enabled := d.Get("enabled").(bool)

	log.Printf("[DEBUG] Updating ssh key with name %s, key %s, enabled %s",
		name, key, enabled)

	info, err := client.UpdateSSHKey(name, key, enabled)
	if err != nil {
		return fmt.Errorf("Error updating ssh key %s: %s", name, err)
	}

	updateSSHKeyResourceData(d, info)
	return nil
}

func resourceSSHKeyDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource data: %#v", d)
	client := meta.(*OPCClient).SSHKeys()
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Deleting ssh key volume %s", name)
	if err := client.DeleteSSHKey(name); err != nil {
		return fmt.Errorf("Error deleting ssh key %s: %s", name, err)
	}
	return nil
}
