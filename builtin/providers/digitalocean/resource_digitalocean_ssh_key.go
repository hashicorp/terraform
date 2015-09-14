package digitalocean

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pearkes/digitalocean"
)

func resourceDigitalOceanSSHKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceDigitalOceanSSHKeyCreate,
		Read:   resourceDigitalOceanSSHKeyRead,
		Update: resourceDigitalOceanSSHKeyUpdate,
		Delete: resourceDigitalOceanSSHKeyDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"public_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceDigitalOceanSSHKeyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*digitalocean.Client)

	// Build up our creation options
	opts := &digitalocean.CreateSSHKey{
		Name:      d.Get("name").(string),
		PublicKey: d.Get("public_key").(string),
	}

	log.Printf("[DEBUG] SSH Key create configuration: %#v", opts)
	id, err := client.CreateSSHKey(opts)
	if err != nil {
		return fmt.Errorf("Error creating SSH Key: %s", err)
	}

	d.SetId(id)
	log.Printf("[INFO] SSH Key: %s", id)

	return resourceDigitalOceanSSHKeyRead(d, meta)
}

func resourceDigitalOceanSSHKeyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*digitalocean.Client)

	key, err := client.RetrieveSSHKey(d.Id())
	if err != nil {
		// If the key is somehow already destroyed, mark as
		// successfully gone
		if strings.Contains(err.Error(), "404 Not Found") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving SSH key: %s", err)
	}

	d.Set("name", key.Name)
	d.Set("fingerprint", key.Fingerprint)

	return nil
}

func resourceDigitalOceanSSHKeyUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*digitalocean.Client)

	var newName string
	if v, ok := d.GetOk("name"); ok {
		newName = v.(string)
	}

	log.Printf("[DEBUG] SSH key update name: %#v", newName)
	err := client.RenameSSHKey(d.Id(), newName)
	if err != nil {
		return fmt.Errorf("Failed to update SSH key: %s", err)
	}

	return resourceDigitalOceanSSHKeyRead(d, meta)
}

func resourceDigitalOceanSSHKeyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*digitalocean.Client)

	log.Printf("[INFO] Deleting SSH key: %s", d.Id())
	err := client.DestroySSHKey(d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting SSH key: %s", err)
	}

	d.SetId("")
	return nil
}
