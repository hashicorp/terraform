package vault

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/vault/api"
)

func resourceVaultSecretBackend() *schema.Resource {
	return &schema.Resource{
		Create: resourceVaultSecretBackendCreate,
		Read:   resourceVaultSecretBackendRead,
		Update: resourceVaultSecretBackendUpdate,
		Delete: resourceVaultSecretBackendDelete,

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
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "Managed by Terraform",
			},

			"default_lease_ttl": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: ValidateDurationString,
				StateFunc:    NormalizeDurationString,
			},

			"max_lease_ttl": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: ValidateDurationString,
				StateFunc:    NormalizeDurationString,
			},
		},
	}
}

func resourceVaultSecretBackendCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	path := d.Get("path").(string)
	// Mimic the behavior of the Vault CLI by defaulting the path to the type.
	if path == "" {
		path = d.Get("type").(string)
	}
	input := &api.MountInput{
		Type:        d.Get("type").(string),
		Description: d.Get("description").(string),
		Config: api.MountConfigInput{
			DefaultLeaseTTL: d.Get("default_lease_ttl").(string),
			MaxLeaseTTL:     d.Get("max_lease_ttl").(string),
		},
	}
	err := client.Sys().Mount(path, input)
	if err != nil {
		return fmt.Errorf("Error creating secret backend %q: %s", path, err)
	}

	d.SetId(path)
	d.Set("path", path)

	return resourceVaultSecretBackendRead(d, meta)
}

func resourceVaultSecretBackendRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	mounts, err := client.Sys().ListMounts()
	if err != nil {
		return fmt.Errorf("Error listing mounts: %s", err)
	}

	var foundMount *api.MountOutput
	for mountPoint, mount := range mounts {
		if mountPoint[:len(mountPoint)-1] == d.Id() {
			foundMount = mount
		}
	}

	if foundMount == nil {
		log.Printf("[WARN] Secret backend not found; removing from state: %s", d.Id())
		d.SetId("")
		return nil
	}

	config, err := client.Sys().MountConfig(d.Id())
	if err != nil {
		return fmt.Errorf("Error checking mount config for %q: %s", d.Id(), err)
	}

	defaultTTL := time.Duration(config.DefaultLeaseTTL) * time.Second
	maxTTL := time.Duration(config.MaxLeaseTTL) * time.Second

	d.Set("default_lease_ttl", defaultTTL.String())
	d.Set("max_lease_ttl", maxTTL.String())

	return nil
}

func resourceVaultSecretBackendUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	if d.HasChange("default_lease_ttl") || d.HasChange("max_lease_ttl") {
		err := client.Sys().TuneMount(d.Id(), api.MountConfigInput{
			DefaultLeaseTTL: d.Get("default_lease_ttl").(string),
			MaxLeaseTTL:     d.Get("max_lease_ttl").(string),
		})
		if err != nil {
			return fmt.Errorf("Error while tuning secret backend %q: %s", d.Id(), err)
		}
	}

	if d.HasChange("path") {
		oldPath, newPath := d.GetChange("path")
		err := client.Sys().Remount(oldPath.(string), newPath.(string))
		if err != nil {
			return fmt.Errorf("Error while remounting %q: %s", d.Id(), err)
		}
		d.SetId(newPath.(string))
	}

	return nil
}

func resourceVaultSecretBackendDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	path := d.Id()
	err := client.Sys().Unmount(path)
	if err != nil {
		return fmt.Errorf("Error deleting secret backend %q: %s", d.Id(), err)
	}
	return nil
}
