package packet

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/packethost/packngo"
)

func resourcePacketSSHKey() *schema.Resource {
	return &schema.Resource{
		Create: resourcePacketSSHKeyCreate,
		Read:   resourcePacketSSHKeyRead,
		Update: resourcePacketSSHKeyUpdate,
		Delete: resourcePacketSSHKeyDelete,

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

			"created": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"updated": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourcePacketSSHKeyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	createRequest := &packngo.SSHKeyCreateRequest{
		Label: d.Get("name").(string),
		Key:   d.Get("public_key").(string),
	}

	log.Printf("[DEBUG] SSH Key create configuration: %#v", createRequest)
	key, _, err := client.SSHKeys.Create(createRequest)
	if err != nil {
		return fmt.Errorf("Error creating SSH Key: %s", err)
	}

	d.SetId(key.ID)
	log.Printf("[INFO] SSH Key: %s", key.ID)

	return resourcePacketSSHKeyRead(d, meta)
}

func resourcePacketSSHKeyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	key, _, err := client.SSHKeys.Get(d.Id())
	if err != nil {
		// If the key is somehow already destroyed, mark as
		// succesfully gone
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving SSH key: %s", err)
	}

	d.Set("id", key.ID)
	d.Set("name", key.Label)
	d.Set("public_key", key.Key)
	d.Set("fingerprint", key.FingerPrint)
	d.Set("created", key.Created)
	d.Set("updated", key.Updated)

	return nil
}

func resourcePacketSSHKeyUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	updateRequest := &packngo.SSHKeyUpdateRequest{
		ID:    d.Get("id").(string),
		Label: d.Get("name").(string),
		Key:   d.Get("public_key").(string),
	}

	log.Printf("[DEBUG] SSH key update: %#v", d.Get("id"))
	_, _, err := client.SSHKeys.Update(updateRequest)
	if err != nil {
		return fmt.Errorf("Failed to update SSH key: %s", err)
	}

	return resourcePacketSSHKeyRead(d, meta)
}

func resourcePacketSSHKeyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	log.Printf("[INFO] Deleting SSH key: %s", d.Id())
	_, err := client.SSHKeys.Delete(d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting SSH key: %s", err)
	}

	d.SetId("")
	return nil
}
