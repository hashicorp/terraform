package digitalocean

import (
	"context"
	"fmt"
	"log"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDigitalOceanTag() *schema.Resource {
	return &schema.Resource{
		Create: resourceDigitalOceanTagCreate,
		Read:   resourceDigitalOceanTagRead,
		Delete: resourceDigitalOceanTagDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceDigitalOceanTagCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	// Build up our creation options
	opts := &godo.TagCreateRequest{
		Name: d.Get("name").(string),
	}

	log.Printf("[DEBUG] Tag create configuration: %#v", opts)
	tag, _, err := client.Tags.Create(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("Error creating tag: %s", err)
	}

	d.SetId(tag.Name)
	log.Printf("[INFO] Tag: %s", tag.Name)

	return resourceDigitalOceanTagRead(d, meta)
}

func resourceDigitalOceanTagRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	tag, resp, err := client.Tags.Get(context.Background(), d.Id())
	if err != nil {
		// If the tag is somehow already destroyed, mark as
		// successfully gone
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving tag: %s", err)
	}

	d.Set("name", tag.Name)

	return nil
}

func resourceDigitalOceanTagDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	log.Printf("[INFO] Deleting tag: %s", d.Id())
	_, err := client.Tags.Delete(context.Background(), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting tag: %s", err)
	}

	d.SetId("")
	return nil
}
