package vault

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/vault/api"
	"log"
)

func mountResource() *schema.Resource {
	return &schema.Resource{
		Create: mountWrite,
		Update: mountUpdate,
		Delete: mountDelete,
		Read:   mountRead,

		Schema: map[string]*schema.Schema{
			"path": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    false,
				Description: "Where the secret backend will be mounted",
			},

			"type": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Type of the backend, such as 'aws'",
			},

			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Required:    false,
				ForceNew:    true,
				Description: "Human-friendly description of the mount",
			},

			"default_lease_ttl_seconds": {
				Type:        schema.TypeInt,
				Required:    false,
				Optional:    true,
				ForceNew:    false,
				Description: "Default lease duration for tokens and secrets in seconds",
			},

			"max_lease_ttl_seconds": {
				Type:        schema.TypeInt,
				Required:    false,
				Optional:    true,
				ForceNew:    false,
				Description: "Maximum possible lease duration for tokens and secrets in seconds",
			},
		},
	}
}

func mountWrite(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	info := &api.MountInput{
		Type:        d.Get("type").(string),
		Description: d.Get("description").(string),
		Config: api.MountConfigInput{
			DefaultLeaseTTL: fmt.Sprintf("%ds", d.Get("default_lease_ttl_seconds")),
			MaxLeaseTTL:     fmt.Sprintf("%ds", d.Get("max_lease_ttl_seconds")),
		},
	}

	path := d.Get("path").(string)

	log.Printf("[DEBUG] Creating mount %s in Vault", path)

	if err := client.Sys().Mount(path, info); err != nil {
		return fmt.Errorf("error writing to Vault: %s", err)
	}

	d.SetId(path)

	return nil
}

func mountUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	config := api.MountConfigInput{
		DefaultLeaseTTL: fmt.Sprintf("%ds", d.Get("default_lease_ttl_seconds")),
		MaxLeaseTTL:     fmt.Sprintf("%ds", d.Get("max_lease_ttl_seconds")),
	}

	path := d.Id()

	if d.HasChange("path") {
		newPath := d.Get("path").(string)

		log.Printf("[DEBUG] Remount %s to %s in Vault", path, newPath)

		err := client.Sys().Remount(d.Id(), newPath)
		if err != nil {
			return fmt.Errorf("error remounting in Vault: %s", err)
		}

		d.SetId(newPath)
		path = newPath
	}

	log.Printf("[DEBUG] Updating mount %s in Vault", path)

	if err := client.Sys().TuneMount(path, config); err != nil {
		return fmt.Errorf("error updating Vault: %s", err)
	}

	return nil
}

func mountDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	path := d.Id()

	log.Printf("[DEBUG] Unmounting %s from Vault", path)

	if err := client.Sys().Unmount(path); err != nil {
		return fmt.Errorf("error deleting from Vault: %s", err)
	}

	return nil
}

func mountRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	path := d.Id()

	log.Printf("[DEBUG] Reading mount %s from Vault", path)

	mount, err := client.Sys().MountConfig(path)
	if err != nil {
		return fmt.Errorf("error reading from Vault: %s", err)
	}

	d.Set("default_lease_ttl_seconds", mount.DefaultLeaseTTL)
	d.Set("max_lease_ttl_seconds", mount.MaxLeaseTTL)

	return nil
}
