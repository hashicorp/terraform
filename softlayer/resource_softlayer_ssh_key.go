package softlayer

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

func resourceSoftLayerSSHKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceSoftLayerSSHKeyCreate,
		Read:   resourceSoftLayerSSHKeyRead,
		Update: resourceSoftLayerSSHKeyUpdate,
		Delete: resourceSoftLayerSSHKeyDelete,
		Exists: resourceSoftLayerSSHKeyExists,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeInt,
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

			"notes": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  nil,
			},
		},
	}
}

func resourceSoftLayerSSHKeyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).sshKeyService

	// Build up our creation options
	opts := datatypes.SoftLayer_Security_Ssh_Key{
		Label: d.Get("name").(string),
		Key:   d.Get("public_key").(string),
	}

	if notes, ok := d.GetOk("notes"); ok {
		opts.Notes = notes.(string)
	}

	res, err := client.CreateObject(opts)
	if err != nil {
		return fmt.Errorf("Error creating SSH Key: %s", err)
	}

	d.SetId(strconv.Itoa(res.Id))
	log.Printf("[INFO] SSH Key: %d", res.Id)

	return resourceSoftLayerSSHKeyRead(d, meta)
}

func resourceSoftLayerSSHKeyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).sshKeyService

	keyId, _ := strconv.Atoi(d.Id())

	key, err := client.GetObject(keyId)
	if err != nil {
		// If the key is somehow already destroyed, mark as
		// succesfully gone
		if strings.Contains(err.Error(), "404 Not Found") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving SSH key: %s", err)
	}

	d.Set("id", key.Id)
	d.Set("name", key.Label)
	d.Set("public_key", strings.TrimSpace(key.Key))
	d.Set("fingerprint", key.Fingerprint)
	d.Set("notes", key.Notes)

	return nil
}

func resourceSoftLayerSSHKeyUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).sshKeyService

	keyId, _ := strconv.Atoi(d.Id())

	key, err := client.GetObject(keyId)
	if err != nil {
		return fmt.Errorf("Error retrieving SSH key: %s", err)
	}

	if d.HasChange("name") {
		key.Label = d.Get("name").(string)
	}

	if d.HasChange("notes") {
		key.Notes = d.Get("notes").(string)
	}

	_, err = client.EditObject(keyId, key)
	if err != nil {
		return fmt.Errorf("Error editing SSH key: %s", err)
	}
	return nil
}

func resourceSoftLayerSSHKeyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).sshKeyService

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting SSH Key: %s", err)
	}

	log.Printf("[INFO] Deleting SSH key: %d", id)
	_, err = client.DeleteObject(id)
	if err != nil {
		return fmt.Errorf("Error deleting SSH key: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceSoftLayerSSHKeyExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*Client).sshKeyService

	if client == nil {
		return false, fmt.Errorf("The client was nil.")
	}

	keyId, err := strconv.Atoi(d.Id())
	if err != nil {
		return false, fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}

	result, err := client.GetObject(keyId)
	return result.Id == keyId && err == nil, nil
}
