package vault

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/vault/api"
)

func resourceVaultAuthBackend() *schema.Resource {
	return &schema.Resource{
		Create: resourceVaultAuthBackendCreate,
		Read:   resourceVaultAuthBackendRead,
		Delete: resourceVaultAuthBackendDelete,

		Schema: map[string]*schema.Schema{
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "Managed by Terraform",
			},
		},
	}
}

func resourceVaultAuthBackendCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	path := d.Get("path").(string)
	// Mimic the behavior of the Vault CLI by defaulting the path to the type.
	if path == "" {
		path = d.Get("type").(string)
	}
	err := client.Sys().EnableAuth(
		path,
		d.Get("type").(string),
		d.Get("description").(string),
	)
	if err != nil {
		return fmt.Errorf("Error creating mount: %s", err)
	}

	d.SetId(path)
	d.Set("path", path)

	return resourceVaultAuthBackendRead(d, meta)
}

func resourceVaultAuthBackendRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	auths, err := client.Sys().ListAuth()
	if err != nil {
		return fmt.Errorf("Error listing mounts: %s", err)
	}

	var foundAuthBackend *api.AuthMount
	for mountPoint, auth := range auths {
		if mountPoint[:len(mountPoint)-1] == d.Id() {
			foundAuthBackend = auth
		}
	}

	if foundAuthBackend == nil {
		log.Printf("[WARN] Auth backend not found; removing from state: %s", d.Id())
		d.SetId("")
	}

	return nil
}

func resourceVaultAuthBackendDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	path := d.Id()
	err := client.Sys().DisableAuth(path)
	if err != nil {
		return fmt.Errorf("Error disabling auth backend: %s", err)
	}
	return nil
}
