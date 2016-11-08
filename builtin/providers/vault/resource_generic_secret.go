package vault

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/hashicorp/vault/api"
)

func genericSecretResource() *schema.Resource {
	return &schema.Resource{
		Create: genericSecretResourceWrite,
		Update: genericSecretResourceWrite,
		Delete: genericSecretResourceDelete,
		Read:   genericSecretResourceRead,

		Schema: map[string]*schema.Schema{
			"path": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Full path where the generic secret will be written.",
			},

			// Data is passed as JSON so that an arbitrary structure is
			// possible, rather than forcing e.g. all values to be strings.
			"data_json": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "JSON-encoded secret data to write.",
			},
		},
	}
}

func genericSecretResourceWrite(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	path := d.Get("path").(string)

	var data map[string]interface{}
	err := json.Unmarshal([]byte(d.Get("data_json").(string)), &data)
	if err != nil {
		return fmt.Errorf("data_json %#v syntax error: %s", d.Get("data_json"), err)
	}

	log.Printf("[DEBUG] Writing generic Vault secret to %s", path)
	_, err = client.Logical().Write(path, data)
	if err != nil {
		return fmt.Errorf("error writing to Vault: %s", err)
	}

	d.SetId(path)

	return nil
}

func genericSecretResourceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	path := d.Id()

	log.Printf("[DEBUG] Deleting generic Vault from %s", path)
	_, err := client.Logical().Delete(path)
	if err != nil {
		return fmt.Errorf("error deleting from Vault: %s", err)
	}

	return nil
}

func genericSecretResourceRead(d *schema.ResourceData, meta interface{}) error {
	// We don't actually attempt to read back the secret data
	// here, so that Terraform can be configured with a token
	// that has only write access to the relevant part of the
	// store.
	//
	// This means that Terraform cannot detect drift for
	// generic secrets, but detecting drift seems less important
	// than being able to limit the effect of exposure of
	// Terraform's Vault token.
	log.Printf("[WARN] vault_generic_secret does not automatically refresh")
	return nil
}
