package digitalocean

import (
	"fmt"
	"log"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDigitalOceanDomain() *schema.Resource {
	return &schema.Resource{
		Create: resourceDigitalOceanDomainCreate,
		Read:   resourceDigitalOceanDomainRead,
		Delete: resourceDigitalOceanDomainDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceDigitalOceanDomainCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	// Build up our creation options

	opts := &godo.DomainCreateRequest{
		Name:      d.Get("name").(string),
		IPAddress: d.Get("ip_address").(string),
	}

	log.Printf("[DEBUG] Domain create configuration: %#v", opts)
	domain, _, err := client.Domains.Create(opts)
	if err != nil {
		return fmt.Errorf("Error creating Domain: %s", err)
	}

	d.SetId(domain.Name)
	log.Printf("[INFO] Domain Name: %s", domain.Name)

	return resourceDigitalOceanDomainRead(d, meta)
}

func resourceDigitalOceanDomainRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	domain, resp, err := client.Domains.Get(d.Id())
	if err != nil {
		// If the domain is somehow already destroyed, mark as
		// successfully gone
		if resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving domain: %s", err)
	}

	d.Set("name", domain.Name)

	return nil
}

func resourceDigitalOceanDomainDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	log.Printf("[INFO] Deleting Domain: %s", d.Id())
	_, err := client.Domains.Delete(d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting Domain: %s", err)
	}

	d.SetId("")
	return nil
}
