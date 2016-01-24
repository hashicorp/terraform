package digitalocean

import (
	"fmt"
	"log"
	"strconv"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/schema"
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
	client := meta.(*godo.Client)

	// Build up our creation options
	opts := &godo.KeyCreateRequest{
		Name:      d.Get("name").(string),
		PublicKey: d.Get("public_key").(string),
	}

	log.Printf("[DEBUG] SSH Key create configuration: %#v", opts)
	key, _, err := client.Keys.Create(opts)
	if err != nil {
		return fmt.Errorf("Error creating SSH Key: %s", err)
	}

	d.SetId(strconv.Itoa(key.ID))
	log.Printf("[INFO] SSH Key: %d", key.ID)

	return resourceDigitalOceanSSHKeyRead(d, meta)
}

func resourceDigitalOceanSSHKeyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("invalid SSH key id: %v", err)
	}

	key, resp, err := client.Keys.GetByID(id)
	if err != nil {
		// If the key is somehow already destroyed, mark as
		// successfully gone
		if resp.StatusCode == 404 {
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
	client := meta.(*godo.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("invalid SSH key id: %v", err)
	}

	var newName string
	if v, ok := d.GetOk("name"); ok {
		newName = v.(string)
	}

	log.Printf("[DEBUG] SSH key update name: %#v", newName)
	opts := &godo.KeyUpdateRequest{
		Name: newName,
	}
	_, _, err = client.Keys.UpdateByID(id, opts)
	if err != nil {
		return fmt.Errorf("Failed to update SSH key: %s", err)
	}

	return resourceDigitalOceanSSHKeyRead(d, meta)
}

func resourceDigitalOceanSSHKeyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("invalid SSH key id: %v", err)
	}

	log.Printf("[INFO] Deleting SSH key: %d", id)
	_, err = client.Keys.DeleteByID(id)
	if err != nil {
		return fmt.Errorf("Error deleting SSH key: %s", err)
	}

	d.SetId("")
	return nil
}
