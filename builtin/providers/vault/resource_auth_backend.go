package vault

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/vault/api"
)

func authBackendResource() *schema.Resource {
	return &schema.Resource{
		Create: authBackendWrite,
		Delete: authBackendDelete,
		Read:   authBackendRead,

		Schema: map[string]*schema.Schema{
			"type": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the auth backend",
			},

			"path": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "path to mount the backend. This defaults to the type.",
				ValidateFunc: func(v interface{}, k string) (ws []string, errs []error) {
					value := v.(string)
					if strings.HasSuffix(value, "/") {
						errs = append(errs, errors.New("cannot write to a path ending in '/'"))
					}
					return
				},
			},

			"description": &schema.Schema{
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Description: "The description of the auth backend",
			},
		},
	}
}

func authBackendWrite(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	name := d.Get("type").(string)
	desc := d.Get("description").(string)
	path := d.Get("path").(string)

	log.Printf("[DEBUG] Writing auth %s to Vault", name)

	var err error

	if path == "" {
		path = name
		err = d.Set("path", name)
		if err != nil {
			return fmt.Errorf("unable to set state: %s", err)
		}
	}

	err = client.Sys().EnableAuth(path, name, desc)

	if err != nil {
		return fmt.Errorf("error writing to Vault: %s", err)
	}

	d.SetId(name)

	return nil
}

func authBackendDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	name := d.Id()

	log.Printf("[DEBUG] Deleting auth %s from Vault", name)

	err := client.Sys().DisableAuth(name)

	if err != nil {
		return fmt.Errorf("error disabling auth from Vault: %s", err)
	}

	return nil
}

func authBackendRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	name := d.Id()

	auths, err := client.Sys().ListAuth()

	if err != nil {
		return fmt.Errorf("error reading from Vault: %s", err)
	}

	for path, auth := range auths {
		configuredPath := d.Get("path").(string)

		vaultPath := configuredPath + "/"
		if auth.Type == name && path == vaultPath {
			return nil
		}
	}

	// If we fell out here then we didn't find our Auth in the list.
	d.SetId("")
	return nil
}
