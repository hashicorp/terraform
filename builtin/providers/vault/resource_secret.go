package vault

import (
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/vault/api"
)

func resourceVaultSecret() *schema.Resource {
	return &schema.Resource{
		Create: resourceVaultSecretCreate,
		// Yay for PUT
		Update: resourceVaultSecretCreate,
		Read:   resourceVaultSecretRead,
		Delete: resourceVaultSecretDelete,

		Schema: map[string]*schema.Schema{
			"path": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ttl": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"data": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
		},
	}
}

func resourceVaultSecretCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	data := d.Get("data").(map[string]interface{})
	if ttl := d.Get("ttl").(string); ttl != "" {
		data["ttl"] = ttl
	}

	_, err := client.Logical().Write(d.Get("path").(string), data)
	if err != nil {
		return err
	}

	d.SetId(d.Get("path").(string))
	return nil
}

func resourceVaultSecretRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	secret, err := client.Logical().Read(d.Get("path").(string))
	if err != nil {
		return err
	}
	if secret == nil {
		log.Printf("[WARN] %q seems to be gone, removing from state.", d.Id())
		d.SetId("")
	}

	if ttl, ok := secret.Data["ttl"]; ok {
		d.Set("ttl", ttl.(string))
	}

	delete(secret.Data, "ttl")

	if err := d.Set("data", secret.Data); err != nil {
		return err
	}

	return nil
}

func isSecretNotFoundError(err error) bool {
	return strings.Contains(err.Error(), "bad token")
}

func resourceVaultSecretDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	_, err := client.Logical().Delete(d.Get("path").(string))
	if err != nil {
		return err
	}

	return nil
}
