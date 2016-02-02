package vault

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/vault/api"
)

func resourceVaultAuditBackend() *schema.Resource {
	return &schema.Resource{
		Create: resourceVaultAuditBackendCreate,
		Read:   resourceVaultAuditBackendRead,
		Delete: resourceVaultAuditBackendDelete,

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

			"options": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceVaultAuditBackendCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	path := d.Get("path").(string)
	if path == "" {
		path = d.Get("type").(string)
	}
	options := map[string]string{}
	for k, v := range d.Get("options").(map[string]interface{}) {
		options[k] = v.(string)
	}
	err := client.Sys().EnableAudit(
		path,
		d.Get("type").(string),
		d.Get("description").(string),
		options,
	)
	if err != nil {
		return fmt.Errorf("Error creating mount: %s", err)
	}

	d.SetId(path)
	d.Set("path", path)

	return resourceVaultAuditBackendRead(d, meta)
}

func resourceVaultAuditBackendRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	audits, err := client.Sys().ListAudit()
	if err != nil {
		return fmt.Errorf("Error listing mounts: %s", err)
	}

	var foundAuditBackend *api.Audit
	for mountPoint, audit := range audits {
		if mountPoint[:len(mountPoint)-1] == d.Id() {
			foundAuditBackend = audit
		}
	}

	if foundAuditBackend == nil {
		log.Printf("[WARN] Audit backend not found; removing from state: %s", d.Id())
		d.SetId("")
	}

	return nil
}

func resourceVaultAuditBackendDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	path := d.Id()
	err := client.Sys().DisableAudit(path)
	if err != nil {
		return fmt.Errorf("Error disabling audit backend: %s", err)
	}
	return nil
}
