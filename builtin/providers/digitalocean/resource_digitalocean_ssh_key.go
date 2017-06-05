package digitalocean

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDigitalOceanSSHKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceDigitalOceanSSHKeyCreate,
		Read:   resourceDigitalOceanSSHKeyRead,
		Update: resourceDigitalOceanSSHKeyUpdate,
		Delete: resourceDigitalOceanSSHKeyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"public_key": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: resourceDigitalOceanSSHKeyPublicKeyDiffSuppress,
			},

			"fingerprint": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceDigitalOceanSSHKeyPublicKeyDiffSuppress(k, old, new string, d *schema.ResourceData) bool {
	return strings.TrimSpace(old) == strings.TrimSpace(new)
}

func resourceDigitalOceanSSHKeyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	// Build up our creation options
	opts := &godo.KeyCreateRequest{
		Name:      d.Get("name").(string),
		PublicKey: d.Get("public_key").(string),
	}

	log.Printf("[DEBUG] SSH Key create configuration: %#v", opts)
	key, _, err := client.Keys.Create(context.Background(), opts)
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

	key, resp, err := client.Keys.GetByID(context.Background(), id)
	if err != nil {
		// If the key is somehow already destroyed, mark as
		// successfully gone
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving SSH key: %s", err)
	}

	d.Set("name", key.Name)
	d.Set("fingerprint", key.Fingerprint)
	d.Set("public_key", key.PublicKey)

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
	_, _, err = client.Keys.UpdateByID(context.Background(), id, opts)
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
	_, err = client.Keys.DeleteByID(context.Background(), id)
	if err != nil {
		return fmt.Errorf("Error deleting SSH key: %s", err)
	}

	d.SetId("")
	return nil
}
