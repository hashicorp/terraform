package opc

import (
	"fmt"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceOPCSSHKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceOPCSSHKeyCreate,
		Read:   resourceOPCSSHKeyRead,
		Update: resourceOPCSSHKeyUpdate,
		Delete: resourceOPCSSHKeyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"key": {
				Type:     schema.TypeString,
				Required: true,
			},

			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func resourceOPCSSHKeyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).SSHKeys()

	name := d.Get("name").(string)
	key := d.Get("key").(string)
	enabled := d.Get("enabled").(bool)

	input := compute.CreateSSHKeyInput{
		Name:    name,
		Key:     key,
		Enabled: enabled,
	}
	info, err := client.CreateSSHKey(&input)
	if err != nil {
		return fmt.Errorf("Error creating ssh key %s: %s", name, err)
	}

	d.SetId(info.Name)

	return resourceOPCSSHKeyRead(d, meta)
}

func resourceOPCSSHKeyUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).SSHKeys()

	name := d.Get("name").(string)
	key := d.Get("key").(string)
	enabled := d.Get("enabled").(bool)

	input := compute.UpdateSSHKeyInput{
		Name:    name,
		Key:     key,
		Enabled: enabled,
	}
	_, err := client.UpdateSSHKey(&input)
	if err != nil {
		return fmt.Errorf("Error updating ssh key %s: %s", name, err)
	}

	return resourceOPCSSHKeyRead(d, meta)
}

func resourceOPCSSHKeyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).SSHKeys()
	name := d.Id()

	input := compute.GetSSHKeyInput{
		Name: name,
	}
	result, err := client.GetSSHKey(&input)
	if err != nil {
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading ssh key %s: %s", name, err)
	}

	d.Set("name", result.Name)
	d.Set("key", result.Key)
	d.Set("enabled", result.Enabled)

	return nil
}

func resourceOPCSSHKeyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).SSHKeys()
	name := d.Id()

	input := compute.DeleteSSHKeyInput{
		Name: name,
	}
	if err := client.DeleteSSHKey(&input); err != nil {
		return fmt.Errorf("Error deleting ssh key %s: %s", name, err)
	}

	return nil
}
